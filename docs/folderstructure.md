# Folder Structure

Single Go module — `github.com/nvj9singhnavjot/media-docker` (Go 1.22) — producing eight binaries.
Everything under `cmd/` is a thin entrypoint, and the real code is split between **shared root packages**
and **`internal/`**.

```
media-docker/
├── cmd/                      # entrypoints — one main.go per binary
├── internal/                 # service-private code + the two Dockerfiles
│   ├── consumerapp/          # shared runtime for all six consumers
│   ├── media-docker-server/  # server routes
│   ├── media-docker-client/  # resolver routes
│   └── media-docker-failed-consumer/
├── api/                      # HTTP handlers + janitor (shared: delete consumer imports it)
├── pkg/asset/                # asset layout, metadata, resolution, atomic promotion
├── kafkahandler/             # Kafka producer/consumer managers
├── topics/                   # Kafka message structs = wire format + validation schema
├── pkg/                      # FFmpeg, dirs, delete channels, env loader
├── config/                   # env validation, logger setup, dir bootstrap
├── helper/                   # constants, JSON responses, request id
├── middleware/               # chi middlewares
├── validator/                # validator instance + custom rules
├── logger/                   # structured Kafka error logging
├── shutdown/                 # HTTP graceful shutdown
├── task_scripts/             # bash helpers invoked by Taskfile
├── docs/                     # this documentation
└── _examples/                # client SDK for consuming backends, plus a runnable demo
```

---

## cmd/ — entrypoints

| Path | Env file | Topic |
|---|---|---|
| `media-docker-server/` | `.env.server` | producer only |
| `media-docker-client/` | `.env.client` | — |
| `media-docker-video-consumer/` | `.env.video` | `video` |
| `media-docker-video-resolutions-consumer/` | `.env.video-resolutions` | `video-resolutions` |
| `media-docker-audio-consumer/` | `.env.audio` | `audio` |
| `media-docker-image-consumer/` | `.env.image` | `image` |
| `media-docker-delete-consumer/` | `.env.delete` | `delete-file` |
| `media-docker-failed-consumer/` | `.env.failed` | `failed-letter-queue` |

Every consumer entrypoint is a single `consumerapp.Run(consumerapp.Config{...})` call.

## pkg/asset/ — the layout authority

The package added in v4. Server, client and consumers all import it so that path knowledge lives in
exactly one place.

| File | Contents |
|---|---|
| [paths.go](../pkg/asset/paths.go) | storage-type constants, `VariantOriginal`, `Dir`, `OriginalPath`, `ProcessingDir`, `HLSDir`, `PlaylistPath`, `ConvertedPath`, `StagingDir`, `URLPath`, `VariantURLPath`, `OriginalURLPath`, `IsVariant`, `Ladder` |
| [meta.go](../pkg/asset/meta.go) | `Meta`, `WriteMeta` (atomic), `ReadMeta`, `MarkDispatched` |
| [resolve.go](../pkg/asset/resolve.go) | `Resolve` (picks the best representation that exists, or the one a variant names), `Resolution.Immutable`, `ErrNotFound` |
| [promote.go](../pkg/asset/promote.go) | `PromoteDir`, `PromoteFile`, `WriteMasterPlaylist` |

`Ladder` is the single source of truth for the resolution ladder — widths, heights and advertised
bandwidths. Both the ffmpeg scale filter and the master playlist derive from it.

`VariantOriginal` (`"original"`) shares the URL position with the rung names: `/media/videos/<id>/720`
and `/media/videos/<id>/original` are both "one specific representation" rather than "the best one".
`IsVariant` is what the resolver route validates against, so which variants a type accepts is decided
here rather than in the client. `Resolution.Immutable` reports whether the chosen representation can
still be replaced, and is what drives the cache header.

## internal/

```
internal/
├── Dockerfile                  # slim runner: server, client, delete consumer
├── Dockerfile.ffmpeg           # ffmpeg runner: 4 media consumers + failed consumer
│                               # both select the binary via ARG SERVICE
├── consumerapp/
│   ├── run.go                  # Config + Run: env, logger, Kafka, workers, shutdown
│   ├── process.go              # topic guard + DLQ routing
│   └── handlers/
│       ├── handlers.go         # prepareProcessing, removeProcessing, exists
│       ├── video.go            # Video, VideoResolutions
│       ├── media.go            # Image, Audio
│       └── delete.go           # DeleteFile (RawHandler)
├── media-docker-server/routes/ # uploadRoutes, destroyRoutes, connectionRoutes
├── media-docker-client/routes/ # resolveRoutes — the /media resolver
└── media-docker-failed-consumer/process/
    └── process.go              # DLQ retry, reusing consumerapp/handlers
```

`consumerapp.Config` takes either a `Handler` (payload in, result out — wrapped with the topic guard
and DLQ routing) or a `RawHandler` (raw Kafka message, failures not retried through the DLQ). The
delete consumer uses `RawHandler` because the failed consumer has no handler for its topic. The
failed consumer uses it because it would otherwise re-queue into the topic it is draining.

## api/ — HTTP handlers

| File | Handler | Does |
|---|---|---|
| [fileStorage.go](../api/fileStorage.go) | `FileStorage` | single-shot upload ≤2 MB, written straight into the asset dir |
| [chunksStorage.go](../api/chunksStorage.go) | `ChunksStorage` | 3-phase chunk upload, merges into `original.<ext>` |
| [dispatch.go](../api/dispatch.go) | — | `claimForDispatch`, `releaseDispatch`, `fileURL`, `variantURL`, `originalURL`, shared by all dispatch handlers |
| [video.go](../api/video.go) | `Video` | claim → produce `video` → return URL |
| [videoResolutions.go](../api/videoResolutions.go) | `VideoResolutions` | claim → produce → return master + 4 variant URLs |
| [image.go](../api/image.go) | `Image` | claim → produce `image` (optional compression) |
| [audio.go](../api/audio.go) | `Audio` | claim → produce `audio` (optional bitrate) |
| [store.go](../api/store.go) | `Document`, `Other` | claim → return URL (no topic, no conversion) |

Every dispatch response carries `originalUrl` alongside `fileUrl` — `{fileUrl}/original`, which serves
the upload unchanged whether or not conversion has run.
| [deleteFile.go](../api/deleteFile.go) | `DeleteFile` | `DeleteFileRequest` + produce `delete-file` |
| [janitor.go](../api/janitor.go) | `StartJanitor` | hourly sweep of abandoned uploads and staging dirs |
| [connect.go](../api/connect.go) | `Connect` | handshake / liveness |

`DeleteFileRequest` is the only exported request struct — it doubles as the `delete-file` payload.

## Shared root packages

| Package | Notable contents |
|---|---|
| [topics/](../topics/) | `DLQMessage`, `AudioMessage`, `ImageMessage`, `VideoMessage`, `VideoResolutionsMessage` |
| [kafkahandler/](../kafkahandler/) | `KafkaProducer`, `KafkaConsumer`, `CheckAllKafkaConnections` |
| [pkg/](../pkg/) | `ConvertVideo`, `ConvertVideoResolutions`, `ConvertImage`, `ConvertAudio`, the directory delete channel, dir helpers, `.env` parser |
| [config/](../config/) | `ServerEnv`, `ClientEnv`, `ConsumerEnv` + validators, logger setup, `CreateDirSetup` |
| [helper/](../helper/) | `Constants` (storage paths, chunk size, per-type MIME allowlists and sizes), `SanitizeExt`, JSON responses |
| [middleware/](../middleware/) | default middleware stack, Bearer auth, request logging, secure file server |
| [validator/](../validator/) | `ValidateRequest`, `UnmarshalAndValidate`, `ExtractNewId` |

`Constants.Files` covers six categories: image (50 MB), video (1 GB), audio (50 MB),
document (100 MB, office/pdf/text allowlist), other (200 MB, `AllowAnyType`).

`SanitizeExt` derives the stored extension from the client's file name, falling back to the MIME
subtype and then a per-category default — the MIME subtype of a `.docx` is not a usable extension.

## Ops files

| File | Purpose |
|---|---|
| [Taskfile.yaml](../Taskfile.yaml) | `i`, `build`, `test`, `server`, `client`, `video`, `video-resolutions`, `audio`, `image`, `delete`, `failed`, `proxy`, `dev-kafka`, `kafka-topics`, `compose-up` |
| [docker-compose.yaml](../docker-compose.yaml) | 8 services + 3 brokers, consumers share a YAML anchor |
| [kafka_config.sh](../kafka_config.sh) | `topics_and_partitions` — single source of truth for topics |
| [.env.example](../.env.example) | template for all eight `.env.*` files |

## _examples/

```
_examples/
├── package.json          # "type": "module" and the start script, no dependencies
├── nodejs/
│   └── mediaDocker.ts    # the SDK you copy into your backend
└── app/
    └── server.ts         # a runnable demo that imports it
```

[nodejs/mediaDocker.ts](../_examples/nodejs/mediaDocker.ts) — Node.js SDK, no third-party
dependencies. Handles the connect handshake, 2 MB chunking, and `uploadVideo`,
`uploadVideoResolutions`, `uploadImage`, `uploadAudio`, `uploadDocument`, `uploadOther`, `deleteFile`.
Every upload result carries `id`, `fileUrl` and `originalUrl`, plus `fileUrls` for
`uploadVideoResolutions`. One folder per language, so other bindings can be added without disturbing
this one.

[app/server.ts](../_examples/app/server.ts) — `node:http` demo that uploads a file and then polls the
asset URL, showing whether it still serves the raw upload (`Cache-Control: no-store`) or a settled
representation (`max-age=300`). It imports `../nodejs/mediaDocker.ts` directly rather than keeping a
copy, and Node runs the `.ts` files by stripping types, so there is no build step. Uploads arrive as
`multipart/form-data`, parsed with the WHATWG `Request`/`formData()` bridge over the incoming stream,
so there is still no dependency to install.

[app/index.html](../_examples/app/index.html) — the front end. Polls `/status`, plays the asset URL
before and after conversion, and offers a source picker: `Auto / 360p / 480p / 720p / 1080p /
Original` for video-resolutions, `Converted / Original` for video, image and audio. The `Original`
entry is what makes the retained upload visible — it plays `originalUrl` while the converted output
stays exactly where it was.

## Not in the repo

The eight `.env.*` files (gitignored — create from `.env.example`), `dist/`, and the runtime
directories `uploadStorage/` and `media_docker_files/`.
