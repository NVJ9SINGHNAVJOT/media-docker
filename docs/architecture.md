# Architecture

Media-Docker is a Go monorepo that builds **eight independent binaries** from one shared module
(`github.com/nvj9singhnavjot/media-docker`, Go 1.22). All eight share the same internal packages
(`config`, `helper`, `kafkahandler`, `pkg`, `pkg/asset`, `topics`, `validator`, `middleware`, `logger`,
`shutdown`) and are wired together by an **Apache Kafka cluster (3 brokers, KRaft mode)** and **two
shared Docker volumes**.

Two properties define v4:

**Uploads are usable immediately.** The server writes each upload directly into the publicly served
directory and returns a URL that works at once. Until conversion finishes that URL serves the raw
upload, and afterwards the same URL serves the converted output. Callers never wait on a Kafka message
before using a URL, and a conversion that fails permanently degrades quality rather than breaking
the asset.

**Conversion adds, it never replaces.** The raw upload is kept for the life of the asset and stays
addressable at `<url>/original`, so every asset can serve both its processed output and its source
bytes. Only an explicit delete removes anything.

**One consumer per media type.** Conversion is split across five consumer services plus a retry
service, so a backlog of video transcodes cannot starve image or audio work and each type scales on
its own.

---

## 1. Services

| Service | Entrypoint | Role | Kafka topic |
|---|---|---|---|
| `media-docker-server` | [cmd/media-docker-server/](../cmd/media-docker-server/) | Upload API, storage, job dispatch | producer |
| `media-docker-client` | [cmd/media-docker-client/](../cmd/media-docker-client/) | Resolver + static file serving | none |
| `media-docker-video-consumer` | [cmd/media-docker-video-consumer/](../cmd/media-docker-video-consumer/) | Single-quality HLS | `video` |
| `media-docker-video-resolutions-consumer` | [cmd/media-docker-video-resolutions-consumer/](../cmd/media-docker-video-resolutions-consumer/) | Resolution ladder + master playlist | `video-resolutions` |
| `media-docker-audio-consumer` | [cmd/media-docker-audio-consumer/](../cmd/media-docker-audio-consumer/) | MP3 conversion | `audio` |
| `media-docker-image-consumer` | [cmd/media-docker-image-consumer/](../cmd/media-docker-image-consumer/) | JPEG compression | `image` |
| `media-docker-delete-consumer` | [cmd/media-docker-delete-consumer/](../cmd/media-docker-delete-consumer/) | Asset removal (no ffmpeg) | `delete-file` |
| `media-docker-failed-consumer` | [cmd/media-docker-failed-consumer/](../cmd/media-docker-failed-consumer/) | Retries | `failed-letter-queue` |

Ports are **hardcoded in config**, not env-driven — see [config/validateEnvs.go](../config/validateEnvs.go)
(`CLIENT_PORT = "7000"`, `SERVER_PORT = "7007"`). The values in `.env.example` are documentation only.

### media-docker-server
- Chi router. Middleware: CORS → RequestID → Logger → Recoverer → Throttle(10000) → `ServerKey`
  (Bearer auth) → `AllowContentEncoding` → `AllowContentType` → `LoggingRequest`.
- Routes at `/api/v1/uploads`, `/api/v1/destroys`, `/api/v1/connections`
  ([internal/media-docker-server/routes/](../internal/media-docker-server/routes/)), with handlers in the
  shared [api/](../api/) package (shared because the delete consumer imports `api.DeleteFileRequest`).
- Mounts `uploadStorage` **rw** for chunk staging and `media_docker_files` **rw** — writing uploads
  into the served tree is what makes URLs work before conversion.
- Runs a janitor goroutine ([api/janitor.go](../api/janitor.go)) reaping abandoned uploads.
- Handles `document` and `other` end to end: no topic, no consumer, final the moment they land.

### media-docker-client
- Two route trees, deliberately not overlapping:
  - `/media/**` — the **resolver** ([internal/media-docker-client/routes/](../internal/media-docker-client/routes/)),
    which 302s to whichever representation currently exists.
  - `/media_docker_files/**` — static file serving via [middleware/fileServer.go](../middleware/fileServer.go).
- Only service on the external `proxy` network. No Kafka, no auth.
- Mounts `media_docker_files` **ro**.

### Consumers
All six run the same shared runtime, [internal/consumerapp/](../internal/consumerapp/). Each `main.go`
is ~10 lines supplying a topic, an env file and a handler. The runtime owns env
loading, logger setup, the Kafka connection check, producer/consumer wiring, worker supervision and
the shutdown handshake.

Consumers mount **only** `media_docker_files` **rw**. Since uploads are merged straight into media
storage, nothing but the server touches the staging volume.

---

## 2. Kafka topology

Topics and partition counts are declared once in [kafka_config.sh](../kafka_config.sh) and applied by
`task kafka-topics` (prod, RF=3) or `task dev-kafka-topics` (dev, RF=1).

| Topic | Partitions | Produced by | Consumed by | Payload |
|---|---|---|---|---|
| `video` | 100 | server | video consumer | `topics.VideoMessage` |
| `video-resolutions` | 100 | server | video-resolutions consumer | `topics.VideoResolutionsMessage` |
| `image` | 100 | server | image consumer | `topics.ImageMessage` |
| `audio` | 50 | server | audio consumer | `topics.AudioMessage` |
| `delete-file` | 20 | server | delete consumer | `api.DeleteFileRequest` |
| `failed-letter-queue` | 10 | consumers | failed consumer | `topics.DLQMessage` |

`document` and `other` have **no topics** — they need no processing.

Every topic is internal. No topic is consumed by an integrating backend, which talks to media-docker
over HTTP only.

All payload structs are in [topics/structs.go](../topics/structs.go) and carry `validate:` tags: the
same struct is the wire format and the validation schema.

**Hard constraint:** `KAFKA_WORKERS` for a service, summed across all its instances, must be ≤ that
topic's partition count. Excess workers idle forever.

### Producer / consumer mechanics
- Producer: shared `kafka.Writer`, `MaxAttempts: 10`, topic per message.
- Consumer: `kafka.Reader` per worker, `GroupID` = `consumer-<topic>-group`, `HeartbeatInterval: 3s`.
- **Manual commit after processing** — at-least-once delivery. Redelivery is now harmless: every
  handler returns early if the converted output already exists.
- Reader failures retry 5× with 4s backoff, then the worker dies and `decrementWorker` logs the
  shrinking pool. Workers are **not respawned**, so when the last dies the service exits and Docker's
  `restart: unless-stopped` restarts it.

---

## 3. Storage model

Two named Docker volumes, unchanged from v3:

| Volume | Mount | Constant | Contents |
|---|---|---|---|
| `media-docker-upload-data` | `/app/uploadStorage` | `Constants.UploadStorage` | chunk staging (server only) |
| `media-docker-files-data` | `/app/media_docker_files` | `Constants.MediaStorage` | all assets, publicly served |

**Every asset is a directory**, uniform across types. This is the change that makes the resolver,
deletion, and the raw→converted swap all trivial:

```
media_docker_files/
  videos/<id>/
    meta.json                    id, type, ext, originalName, dispatched, job, createdAt
    original.mp4                 raw upload — served until conversion finishes, then kept
    .processing/                 conversion scratch, never served
    hls/index.m3u8               single-quality playlist, or master playlist
    hls/{360,480,720,1080}/index.m3u8
  images/<id>/    meta.json, original.png, converted.jpeg
  audios/<id>/    meta.json, original.wav, converted.mp3
  documents/<id>/ meta.json, original.pdf
  others/<id>/    meta.json, original.zip
```

`original.<ext>` is never deleted. A consumer only ever adds converted output beside it, so an asset
that has been converted holds both representations: the asset URL serves the converted one, and
`<url>/original` serves the upload. That costs storage — see Known gaps — and buys a source-quality
download, an intact input for any later reprocessing, and a retry path that always still has its
input.

[pkg/asset/](../pkg/asset/) is the single owner of this layout — server, client and consumers all
import it rather than building paths by hand.

**Atomicity is load-bearing.** ffmpeg writes an HLS playlist incrementally, so `index.m3u8` exists
and is incomplete for the whole duration of a transcode. Consumers therefore write into
`.processing/` and promote with `os.Rename` on success ([pkg/asset/promote.go](../pkg/asset/promote.go)).
Converted output either is not there at all or is there complete.

There is exactly **one id per asset**, minted when the upload starts. v3's separate upload handle and
output UUID are gone.

Cleanup is asynchronous via [pkg/channel.go](../pkg/channel.go) for the server's request-path
directory deletions — chunk staging and half-written assets. The janitor, the consumers' scratch
cleanup and asset deletion are synchronous, so nothing is silently dropped when the buffered channel
is full. There is no queue for single files: nothing deletes a file out of an asset directory.

---

## 4. Networks

| Network | Type | Members | Purpose |
|---|---|---|---|
| `proxy` | external | client | public/frontend access |
| `media-docker-proxy` | external | server, all 6 consumers, all 3 brokers | internal, your backend joins this |

Both are created out-of-band (`task proxy`, `task media-docker-proxy`) so they survive `compose down`.

---

## 5. Cross-cutting concerns

**Config** — [pkg/loadEnv.go](../pkg/loadEnv.go) reads a `.env.*` file if present, otherwise real env
vars. Eight env files, one per service. Every consumer uses the same three variables
(`ENVIRONMENT`, `KAFKA_BROKERS`, `KAFKA_WORKERS`) validated by `config.ValidateConsumerEnv()`.

**Auth** — single shared secret, `Authorization: Bearer <SERVER_KEY>`
([middleware/serverKey.go](../middleware/serverKey.go)). The client service has no auth.

**Validation** — one `go-playground/validator` instance per process with two custom rules
([validator/validator.go](../validator/validator.go)), used for both HTTP bodies and Kafka payloads.

**Logging** — `zerolog`, console in `development`, JSON otherwise. Kafka errors carry
topic/partition/offset/highWaterMark/value ([logger/logger.go](../logger/logger.go)).

**Shutdown** — server/client use `shutdown.WaitForShutdownSignal` (60s / 20s). Consumers cancel a
context, `wg.Wait()` and close the producer. They queue no background deletes, so there is nothing to
drain.

---

## 6. Reliability

The failure path, in order:

1. Conversion fails → handler removes `.processing/`, **leaves `original.<ext>` in place** →
   `DLQMessage` to `failed-letter-queue`.
2. `failed-consumer` retries the **same handler** up to 3× with backoff. It does not maintain a
   parallel implementation. The handlers are idempotent and safe to re-run, and since no path removes the
   raw upload, a retry always still has its input, however many deliveries preceded it.
3. Success → the converted output is promoted. Exhaustion → logged and abandoned.
4. **Either way the URL keeps working.** Exhaustion means the asset permanently serves its raw upload.

Where a retry can still be lost:

- The DLQ produce fails → logged, and the conversion is never retried.
- The failed consumer cannot unmarshal a DLQ message → logged, and there is nothing to retry.

Each of these costs an asset its upgrade, not its availability: the raw upload stays in place and the
URL keeps serving it.

### Known gaps

- Dead workers are not respawned ([kafkahandler/consumer.go](../kafkahandler/consumer.go) `HACK:`).
- The server's request-path directory delete channel still drops work silently when full
  ([pkg/channel.go](../pkg/channel.go)). It only ever loses a temporary staging directory, which the
  janitor then reaps.
- No metrics, health endpoints, or consumer-lag visibility.
- **No storage lifecycle policy: `media_docker_files` grows unboundedly**, and retaining originals
  makes it grow roughly twice as fast for converted media. Nothing ages an asset out, and only
  `deleteFile` and the janitor's undispatched sweep remove anything. This is the most significant
  gap on the list.
- No automated tests. [pkg/asset/](../pkg/asset/) is the obvious place to start: it owns the layout,
  and `Resolve` is a pure state matrix over what exists on disk.
