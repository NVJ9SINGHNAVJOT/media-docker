<p align="center">
   <img src="./assets/images/media_docker_icon.png" alt="Description of the image" style="width: 40%;" />
   <p align="center">Streamline Your Media Management: On-Demand Streaming and Scalable Storage with Media-Docker</p>
</p>

# Media-Docker - Version 2

**Media-Docker** is a media management service designed to handle video, audio, and image processing with the power of **FFmpeg** for conversion, storage, and on-demand streaming. This version introduces several new components, including a client, server, Kafka-based message queuing, and worker-based message consumption for media processing tasks.

## Note

- This project is created for developers, eliminating the hassle of using third-party services to store critical media files.
- It **_can be deployed on a live server_** as **_all features_** all fully **_operational_**.

## Acknowledgements

This project uses several open-source libraries, each of which is credited to its respective authors and owners. We would like to extend a big thank you to all the contributors and maintainers of these libraries for their hard work and dedication.

A complete list of the libraries used can be found in the go.mod and go.sum files.

---

## Project Structure

- **media-docker-client**: Serves static media assets and processes frontend requests for video streaming, image, and audio retrieval.

- **media-docker-server**: The backend service that handles media upload requests, validating them and sending messages to Kafka for processing.

- **media-docker-kafka-cluster**: Manages the flow of messages from the media-docker-server to the media-docker-consumer by distributing them across Kafka topics to aid in media processing tasks. It also enables the mediaDocker module (as a consumer) to consume response messages for media files once conversion is complete.

- **media-docker-kafka-consumer**: Consumes messages from Kafka topics and performs media-related operations, such as creating video segments or converting audio bitrates with **FFmpeg**. This component handles resource-intensive tasks.

- **mediaDocker module (in the \_examples folder for backend servers, according to language)**: Contains the core logic for uploading files to the Media-Docker service and manages the consumption of messages from the Kafka response topic for media file task results.

## Features

### Video Streaming

- **Media-Docker** utilizes **FFmpeg** to convert uploaded video files into various resolutions (360p, 480p, 720p, 1080p), making them available for on-demand streaming.
- Videos are segmented for seamless playback and adaptive quality streaming, allowing users to switch between different qualities dynamically.

### Audio Processing

- Audio files are stored with the required **bitrate**, as specified by the backend, ensuring flexibility and support for various audio quality needs.
- Dedicated **consumer workers** manage audio processing tasks, ensuring efficient and scalable handling of large media libraries.

### Image Compression

- Images are compressed and stored according to custom **compression settings** provided by the backend service.
- Processing and compression of images are managed by consumer workers, optimizing efficiency and storage.

## Kafka Integration

The media-docker-kafka-cluster component leverages a Kafka cluster with 3 brokers in KRaft mode to receive messages from different topics, promoting scalable and asynchronous media processing. The **media-docker-kafka-consumer** executes the primary tasks associated with each topic. Key topics and their respective responsibilities include:

- **video**: Manages video conversion and segmentation tasks.
- **video-resolutions**: Handles resolution conversion for videos.
- **audio**: Processes and stores audio files based on the specified bitrate.
- **image**: Manages image compression and storage.
- **delete-file**: Oversees requests for media file deletion.
- **media-docker-files-response**: Holds the results of media file conversions for the mediaDocker module to consume.
- **failed-letter-queue**: Facilitates the retry mechanism for media files that have encountered issues.

By leveraging **Kafka** and **FFmpeg**, the project guarantees scalable, efficient media processing with dedicated workers for each topic.

## FFmpeg Integration

- **FFmpeg** serves as the primary tool for converting video and audio files, segmenting videos into streamable parts, adjusting resolutions, and compressing images for storage.

## Usage

After setting up all components, upload media files through the server, which processes the uploads and sends messages to Kafka for various media tasks. Consumers will handle the intensive operations, while the client serves the processed files.

## Contributing

Contributions to Media-Docker are always welcome! To submit feature requests, report bugs, or contribute to the project, please open an issue or submit a pull request. For guidelines on contributing and maintaining the project, refer to the [CODE_OF_CONDUCT.md](https://github.com/NVJ9SINGHNAVJOT/media-docker/blob/main/CODE_OF_CONDUCT.md) and [CONTRIBUTING.md](https://github.com/NVJ9SINGHNAVJOT/media-docker/blob/main/CONTRIBUTING.md) files.

## Conclusion

The **Media-Docker** project, now in version 2, is a complete media processing solution built for scalability and efficiency using **Kafka** workers, **FFmpeg**, and a robust client-server architecture. It supports advanced video streaming, flexible audio processing, and image compression tailored to specific needs, making it an ideal solution for media-heavy applications.

## Installation

- Clone the repository to your local machine.
  ```sh
  git clone https://github.com/NVJ9SINGHNAVJOT/media-docker.git
  ```
- Set up environment variables.
  In the root directory **.env.example** file is present. Replace it with **.env.client**, **.env.server**, **.env.consumer** file and set the required variables running application _(.env.example contains all variables examples for all envs)_.
- Project can be run on local machine by Docker or by installing dependencies locally.
- **Using Docker:** **_Recommended for Production_**

  ```sh
  cd media-docker
  task compose-up
  ```

- **Using local machine dependencies:** **_Recommended for Development_**

1. Install golang (if not already installed).
2. Install the required modules and components.
3. If you have Apache Kafka installed locally, skip the _task dev-kafka_ and _task dev-kafka-topics_ steps, and create the topics as described in the _this_create_kafka_topics.sh_ file. Otherwise, start Docker (Apache Kafka is used in this project with Docker) and execute the following task commands:

   ```sh
   cd media-docker
   task i
   task dev-kafka
   task dev-kafka-topics

   # Below tasks need to run in different terminals:
   task consumer
   task server
   task client
   ```

- Client will start running at (eg: 7000) 7000 port. [`http://localhost:7000`](http://localhost:7000).
- Server will start running at (eg: 7007) 7007 port. [`http://localhost:7007`](http://localhost:7007).

- You can execute the **_task_** command in the terminal to view all the available commands in the task file.

---

## Examples

### Node.js Integration

- Copy the mediaDocker.ts file from the examples folder into your project.
- Example: Place the file in the utils folder of your project.

- First connect to media-docker-server

```ts
import mediaDocker from "@/utils/mediaDocker";

// First, connect to the media-docker-server
async function connectToMediaDocker() {
  try {
    // Establish a connection to the Media-Docker service and the Kafka broker.
    // The connect function requires three parameters:
    // 1. serverKey: A string representing the server key (e.g., "your_server_key").
    // 2. mediaDockerURL: The URL for the Media-Docker server (e.g., "http://localhost:7007" or "https://your-media-docker-server.com").
    // 3. messageHandler: A function that processes incoming Kafka messages.
    //    This handler operates asynchronously in the background for each message received from the Kafka topic.

    await mediaDocker.connectMediaDockerAndKafka(
      "YOUR_SERVER_API_KEY",
      "https://your.media.docker.server/api",
      async (message) => {
        // Handle incoming messages from Kafka
        console.log("Received message:", message);
        // Received message:
        // {
        //     id: "123e4567-e89b-12d3-a456-426614174000",
        //     fileType: "video",
        //     status: "completed"
        // }

        // Implement your logic based on message processing status
        if (message.status === "completed") {
          console.log(`Processing completed for file ID: ${message.id}`);
          // Processing completed for file ID: 123e4567-e89b-12d3-a456-426614174000
        } else {
          console.error(`Processing failed for file ID: ${message.id}`);
        }
      }
    );
  } catch (error) {
    console.error("Error connecting to Media-Docker:", error);
  }
}
```

- video

```ts
import mediaDocker from "@/utils/mediaDocker";

// upload video
const result = await mediaDocker.uploadVideo("/path/to/video.mp4", 80);

console.log(result);
// {
//     "message": "video uploaded successfully",
//     "data": {
//         "fileUrl": "http://example.com/media_docker_files/videos/5d71228e-bff9-44a5-b949-f8e5a32b95a4/index.m3u8",
//         "id": "5d71228e-bff9-44a5-b949-f8e5a32b95a4"
//     }
// }
```

- video resolutions

```ts
import mediaDocker from "@/utils/mediaDocker";

// upload video resolutions
const result = await mediaDocker.uploadVideoResolutions("/path/to/video.mp4");

console.log(result);
// {
//     "message": "video uploaded successfully",
//     "data": {
//         "id": "8a39e8c1-e0fb-4d34-9719-58ac2cb2f3b0"
//         "fileUrls": {
//             "360": "http://example.com/media_docker_files/videos/8a39e8c1-e0fb-4d34-9719-58ac2cb2f3b0/360/index.m3u8",
//             "480": "http://example.com/media_docker_files/videos/8a39e8c1-e0fb-4d34-9719-58ac2cb2f3b0/480/index.m3u8",
//             "720": "http://example.com/media_docker_files/videos/8a39e8c1-e0fb-4d34-9719-58ac2cb2f3b0/720/index.m3u8"
//             "1080": "http://example.com/media_docker_files/videos/8a39e8c1-e0fb-4d34-9719-58ac2cb2f3b0/1080/index.m3u8",
//         },
//     }
// }
```

- audio

```ts
import mediaDocker from "@/utils/mediaDocker";

// upload audio
const result = await mediaDocker.uploadAudio("/path/to/audio.wav", "192k");

console.log(result);
// {
//   "message": "audio uploaded and processed successfully",
//   "data": {
//       "id": "3ef614d5-8d1c-4e2d-a463-dc412f31dc46"
//       "fileUrl": "http://example.com/media_docker_files/audios/3ef614d5-8d1c-4e2d-a463-dc412f31dc46.mp3",
//   }
// }
```

- image

```ts
import mediaDocker from "@/utils/mediaDocker";

// upload image
const result = await mediaDocker.uploadImage("/path/to/image.png");

console.log(result);
// {
//   "message": "image uploaded successfully",
//   "data": {
//       "id": "2321155f-af55-4819-b5b4-0bf667086a18"
//       "fileUrl": "http://example.com/media_docker_files/images/2321155f-af55-4819-b5b4-0bf667086a18.jpeg",
//   }
// }
```

## System Design

- [`Open`](https://raw.githubusercontent.com/NVJ9SINGHNAVJOT/media-docker/e2df0cef4f2721c04e7d138d568806ebf310bc98/Media-Docker-System-Design.svg)

  ![Media-Docker-System-Design](https://raw.githubusercontent.com/NVJ9SINGHNAVJOT/media-docker/e2df0cef4f2721c04e7d138d568806ebf310bc98/Media-Docker-System-Design.svg)

## Important

- Media-Docker utilizes FFmpeg for media file conversion. However, it’s important to note that FFmpeg can be resource-intensive. To optimize performance, consider adjusting your API rate limits and worker pool size based on your system’s available resources.

---
