# Project Flow

End-to-end traces of every path through the system, with the file that owns each step.
For structure see [folderstructure.md](./folderstructure.md), for rationale see
[systemdesign.md](./systemdesign.md), and for the whole thing in one picture see
[diagram.md](./diagram.md).

---

## 1. The happy path

```
your backend            media-docker-server        Kafka         video-consumer      client
     │                          │                    │                  │              │
     │─ upload chunks ─────────▶│ merge into         │                  │              │
     │                          │ videos/<id>/       │                  │              │
     │◀ { id } ─────────────────│ original.mp4       │                  │              │
     │                          │                    │                  │              │
     │─ POST /video {id} ──────▶│── produce ────────▶│                  │              │
     │◀ 201 { id, fileUrl,      │                    │─ fetch ─────────▶│              │
     │       originalUrl } ─────│                    │            ffmpeg → .processing/│
     │   ┌──────────────────────────────────────────────────────────────┐             │
     │   │ fileUrl ALREADY WORKS — 302 → original.mp4                   │─────────────▶│
     │   └──────────────────────────────────────────────────────────────┘             │
     │                          │                    │          rename → hls/          │
     │                          │                    │      original.mp4 stays         │
     │                                                                                 │
     │   same fileUrl — 302 → hls/index.m3u8 ────────────────────────────────────────▶│
     │   originalUrl  — 302 → original.mp4 ──────────────────────────────────────────▶│
```

The boxed step is the point of v4: the URL is usable before conversion has started, and the caller
never has to switch to a different URL afterwards.

Note what is **not** in the diagram: no message ever comes back to your backend. The consumer
commits its offset and stops. There is no completion topic and nothing to subscribe to. The two
final lines are the same two URLs the dispatch response already returned.

---

## 2. Startup flow

**Server** ([cmd/media-docker-server/main.go](../cmd/media-docker-server/main.go)):
`LoadEnv(".env.server")` → `ValidateServerEnv` → `SetUpLogger` → `DirExist` both roots →
`config.CreateDirSetup()` (creates the full tree for all five categories) →
`CheckAllKafkaConnections` → `InitializeKafkaProducerManager` → `go pkg.DeleteDirWorker()` →
`go api.StartJanitor()` → `InitializeValidator` → router → `go WaitForShutdownSignal(srv, 60)` →
`ListenAndServe`.

The server is the only service that creates directories, because it is the only one that creates
assets.

**Client** ([cmd/media-docker-client/main.go](../cmd/media-docker-client/main.go)):
env → logger → `DirExist(MediaStorage)` → router → resolver routes at `/media` →
static file server at `/media_docker_files` → `go WaitForShutdownSignal(srv, 20)` → serve.

**Consumers** — all six run [internal/consumerapp/run.go](../internal/consumerapp/run.go):
`LoadEnv` → `ValidateConsumerEnv` → `SetUpLogger` → `DirExist(MediaStorage)` →
`CheckAllKafkaConnections` → `InitializeValidator` → producer + consumer manager for the single
configured topic → `go KafkaConsumeSetup()` → select loop on `sigChan` / `workDone`.

Consumers start no delete workers. They only ever add files to an asset directory, and they clear
their own scratch directory synchronously, so there is no background deletion to run or drain.

Each spawns `KAFKA_WORKERS` goroutines in group `consumer-<topic>-group`, workers named
`consumer-<topic>-group-worker-<i>`.

---

## 3. Upload flow — chunked (> 2 MB)

`POST /api/v1/uploads/chunks-storage`, multipart, `Authorization: Bearer <SERVER_KEY>`
([api/chunksStorage.go](../api/chunksStorage.go)).

**Phase 1 — `start`** (`type`, `status=start`, `chunk=0`, `fileName`, `<type>File`)
1. `checkForm` validates the category, parses `chunk`, enforces *chunk 0 ⟺ start*, and **mints the
   asset id** — the final media id, not a throwaway handle.
2. MIME allowlist check → 415, size vs 2 MB → 413. (`other` skips the allowlist by design.)
3. Creates `uploadStorage/<type>s/<id>/`, writes `chunk_0`. → `200 { id }`

**Phase 2 — `uploading`** (adds `id`, `chunk=n`) — validates the id as UUIDv4, writes `chunk_<n>`.

**Phase 3 — `completed`**
1. Writes the final chunk, then queues the staging directory for deletion whatever happens next.
2. `totalChunksSize` vs the category max → 400 if over.
3. `finalizeUpload`: derive the extension via `Constants.SanitizeExt`, create
   `media_docker_files/<type>s/<id>/`, **merge chunks directly into `original.<ext>`**, write
   `meta.json` with `dispatched:false`.
4. → `200 { id }` — **the asset is now publicly resolvable.**

Any failure during finalisation removes the asset directory, so a truncated file is never left
resolvable.

**Single-shot** (`/file-storage`, ≤ 2 MB) does the same without staging.

---

## 4. Dispatch flow

```
POST /api/v1/uploads/{video|video-resolutions|image|audio|document|other}
{ "id": "<uuid4>", "quality": 80 }     // or bitrate / compression / nothing
```

Shared logic in [api/dispatch.go](../api/dispatch.go):

1. `ValidateRequest` — decode + struct-tag validate.
2. `claimForDispatch` — read `meta.json` (404-ish 400 if absent), reject if already dispatched (409),
   confirm `original.*` is on disk, then **mark dispatched before producing**. The ordering matters:
   the asset is already live, and the janitor reaps anything still flagged undispatched, so marking
   after a slow produce could let the janitor delete an asset mid-conversion.
3. Produce the job with `FilePath` pointing at `original.<ext>`.
4. On produce failure, `releaseDispatch` reverts the flag so the caller can retry and the janitor can
   still reap it → 500.
5. → `201 { id, fileUrl, originalUrl }`, plus `fileUrls` for video-resolutions.

Every response carries `originalUrl`. Bodies are wrapped in the standard `{ message, data }`
envelope, so the fields below are what appears under `data`.

| Endpoint | Topic | `fileUrl` | Also returned |
|---|---|---|---|
| `/video` | `video` | `{BASE_URL}/media/videos/{id}` | `originalUrl` |
| `/video-resolutions` | `video-resolutions` | same, master playlist once converted | `fileUrls` for 360/480/720/1080, `originalUrl` |
| `/image` | `image` | `{BASE_URL}/media/images/{id}` | `originalUrl` |
| `/audio` | `audio` | `{BASE_URL}/media/audios/{id}` | `originalUrl` |
| `/document`, `/other` | **none** | `{BASE_URL}/media/{documents,others}/{id}` | `originalUrl`, addressing the same bytes |

`originalUrl` is always `{fileUrl}/original`. It is returned rather than left to be constructed so
that callers never build media-docker paths themselves — the same reason `pkg/asset` owns every path
on the server side.

`document` and `other` ([api/store.go](../api/store.go)) queue nothing and return. They still require
this call — it is what marks them claimed so the janitor leaves them alone.

---

## 5. Consumption flow

Per worker, forever ([kafkahandler/consumer.go](../kafkahandler/consumer.go)):

```
FetchMessage(ctx) ──▶ ProcessMessage(msg, worker) ──▶ CommitMessages(1-min ctx)
       │                                                     │
   error → retry 5× w/ 4s backoff                        error → log (may redeliver)
   ctx cancelled → clean exit
```

[internal/consumerapp/process.go](../internal/consumerapp/process.go) wraps the handler: verify the
topic matches, call it, then publish `completed` or route to the DLQ.

Every handler ([internal/consumerapp/handlers/](../internal/consumerapp/handlers/)) follows one
contract:

1. **Skip if already converted** — at-least-once delivery means redelivery is normal.
2. `prepareProcessing` — clear and recreate `<id>/.processing/`.
3. Run ffmpeg into the scratch directory.
4. **Success** → `PromoteDir` / `PromoteFile` (atomic rename). Nothing else.
5. **Failure** → remove `.processing/` synchronously → return the error.

**`original.<ext>` is never touched on either path.** On failure that is why nothing breaks: the URL
keeps resolving to the raw file. On success it is why the source bytes remain available at
`<url>/original`, and why a redelivered or retried message always still has its input. The only thing
that removes an asset is the delete flow below.

Scratch removal is synchronous rather than queued, because the failed consumer retries immediately
and a queued delete could land after the retry recreated the directory.

`VideoResolutions` additionally writes the master playlist into `.processing/` before promoting, so
the ladder and its index appear together.

---

## 6. Delete flow

```
DELETE /api/v1/destroys/delete-file
{ "id": "<uuid4>", "type": "image|video|audio|document|other" }
```

[api/deleteFile.go](../api/deleteFile.go) validates, checks the asset directory exists, produces to
`delete-file`, and returns 200 ("queued for deletion" — removal is asynchronous).

[handlers/delete.go](../internal/consumerapp/handlers/delete.go) does one `os.RemoveAll` on the asset
directory. Uniform layout means no per-type branching: raw upload, converted output and metadata all
go together. Failures are logged rather than routed to the DLQ, which has no handler for this topic.

---

## 7. Failure flow — DLQ round trip

**Step 1** — the handler fails, and `handleErrorResponse` builds a `DLQMessage` (original
topic/partition/offset/high-water mark/raw value, error, timestamps, worker) and produces it to
`failed-letter-queue`. If that produce fails the message is logged and the conversion is never
retried.

**Step 2** — [internal/media-docker-failed-consumer/process/process.go](../internal/media-docker-failed-consumer/process/process.go):

```
unmarshal DLQMessage ──ok──▶ handleDLQMessage
        │
       fail ──▶ log only (nothing to retry)
```

**Step 3** — `retryConversion` re-runs **the same handler** the primary consumer used, up to 3
attempts with 2s backoff. No parallel implementation exists.

**Step 4** — success → the converted output is promoted, exhaustion → logged as *"asset keeps serving
its raw upload"*. Either way the URL still works, and nothing needs to be told which happened.

---

## 8. Serving flow

Two route trees on the client, deliberately separate so they cannot collide:

**`/media/**` — the resolver** ([internal/media-docker-client/routes/resolveRoutes.go](../internal/media-docker-client/routes/resolveRoutes.go)):

| Route | Serves |
|---|---|
| `/media/{types}/{id}` | the best representation that exists — converted output, else the raw upload |
| `/media/videos/{id}/{360\|480\|720\|1080}` | that rung of the ladder |
| `/media/{types}/{id}/original` | the raw upload, for every type, whatever else exists |

`asset.Resolve` picks the representation and the route replies `302`. `Cache-Control` is `no-store`
only while the answer is still provisional — an unconverted asset whose URL a consumer may replace at
any moment. Everything settled gets `max-age=300`, including an explicitly requested original and the
never-converted `documents`/`others`. A variant the type cannot serve is rejected with a 404 rather
than silently falling back. `documents`/`others` get download parameters appended.

**`/media_docker_files/**` — static files** ([middleware/fileServer.go](../middleware/fileServer.go)):
registers `.m3u8` and `.ts` MIME types (Go's builtin table has neither and the Alpine image has no
`/etc/mime.types`), sets `nosniff` on everything, and forces `Content-Disposition: attachment` for
`documents/` and `others/` **based on the path**, so it cannot be bypassed.

---

## 9. Shutdown flow

**Server / client** — SIGINT/SIGTERM → `srv.Shutdown(ctx)` (60s / 20s). The server then closes the
producer, closes the directory delete channel and sleeps 10s so the delete goroutine drains.

**Consumers** — the select loop catches the signal → `cancel()` → every reader sees `ctx.Done()` →
`wg.Wait()` → close producer. There is no delete channel to close and nothing to drain. The same loop
also exits when `workDone` closes, i.e. every worker has died and the pool is exhausted, and Docker then
restarts it.

Compose `stop_grace_period`: 60s server and consumers, 30s client.

---

## 10. Developer flow

**First run**

```sh
cp .env.example .env.server        # then create all eight files and fill values
task i                             # go mod download && verify
task dev-kafka                     # single-broker dev container
task dev-kafka-topics              # topics from kafka_config.sh, RF=1

# separate terminals, one command each:
task server
task client
task video
task video-resolutions
task audio
task image
task delete
task failed
```

Requires local Go 1.22 and ffmpeg on `PATH`. Server `:7007`, client `:7000`. You only need the
consumers for the media types you are exercising.

**Production**

```sh
task compose-up      # networks → brokers → sleep 10 → topics (RF=3) → server → client
task compose-down
```

**Useful**

| Command | Does |
|---|---|
| `task` | list all tasks |
| `task build` | eight binaries into `dist/` |
| `task test` | `go test ./...` — wired up, but there are no tests yet |
| `task k-cluster` | KRaft quorum + replication status |
| `task kafka-topics` / `task delete-topics` | create / delete topics |

**Changing things — the coupled edits**

| Change | Also update |
|---|---|
| add/rename a Kafka topic | [kafka_config.sh](../kafka_config.sh), [topics/structs.go](../topics/structs.go), a new `cmd/` entrypoint, the failed consumer's `topicHandlers`, `.env.example`, compose |
| change chunk size | `helper.Constants.MaxChunkSize` **and** `CHUNK_SIZE` in [_examples/nodejs/mediaDocker.ts](../_examples/nodejs/mediaDocker.ts) |
| change an upload form field, endpoint or status string | the `api/` handler **and** [_examples/nodejs/mediaDocker.ts](../_examples/nodejs/mediaDocker.ts) |
| change the asset layout | [pkg/asset/](../pkg/asset/) only — every other package goes through it |
| change the resolution ladder | `asset.Ladder` only — scale filter and master playlist both derive from it |
| add a media type | `helper.Constants.Files`, `asset` type constants, resolver routes, an API route, SDK method |
| change a topic struct | breaks in-flight messages of the old shape — version it first |
| rename a volume or network | don't, compose marks these as breaking changes |

---

## 11. Migrating from v3

- **Old URLs keep working.** Pre-v4 assets are flat (`videos/<id>/index.m3u8`, `images/<id>.jpeg`)
  and the static mount is unchanged, so every URL already handed out still resolves. They are not
  reachable through `/media/...`. Old and new coexist with no data migration.
- **`.env.consumer` is gone**, replaced by five per-consumer files. Create them before `compose-up`.
- **API responses changed**: storage endpoints return `{ id }` instead of `{ uuidFilename }` /
  `{ newChunkId }`, and dispatch endpoints take `{ id }`. Copy the updated
  [_examples/nodejs/mediaDocker.ts](../_examples/nodejs/mediaDocker.ts).
- **`fileUrl` now points at `/media/...`**, not directly at a file.
- **The server needs `media_docker_files` read-write**, and the consumers no longer need `uploadStorage`
  at all.
