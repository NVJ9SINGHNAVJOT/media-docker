/*
  A tiny demo backend for media-docker.

  It exists to show the one thing that is hard to picture from the README: a media-docker
  URL works the moment the upload returns, and it silently upgrades to the converted output
  when a consumer finishes. Same URL, no callback, nothing to wait for.

  It also shows the other half of that: converting does not consume the upload. Every asset
  keeps the file you sent, at originalUrl, so the UI can switch between the converted output
  and the source at any point.

  This imports ../nodejs/mediaDocker.ts directly -- the same file you would copy into your
  own backend, not a copy of it. Nothing here is compiled or bundled first.

  Run (Node 23.6+, which strips TypeScript types natively):
    MEDIA_DOCKER_SERVER_KEY=your_server_key node app/server.ts

  On Node 22.6-23.5 the same command needs --experimental-strip-types. Below 22.6, run it
  through tsx or ts-node instead. There are no dependencies to install either way.

  Environment:
    MEDIA_DOCKER_SERVER_KEY   required, must match SERVER_KEY in the server's .env.server
    MEDIA_DOCKER_SERVER_URL   default http://localhost:7007  (media-docker-server)
    PORT                      default 3000

  The asset URLs come back already pointing at media-docker-client, because the server
  builds them from its own BASE_URL -- so there is nothing to configure here for it.

  You need media-docker-server and media-docker-client running, plus the consumer for
  whichever media type you upload -- otherwise the URL stays on the raw upload forever,
  which is itself a fair demonstration of the point.
*/

import http from "node:http";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import mediaDocker from "../nodejs/mediaDocker.ts";

const SERVER_KEY = process.env.MEDIA_DOCKER_SERVER_KEY;
const SERVER_URL = process.env.MEDIA_DOCKER_SERVER_URL || "http://localhost:7007";
const PORT = Number(process.env.PORT) || 3000;

/** The upload endpoints this demo exposes. */
type Endpoint = "video" | "video-resolutions" | "image" | "audio" | "document" | "other";

/**
 * Maps each endpoint to the storage category the asset ends up in, which is what
 * deleteFile needs. Note that "video-resolutions" is stored as a video -- the two
 * endpoints differ in how the file is converted, not in where it lands.
 */
const STORAGE_TYPE: Record<Endpoint, string> = {
  video: "video",
  "video-resolutions": "video",
  image: "image",
  audio: "audio",
  document: "document",
  other: "other",
};

if (!SERVER_KEY) {
  console.error("MEDIA_DOCKER_SERVER_KEY is not set. It must match SERVER_KEY in .env.server.");
  process.exit(1);
}

/** Sends a JSON response. */
function json(res: http.ServerResponse, status: number, body: unknown): void {
  const payload = JSON.stringify(body);
  res.writeHead(status, {
    "Content-Type": "application/json",
    "Content-Length": Buffer.byteLength(payload),
  });
  res.end(payload);
}

import { Readable } from "node:stream";

async function saveToTempFile(req: http.IncomingMessage, fileName: string): Promise<{ target: string; dir: string; parsedType?: string; parsedName?: string; parsedOption?: string }> {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "media-docker-demo-"));
  let target = path.join(dir, path.basename(fileName || "upload.tmp"));

  const contentType = req.headers["content-type"] || "";
  if (contentType.includes("multipart/form-data")) {
    const headers = new Headers();
    for (let i = 0; i < req.rawHeaders.length; i += 2) {
      headers.append(req.rawHeaders[i], req.rawHeaders[i + 1]);
    }

    const webReq = new Request("http://localhost", {
      method: req.method,
      headers,
      body: Readable.toWeb(req) as any,
      duplex: "half",
    });

    const formData = await webReq.formData();
    const file = formData.get("file") || formData.get("videoFile") || formData.get("imageFile") || formData.get("audioFile");
    const parsedType = formData.get("type") as string;
    const parsedOption = formData.get("option") as string;
    const parsedName = formData.get("name") as string || (file && (file as File).name);

    if (parsedName) target = path.join(dir, path.basename(parsedName));

    if (file && typeof file !== "string") {
      const buffer = Buffer.from(await file.arrayBuffer());
      fs.writeFileSync(target, buffer);
    }

    return { target, dir, parsedType, parsedName, parsedOption };
  }

  // Fallback to raw body streaming
  return new Promise((resolve, reject) => {
    const out = fs.createWriteStream(target);
    req.pipe(out);
    out.on("finish", () => resolve({ target, dir }));
    out.on("error", reject);
    req.on("error", reject);
  });
}

/** Removes a temp directory, ignoring failures -- it is only a temp file. */
function cleanup(dir: string): void {
  fs.rm(dir, { recursive: true, force: true }, () => {});
}

/** Calls the matching module method for an endpoint. */
async function upload(endpoint: Endpoint, filePath: string, option: string) {
  switch (endpoint) {
    case "video":
      return await mediaDocker.uploadVideo(filePath, option ? Number(option) : undefined);
    case "video-resolutions":
      return await mediaDocker.uploadVideoResolutions(filePath);
    case "image":
      return await mediaDocker.uploadImage(filePath, option ? Number(option) : undefined);
    case "audio":
      return await mediaDocker.uploadAudio(filePath, (option || undefined) as "128k" | "192k" | "256k" | "320k" | undefined);
    case "document":
      return await mediaDocker.uploadDocument(filePath);
    case "other":
      return await mediaDocker.uploadOther(filePath);
  }
}

type AssetState = "raw" | "converted" | "missing" | "unknown";

/**
 * Reports whether an asset URL currently serves the raw upload or the converted output.
 *
 * media-docker-client answers an asset URL with a 302 to whichever representation exists,
 * and the cache header says whether that representation can still change: `no-store` while
 * the URL is still on the raw upload and a conversion could replace it, `max-age=300` once
 * the answer is settled. Reading that header is all it takes to know which one you have.
 *
 * Documents and others therefore report "converted" straight away. Nothing converts them,
 * so their one representation is final from the start -- the caller renders that as
 * "stored" rather than "converted", and stops polling.
 *
 * It runs here rather than in the browser only so the demo needs no CORS setup on the
 * client service.
 */
async function checkStatus(fileUrl: string): Promise<{
  state: AssetState;
  status: number;
  cacheControl?: string;
  target?: string;
}> {
  const res = await fetch(fileUrl, { method: "GET", redirect: "manual" });

  // A 404 means the asset is gone -- deleted, or reaped by the janitor before dispatch.
  if (res.status === 404) {
    return { state: "missing", status: res.status };
  }

  const cacheControl = res.headers.get("cache-control") || "";
  const target = res.headers.get("location") || "";

  if (res.status !== 302) {
    return { state: "unknown", status: res.status, cacheControl, target };
  }

  return {
    state: cacheControl.includes("no-store") ? "raw" : "converted",
    status: res.status,
    cacheControl,
    target,
  };
}

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url ?? "/", `http://localhost:${PORT}`);

  try {
    if (req.method === "GET" && url.pathname === "/") {
      const html = fs.readFileSync(path.join(import.meta.dirname, "index.html"));
      res.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
      res.end(html);
      return;
    }

    if (req.method === "POST" && url.pathname === "/upload") {
      let endpoint = (url.searchParams.get("type") || "") as Endpoint;
      let fileName = url.searchParams.get("name") || "";
      let option = url.searchParams.get("option") || "";

      const { target, dir, parsedType, parsedName, parsedOption } = await saveToTempFile(req, fileName);
      
      // Override with form-data fields if they exist
      if (parsedType) endpoint = parsedType as Endpoint;
      if (parsedName) fileName = parsedName;
      if (parsedOption) option = parsedOption;

      if (!STORAGE_TYPE[endpoint]) {
        cleanup(dir);
        json(res, 400, { message: `unknown type: ${endpoint}` });
        return;
      }
      if (!fileName) {
        cleanup(dir);
        json(res, 400, { message: "name is required" });
        return;
      }

      try {
        // result.data is passed through as-is: id, fileUrl, originalUrl, and
        // fileUrls for video-resolutions. The browser needs all of them.
        const result = await upload(endpoint, target, option);
        json(res, 200, { ...result.data, type: STORAGE_TYPE[endpoint], endpoint, message: result.message });
      } finally {
        cleanup(dir);
      }
      return;
    }

    if (req.method === "GET" && url.pathname === "/status") {
      const fileUrl = url.searchParams.get("url");
      if (!fileUrl) {
        json(res, 400, { message: "url is required" });
        return;
      }
      json(res, 200, await checkStatus(fileUrl));
      return;
    }

    if (req.method === "DELETE" && url.pathname === "/file") {
      const id = url.searchParams.get("id");
      const type = url.searchParams.get("type");
      if (!id || !type) {
        json(res, 400, { message: "id and type are required" });
        return;
      }
      await mediaDocker.deleteFile(id, type as "image" | "video" | "audio" | "document" | "other");
      json(res, 200, { message: "deleted" });
      return;
    }

    json(res, 404, { message: "not found" });
  } catch (err) {
    // Every failure in the module surfaces as a thrown Error carrying the server's message.
    json(res, 500, { message: (err as Error).message || "unknown" });
  }
});


// Fail fast on a bad key or an unreachable server, before we ever start listening.
try {
  await mediaDocker.connect(SERVER_KEY, SERVER_URL as "http://localhost:7007");
} catch (err) {
  console.error(`[ERROR] [demo] could not reach media-docker-server at ${SERVER_URL}: ${(err as Error).message}`);
  console.error("[ERROR] [demo] is it running, and does MEDIA_DOCKER_SERVER_KEY match SERVER_KEY in .env.server?");
  process.exit(1);
}

server.on("error", (err: NodeJS.ErrnoException) => {
  if (err.code === "EADDRINUSE") {
    console.error(`[ERROR] [demo] port ${PORT} is already in use. Set PORT to something else.`);
    process.exit(1);
  }
  throw err;
});

server.listen(PORT, () => {
  console.log(`[INFO] [demo] listening on http://localhost:${PORT}`);
});
