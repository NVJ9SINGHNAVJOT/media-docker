# Build Stage
FROM golang:1.22-alpine AS builder

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
RUN cd cmd/media-docker-failed-consumer && go build -o ../../dist/main main.go

# Runner Stage
FROM alpine:latest AS runner

WORKDIR /app

# Copy the built binary from the builder stage to the runner stage
# This transfers the compiled application to the new image
COPY --from=builder /app/dist .

# Install FFmpeg in the runner image for media processing capabilities
# The --no-cache option ensures no cache is used, keeping the image size smaller
RUN apk add --no-cache ffmpeg

ENTRYPOINT ["/app/main"]
