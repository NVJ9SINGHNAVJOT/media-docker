# Examples

```
_examples/
├── nodejs/
│   └── mediaDocker.ts   the module you copy into your own backend
└── app/
    └── server.ts        a runnable demo that uses it
```

No dependencies. There is nothing to install for either one.

## Integrating with your backend

Copy [nodejs/mediaDocker.ts](nodejs/mediaDocker.ts) into your project — the `utils` folder is a
reasonable home — and import the default export:

```ts
import mediaDocker from "@/utils/mediaDocker";

await mediaDocker.connect("YOUR_SERVER_API_KEY", "http://localhost:7007");

const { data } = await mediaDocker.uploadVideo("/tmp/clip.mp4", 80);
// data.fileUrl works right now. Store it and move on.
```

`connect()` is the only handshake; nothing is held open afterwards, so there is no connection to
manage and nothing to disconnect.

Available methods: `uploadVideo`, `uploadVideoResolutions`, `uploadImage`, `uploadAudio`,
`uploadDocument`, `uploadOther`, `deleteFile`.

**The URL is the whole contract.** An upload returns a `fileUrl` that already works — it serves your
file exactly as you uploaded it, so a video plays immediately as a progressive download. When the
consumer finishes, the *same URL* starts serving the converted output. Nothing on your side changes,
there is no completion message to wait for, and if conversion never succeeds the URL keeps serving
your original upload indefinitely: degraded, not broken.

## Running the demo

The demo makes that upgrade visible: it uploads a file, then polls the asset URL and shows whether
it is still serving your raw upload or has switched to the converted output.

```bash
cd _examples
MEDIA_DOCKER_SERVER_KEY=your_server_key npm start
# then open http://localhost:3000
```

`npm start` runs no install step — it is just `node app/server.ts`. TypeScript is executed directly;
see *Running TypeScript* below.

| Variable | Default | |
|---|---|---|
| `MEDIA_DOCKER_SERVER_KEY` | — | required; must match `SERVER_KEY` in `.env.server` |
| `MEDIA_DOCKER_SERVER_URL` | `http://localhost:7007` | media-docker-server |
| `PORT` | `3000` | the demo's own port |

You need **media-docker-server** and **media-docker-client** running, plus the consumer for whichever
media type you upload. Without that consumer the URL stays on the raw upload forever — which is
itself a fair demonstration of the point.

Asset URLs come back already pointing at media-docker-client, because the server builds them from its
own `BASE_URL`, so there is nothing to configure for that.

[app/server.ts](app/server.ts) imports `../nodejs/mediaDocker.ts` directly — the same file you would
copy, not a copy of it. It is ~200 lines of `node:http` plus an inline HTML page, and it accepts the
browser's file as a raw request body rather than as `multipart/form-data`: parsing multipart in core
Node would be most of the file and would demonstrate nothing about media-docker.

## Running TypeScript

Node executes `.ts` files by stripping the types — no compiler, no bundler, no `tsconfig.json`.

| Node version | Command |
|---|---|
| 23.6+ (incl. 24, 25) | `node app/server.ts` — works as-is |
| 22.6 – 23.5 | `node --experimental-strip-types app/server.ts` |
| below 22.6 | run it through `tsx` or `ts-node` |

Two constraints come with native type stripping, both already satisfied here: import specifiers must
carry the `.ts` extension (`from "../nodejs/mediaDocker.ts"`), and only erasable TypeScript syntax is
allowed — no `enum`, no `namespace`, no constructor parameter properties.

`npm start` adds `--disable-warning=ExperimentalWarning` purely to keep the output clean; type
stripping itself is still flagged experimental even though it is on by default.

## If you change the module

These values are matched against the server and will break uploads if they drift:

| Change | Also update |
|---|---|
| chunk size | `helper.Constants.MaxChunkSize` **and** `CHUNK_SIZE` in `nodejs/mediaDocker.ts` |
| an upload endpoint, form field, or status string | `api/` handlers **and** `nodejs/mediaDocker.ts` |

The MIME map in the module is not cosmetic either: the server validates each upload's per-part
`Content-Type` against its own allowlist, which is why the non-standard `image/jpg` and `video/mkv`
entries are there deliberately.
