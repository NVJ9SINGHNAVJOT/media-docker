# System Design

Design rationale, contracts, capacity model and failure semantics. Complements
[architecture.md](./architecture.md) (what the pieces are) and [project-flow.md](./project-flow.md)
(how a request moves through them).

---

## 1. Design goals

| Goal | How it is met |
|---|---|
| A URL that works the moment an upload finishes | Uploads are written into the served tree; a resolver picks the best representation per request |
| Never block the HTTP request on ffmpeg | Server only writes bytes and produces to Kafka |
| Independent scaling per media type | One consumer service per topic, each with its own worker count |
| A failed conversion must not break an asset | The raw upload is kept unless conversion succeeds |
| Survive worker and process crashes | Consumer groups rebalance; manual commit; idempotent handlers |
| Self-hosted, no third-party storage | Docker volumes plus a static file server |

The system is still **not** trying to be multi-tenant, per-user authenticated, geo-distributed, or
strongly consistent. One shared secret, one filesystem, one region.

---

## 2. The core contract: URLs that upgrade

v3 returned a URL for a file that did not exist yet, so every integrating backend had to consume a
Kafka response topic and hold assets in a pending state. v4 inverts that:

```
POST /api/v1/uploads/video     →  201 { id, fileUrl }   ← already works
GET  {fileUrl}                 →  302 → original.mp4    ← raw upload, playable now
      ⋮ (consumer finishes, promotes atomically)
GET  {fileUrl}                 →  302 → hls/index.m3u8  ← same URL, better representation
```

There is no completion message at all. The URL is the entire contract: it is usable from the moment
the upload returns, and if conversion never succeeds it permanently keeps serving the raw upload. A
backend that wants to distinguish the two can read the resolver's cache header — `no-store` while
raw, `max-age=300` once converted — but nothing requires it to.

### Why a redirect

Three options were considered for the upgrade mechanism:

- **302 to the current representation** — chosen. The redirect lands *inside* the asset directory, so
  relative references in an HLS playlist (`segment000.ts` next to `index.m3u8`) resolve correctly.
- **Serving bytes from the resolver URL** — would require rewriting every playlist to use absolute
  segment URLs, plus Range handling for the raw file.
- **Two URLs (raw + converted)** — leaves the backend implementing the state machine v4 exists to
  remove.

Cache headers follow the representation: `no-store` while unconverted, since it can change at any
moment; `max-age=300` once converted, since that state is terminal.

### Why promotion must be atomic

ffmpeg builds an HLS playlist incrementally. Even with `-hls_playlist_type vod`, `index.m3u8` exists
and is incomplete for the entire transcode. A resolver keyed on "does index.m3u8 exist" would hand
clients a truncated playlist.

So consumers write into `<id>/.processing/` and promote with a single `os.Rename`
([pkg/asset/promote.go](../pkg/asset/promote.go)). Converted output is either absent or complete —
there is no third state. `.processing/` is never reachable through the resolver, which
[a test](../pkg/asset/resolve_test.go) pins explicitly.

Re-processing moves the existing output aside, renames the new output in, then deletes the old, so
the window in which neither is in place is one rename long.

---

## 3. Storage and upload design

**One id per asset**, minted when the upload starts and used for the directory, the Kafka messages
and the public URL. v3's separate upload handle and output UUID are gone.

**Uploads are merged straight into media storage.** Assembling a chunked file is one full pass over
the data regardless of destination, so that pass writes to the final location — avoiding a second
full copy of every upload and any cross-volume move. Chunk staging stays on the private upload
volume; only the merged result reaches media storage.

The cost of that choice is uploads that are stored but never claimed. `meta.json` carries a
`dispatched` flag, and [api/janitor.go](../api/janitor.go) sweeps hourly for undispatched assets and
orphaned staging directories older than 6 hours. This also fixes a pre-existing v3 leak: abandoned
chunk directories previously accumulated with nothing to clean them up.

**Two upload modes:**

- **Single-shot** (`/file-storage`), for files ≤ 2 MB, written directly into the asset directory.
- **Chunked** (`/chunks-storage`), a three-state protocol (`start` → `uploading` → `completed`).
  `start` mints the id and creates the staging directory; `completed` checks total size, merges, and
  writes `meta.json`.

The 2 MB chunk cap (`helper.Constants.MaxChunkSize`) is a hard contract with the SDK's `CHUNK_SIZE`;
both sides must change together. Total size is enforced at `completed` by walking the staging
directory.

**Extension handling** takes the extension from the client's file name (sanitised to
`[A-Za-z0-9]{1,8}`), falling back to the MIME subtype and then a per-category default. Deriving it
from the MIME subtype alone — as v3 did — produces
`vnd.openxmlformats-officedocument.wordprocessingml.document` for a `.docx`.

---

## 4. Processing design

| Job | ffmpeg | Output |
|---|---|---|
| video | `libx264` + `aac`, HLS 10s segments, VOD | `hls/index.m3u8` + segments |
| video-resolutions | same, plus `-vf scale=W:H`, once per rung | `hls/{360,480,720,1080}/` + master playlist |
| image | `-q:v <compression>` | `converted.jpeg` |
| audio | `-vn -ar 44100 -ac 2 [-b:a bitrate]` | `converted.mp3` |

Quality knobs: video `quality` 40–100 maps to bitrates `500+(q-40)*15` kbps video and `64+(q-40)*2`
kbps audio; audio `bitrate` ∈ {128k, 192k, 256k, 320k}; image `compression` 1–31 (ffmpeg `-q:v`),
defaulting to 1. Image compression was hardcoded in v3 despite the README advertising it as
configurable; it is now plumbed through `ImageMessage`.

**The resolution ladder** lives in `asset.Ladder`: 360 (640×360), 480 (854×480), 720 (1280×720),
1080 (1920×1080), each with an advertised bandwidth. v3 scaled 360p to 640**740**×360, a non-16:9
aspect that distorted the picture; the ladder now supplies both the ffmpeg scale filter and the
master playlist from one definition.

**The master playlist** is new. Without it a video-resolutions asset has no top-level playlist for
its base URL to resolve to. It also delivers the adaptive quality switching the project has always
advertised: players read it and select a variant themselves. Only rungs that were actually produced
are advertised, so a partial ladder still yields a valid playlist.

`runCommand` discards ffmpeg output in production; the streaming variant is commented out in
[pkg/ffmpegCommand.go](../pkg/ffmpegCommand.go).

---

## 5. Concurrency and capacity

```
per service:  1 topic  ×  1 consumer group  ×  KAFKA_WORKERS goroutines
```

**The partition ceiling.** Partition counts (video 100, video-resolutions 100, image 100, audio 50,
delete-file 20, DLQ 10) bound useful parallelism per topic across all instances of that
service. Excess workers hold a broker connection and never receive a partition.

**The real ceiling is CPU.** Each video worker is one ffmpeg process on a full transcode.
`video-resolutions` runs four transcodes per message, so at equal worker counts its throughput is
roughly a quarter of `video` — size it against cores, not partitions.

What the split buys: in v3 all five topics shared one process and one CPU budget, so a video backlog
starved everything else. Now `KAFKA_WORKERS` is per service, and video and image contend only for the
host, not for a worker pool.

Throttle middleware caps concurrent in-flight HTTP requests at 10 000 (server) and 40 000 (client) —
connection guards, unrelated to processing capacity.

---

## 6. Failure design

```
consumer handler
  └─ ffmpeg / validation error
       ├─ remove .processing/, KEEP original.<ext>
       └─ DLQMessage → failed-letter-queue
            └─ failed-consumer: same handler, 3 attempts, 2s backoff
                 ├─ success → "completed"
                 └─ exhausted → "failed"  (asset keeps serving the raw upload)
```

The failed consumer calls the **same handlers** the primary consumers use rather than keeping a
parallel implementation. Those handlers are idempotent, write through a scratch directory, and leave
the raw upload alone on failure — exactly what a retry needs. In v3 the two implementations were
separate and free to drift.

**`DLQMessage`** carries `NewId?`, `OriginalTopic`, `Partition`, `Offset`, `HighWaterMark`, the raw
`Value`, `ErrorDetails`, timestamps, `Worker` and `CustomMessage` — enough to replay by hand.

**Duplicate processing** is now benign. Offsets commit after processing, so a crash in between
redelivers the message; each handler returns early when the converted output already exists. This
also prevents a worse v3 outcome: a redelivered message whose raw input had already been deleted
would fail conversion and end up in the DLQ.

**A retry can still be lost** in two places: the DLQ produce fails, or a DLQ message is unparseable.
Each costs an asset its upgrade rather than its availability — the raw upload stays in place and the
URL keeps serving it.

---

## 7. Security model

| Aspect | State |
|---|---|
| Server auth | one shared `SERVER_KEY`, Bearer |
| Client auth | none — anyone who can reach `:7000` can read any asset by URL |
| Kafka auth | none, `PLAINTEXT` |
| Transport | plain HTTP internally; TLS expected at an external nginx-proxy |
| CORS | per-service allowlists |
| Isolation | `media-docker-proxy` (internal) vs `proxy` (public, client only) |

**The `other` category needs specific care.** It accepts arbitrary uploads by design, so an uploaded
HTML or SVG file served inline would execute on the client's origin — stored XSS against every site
sharing that origin. Two defences in [middleware/fileServer.go](../middleware/fileServer.go):

- `X-Content-Type-Options: nosniff` on every static response.
- `Content-Disposition: attachment` for anything under `documents/` or `others/`, decided **from the
  request path**, not from a query parameter, so it cannot be bypassed by requesting the file
  directly. The resolver passes the original file name for the download, sanitised of path
  separators, quotes and control characters.

Media URLs remain unguessable UUIDs but are not access-controlled: anyone with the URL has the file.

---

## 8. Deployment

`task compose-up` is ordered because the dependencies are timing-sensitive:

1. `task proxy` and `task media-docker-proxy` — create both external networks
2. `docker compose up -d media-docker-kafka-0` — brokers start in reverse dependency order
3. `sleep 10` — wait for KRaft quorum
4. `task kafka-topics` — create topics from `kafka_config.sh` with **RF=3**
5. `docker compose up -d media-docker-server` — pulls up all six consumers via `depends_on`
6. `docker compose up -d media-docker-client`

Dev uses one broker (RF=1) and runs services natively, requiring local Go and ffmpeg.

Both Dockerfiles are parameterised by `ARG SERVICE`, so the eight services build from two definitions
instead of eight near-identical files. Only the four media consumers and the failed consumer include
ffmpeg.

---

## 9. Scaling notes

Where the design bends next:

- **Consumers still need shared storage to scale out.** Every instance needs `media_docker_files`
  read-write, so a multi-host deployment needs NFS or an object store. This remains the biggest
  constraint, and the raw-upload-served-immediately model makes shared storage more load-bearing, not
  less.
- **Storage has no lifecycle.** Nothing expires or tiers `media_docker_files`; the janitor only
  reaps *unclaimed* uploads, not old assets.
- **`video-resolutions` is still four sequential transcodes in one message.** Splitting it into four
  messages would use the partition budget far better, but needs a completion barrier before the
  master playlist can be written — the one piece the current design writes last.
- **No backpressure signal.** The server produces with no view of consumer lag; a burst queues
  invisibly. Less damaging than in v3, since the raw file is already servable, but still opaque.
- **No observability.** Diagnosing a stuck pipeline means reading container logs.
- **Kafka structs remain unversioned.** [topics/structs.go](../topics/structs.go) warns that changing
  a struct breaks in-flight messages; a future change to a payload should add a version field or
  version-suffixed topics first.
