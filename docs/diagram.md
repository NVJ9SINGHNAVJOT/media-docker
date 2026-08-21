# System Diagram

Every component and every data path in one picture: uploads, dispatch, the Kafka topics, the six
consumer services, the shared storage volume, the resolver, and the DLQ retry path. Complements
[architecture.md](./architecture.md) (what the pieces are), [systemdesign.md](./systemdesign.md)
(why they're built this way) and [project-flow.md](./project-flow.md) (per-flow detail).

Dotted edges are failure paths. Circled numbers trace the happy path from
[project-flow.md](./project-flow.md) section 1, in order.

```mermaid
flowchart TB
    BE["Your backend<br/>(Node.js SDK)"]

    subgraph SERVER["<div style='background-color: black; color: white; border: 1px solid #aaa; padding: 2px 6px; border-radius: 4px; font-size: 1.1em; font-weight: bold; display: inline-block;'>media-docker-server :7007</div>"]
        direction TB
        UP["Upload API<br/>FileStorage / ChunksStorage"]
        DISP["Dispatch API<br/>video / video-resolutions / image / audio / document / other"]
        DEL["Delete API"]
        JAN["Janitor<br/>hourly sweep"]
    end

    subgraph VOL["<div style='background-color: black; color: white; border: 1px solid #aaa; padding: 2px 6px; border-radius: 4px; font-size: 1.1em; font-weight: bold; display: inline-block;'>media_docker_files/ — shared volume</div>"]
        direction TB
        ASSET["type/id/<br/>meta.json, original.ext, .processing/, hls/"]
    end

    subgraph KAFKA["<div style='background-color: black; color: white; border: 1px solid #aaa; padding: 2px 6px; border-radius: 4px; font-size: 1.1em; font-weight: bold; display: inline-block;'>Kafka cluster — 3 brokers, KRaft</div>"]
        direction LR
        T1(["video"])
        T2(["video-resolutions"])
        T3(["image"])
        T4(["audio"])
        T5(["delete-file"])
        T6(["failed-letter-queue"])
    end

    subgraph CONS["<div style='background-color: black; color: white; border: 1px solid #aaa; padding: 2px 6px; border-radius: 4px; font-size: 1.1em; font-weight: bold; display: inline-block;'>Per-type consumer services</div>"]
        direction TB
        C1["video-consumer"]
        C2["video-resolutions-consumer"]
        C3["image-consumer"]
        C4["audio-consumer"]
        C5["delete-consumer"]
        C6["failed-consumer<br/>retries, 3x backoff"]
    end

    subgraph CLIENT["<div style='background-color: black; color: white; border: 1px solid #aaa; padding: 2px 6px; border-radius: 4px; font-size: 1.1em; font-weight: bold; display: inline-block;'>media-docker-client :7000</div>"]
        direction TB
        RES["Resolver<br/>/media/type/id/variant"]
        STAT["Static file server<br/>/media_docker_files/**"]
    end

    BE -- "① upload chunks or file" --> UP
    UP -- "writes original.ext" --> ASSET
    BE -- "② POST dispatch, id" --> DISP
    DISP -- "produce job" --> T1 & T2 & T3 & T4
    DISP -- "③ 201 id, fileUrl, originalUrl" --> BE
    BE -- "DELETE delete-file" --> DEL
    DEL -- "produce" --> T5
    JAN -. "reap undispatched, hourly" .-> ASSET

    T1 --> C1
    T2 --> C2
    T3 --> C3
    T4 --> C4
    T5 --> C5

    C1 -- "ffmpeg, promote atomically" --> ASSET
    C2 -- "ffmpeg x4, master playlist" --> ASSET
    C3 -- "ffmpeg" --> ASSET
    C4 -- "ffmpeg" --> ASSET
    C5 -- "os.RemoveAll" --> ASSET

    C1 -. "on failure" .-> T6
    C2 -. "on failure" .-> T6
    C3 -. "on failure" .-> T6
    C4 -. "on failure" .-> T6
    T6 --> C6
    C6 -- "same handler, 3 attempts" --> ASSET

    BE -- "④ GET fileUrl or originalUrl" --> RES
    RES -- "reads meta<br/>if converted ➔ pick converted<br/>otherwise ➔ pick original" --> ASSET
    RES -- "302 redirect to picked file" --> STAT
    STAT -- "reads bytes" --> ASSET
    STAT -- "⑤ response body" --> BE
```

Two things this makes visible that the tables in the other docs don't: every consumer writes only to
the shared `media_docker_files` volume, never back through Kafka, and the failed-consumer is the only
service that reads from `failed-letter-queue` — nothing else in the diagram depends on it being up.
