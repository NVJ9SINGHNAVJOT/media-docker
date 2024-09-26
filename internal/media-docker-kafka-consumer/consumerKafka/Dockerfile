# Build Stage
# Use the official Go image based on Alpine for a lightweight build environment
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container for the build process
WORKDIR /app

# Pre-copy go.mod and go.sum to leverage Docker caching
# This helps in downloading dependencies only when these files change, 
# speeding up subsequent builds
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the entire application source code into the working directory
# This includes the task script and any other necessary files
COPY . .

# Build the Go application
# Change directory to the location of the main.go file and build the binary
RUN cd cmd/media-docker-kafka-consumer && go build -o ../../dist/main main.go

# Ensure the Kafka topic creation script is executable
# This step grants execute permissions to the script
RUN chmod +x task_scripts/create_kafka_topics.sh

# Create Kafka topics using the hardcoded Kafka container name
# This step runs the script to create the necessary Kafka topics
RUN ./task_scripts/create_kafka_topics.sh "media-docker-kafka"

# Runner Stage
# Start a new stage for running the application using a minimal Alpine image
FROM alpine:latest AS runner

# Set the working directory for the runtime environment
WORKDIR /app

# Copy the built binary from the builder stage to the runner stage
# This transfers the compiled application to the new image
COPY --from=builder /app/dist .

# Install FFmpeg in the runner image for media processing capabilities
# The --no-cache option ensures no cache is used, keeping the image size smaller
RUN apk add --no-cache ffmpeg

# Set the entry point for the container to execute the compiled Go binary
# This is the command that will run when the container starts
ENTRYPOINT ["/app/main"]
