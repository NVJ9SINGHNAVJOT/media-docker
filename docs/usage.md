# Integration with Backend

The [_examples/](../_examples/) folder holds the client module you copy into your backend, plus a
runnable demo that uses it. Neither has any dependencies.

```
_examples/
├── nodejs/mediaDocker.ts   the module you copy
└── app/server.ts           a demo that uses it
```

## Copy the module

Copy [_examples/nodejs/mediaDocker.ts](../_examples/nodejs/mediaDocker.ts) into your project and
import the default export. `connect()` is the only handshake — nothing is held open afterwards, so
there is no connection to manage and nothing to disconnect.

```ts
import mediaDocker from "@/utils/mediaDocker";

await mediaDocker.connect("YOUR_SERVER_API_KEY", "http://localhost:7007");
const { data } = await mediaDocker.uploadVideo("/tmp/clip.mp4", 80);
// data.fileUrl works immediately.
```

The upload returns a URL that already works, serving your file as uploaded until conversion finishes
and the converted output silently takes its place at the same URL. There is no completion message to
consume.

Currently only a Node.js module is available. Other language integrations are under development, and the
`_examples/` layout has a folder per language so they can be added without disturbing this one.

## See it work

```bash
cd _examples
MEDIA_DOCKER_SERVER_KEY=your_server_key npm start   # then open http://localhost:3000
```

The demo uploads a file and then polls the asset URL, showing whether it is still serving your raw
upload or has upgraded to the converted output. It needs media-docker-server and
media-docker-client running, plus the consumer for whichever media type you upload.

`npm start` installs nothing — Node runs the `.ts` files directly by stripping types (23.6+, on
22.6–23.5 add `--experimental-strip-types`). See [_examples/README.md](../_examples/README.md) for
the details and the full environment variable list.
