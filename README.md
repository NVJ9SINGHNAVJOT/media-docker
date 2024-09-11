<p align="center">
   <img src="./assets/images/media_docker_icon.png" alt="Description of the image" style="width: 40%;" />
   <p align="center">Streamline Your Media Management: On-Demand Streaming and Scalable Storage with Media-Docker</p>
</p>

# Media-Docker

Media-Docker is a Docker service designed for storing media files for static use by servers.

## Note

- This project is created for developers, eliminating the hassle of using third-party services to store critical media files.
- It **_can be deployed on a live server_** as **_all features_** all fully **_operational_**.

---

## Features

1. #### Video Streaming :

   - Media-Docker converts video files into segments using **FFmpeg**, enabling on-demand video streaming to the frontend in segments.

2. #### Audio files :

   - Audio files are stored based on the required bitrate provided by the backend service.

3. #### Image files :

   - Image files are stored according to the compression settings provided by the backend service.

## Installation

- Clone the repository to your local machine.
  ```
  git clone https://github.com/NVJ9SINGHNAVJOT/media-docker.git
  ```
- Set up environment variables.
  In the root directory **.env.example** file is present. Replace it with **.env** file and set the required variables running application _(.env.example contains all variables examples)_.
- Project can be run on local machine by Docker or by installing dependencies locally.
- **Using Docker**

  ```
  cd media-docker
  docker compose up -d
  ```

- **Using local machine dependencies**

1. Install golang (if not already installed).
2. Install the required modules and start server.
   ```
   cd media-docker
   task i
   task server
   task client
   ```

- Client will start running at (eg: 7000) 7000 port. [`http://localhost:7000`](http://localhost:7000).
- Server will start running at (eg: 7007) 7007 port. [`http://localhost:7007`](http://localhost:7007).

## Examples

Node.js
   - copy and paste api_node.js.ts file from examples folder to your project
   - eg: api_node.js.ts file is in utils folder in your project

   - video
   ```
   import mediaDocker from "@/utils/api_node_js";

   // upload video
   const video_fileUrl = await mediaDocker.uploadVideo(
      "your_server_key",
      "http://localhost:7007",
      req.file.path
   );
   console.log(video_fileUrl)
   // "http://localhost:7000/media_docker_files/videos/ac9ec121-dad2-48ee-91a5-b9e0e8bcce27/index.m3u8"
   ```

   - image
   ```
   import mediaDocker from "@/utils/api_node_js";

   // upload image
   const image_fileUrl = await mediaDocker.uploadImage(
      "your_server_key",
      "http://localhost:7007",
      req.file.path
   );
   console.log(image_fileUrl)
   // "http://localhost:7000/media_docker_files/images/5f157386-dbf2-46d1-a927-4d837aedbaeb.jpeg"
   ```

## System Design

- [`Open`](https://raw.githubusercontent.com/NVJ9SINGHNAVJOT/media-docker/5fcca46631e9e69bc2f89f0097d55ec4e32561a1/Media-Docker-System-Design.svg)
![Media-Docker-System-Design](https://raw.githubusercontent.com/NVJ9SINGHNAVJOT/media-docker/5fcca46631e9e69bc2f89f0097d55ec4e32561a1/Media-Docker-System-Design.svg)

---

## Important

- Media-Docker utilizes FFmpeg for media file conversion. However, it’s important to note that FFmpeg can be resource-intensive. To optimize performance, consider adjusting your API rate limits and worker pool size based on your system’s available resources.

---


 
