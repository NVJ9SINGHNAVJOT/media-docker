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
   task run
   ```

- Project will start running at (eg: 7000) 7000 port. [`http://localhost:7000`](http://localhost:7000).

## Important

- Media-Docker utilizes FFmpeg for media file conversion. However, it’s important to note that FFmpeg can be resource-intensive. To optimize performance, consider adjusting your API rate limits based on your system’s available resources.

---


 
