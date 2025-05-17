# Contributing to Media Docker Project

Thank you for considering contributing to the Media Docker project! Your contributions are what make this project great.

## Found an Issue or Got a Question?

For now, contact can be made on [LinkedIn](https://www.linkedin.com/in/nvjsinghnavjot).

## Requirements

Before you begin, please ensure you have the following installed:

- [Docker](https://www.docker.com/get-started)
- [Apache Kafka](https://kafka.apache.org)
- [Go](https://go.dev)

If you prefer not to install Kafka locally, you can use the Bitnami/Kafka Docker image:
- [Bitnami/Kafka](https://hub.docker.com/r/bitnami/kafka)

## FFmpeg

This project uses FFmpeg for all media file conversions. All commands are defined in the `pkg/ffmpegCommand.go` file.

To learn more about FFmpeg, visit:
- [FFmpeg](https://www.ffmpeg.org)

#### media-docker-consumer

FFmpeg commands need to be properly integrated in order and settings as they are solely and mainly responsible for load in media-docker-consumer service as for heavy tasks.

## How to Contribute

1. **Fork the Repository**: Start by forking the repository to your own GitHub account.
2. **Clone the Repository**: Clone your forked repository to your local machine.

   ```bash
   git clone https://github.com/NVJ9SINGHNAVJOT/media-docker.git
   cd media-docker
   ```
   
3. Create a New Branch: Create a new branch for your changes.
4. Make Changes: Make your changes, commit them, and push your changes to your forked repository on GitHub.
5. Create a Pull Request: Go to the original repository where you want to contribute.
   You should see a prompt to create a pull request for your newly pushed branch. Click on that and fill in the necessary details.
7. Review: Once your pull request is created, it will be reviewed. You may be asked to make additional changes based on feedback.

## Development Workstation Setup

Once the repo is cloned, the first thing is to set up environment variables for:
- .env.client
- .env.server
- .env.consumer
- .env.failed

All examples are provided in a single .env.example file for all four files.

(Dependencies installed locally as mentioned in the Installation section of the readme.md file)

All required commands are given in the task file in the root folder.

## Order of Starting Services

#### media-docker-kafka:
- Run **_task dev-kafka_** then **_task dev-kafka-topics_**. This starts the Kafka development container for development.

#### media-docker-consumer:
- Run **_task consumer_**. This service is responsible for all heavy tasks related to file conversions and storage (Max resource consumption).

#### media-docker-failed:
- Run **_task failed_**. This service handles retries for failed messages from other topics by consuming them from the "failed-letter-queue".

#### media-docker-server:
- Run **_task server_**. This service provides API services for file upload and sending messages to Kafka.

#### media-docker-client:
- Run **_task client_**. This service provides static files for HTTP requests.

Once you have started the above servers in order, it is not necessary to follow this order for subsequent servers.
Only media-docker-server and media-docker-consumer depend on the media-docker-kafka container, so now each server can be run independently for development and testing.

## Concurrent Workers in Consumer

Properly closing the media-docker-consumer and media-docker-failed services is crucial, as they run Kafka message workers concurrently using goroutines. If not shut down correctly, these workers may continue running in the background, consuming system resources unnecessarily. Each media-docker service includes a graceful shutdown procedure to ensure resources are released appropriately.

## Code Comments
The Media Docker project follows this format for code comments, each with a specific purpose:

// CAUTION: Warns about potential risks or sensitive areas in the code.

// FIXME: Highlights an issue in the code that needs to be fixed.

// BUG: Indicates a known bug or defect that should be addressed.

// HACK: Signifies a temporary or non-ideal solution that needs improvement later.

// PERF: Marks a section of code that could be optimized for better performance.

// TODO: Marks a task or feature that needs to be implemented in the future.

// NOTE: Provides additional context or explanation about a specific part of the code.

// INFO: Offers useful information or clarification about the current implementation.
