# Runtime image for services that do not run ffmpeg: the server, the client and
# the delete consumer.
#
# The service to build is selected by the SERVICE build argument.
#
#   docker build --build-arg SERVICE=media-docker-server -f internal/Dockerfile .

# Build Stage
FROM golang:1.22-alpine AS builder

ARG SERVICE
WORKDIR /app

# Pre-copy go.mod and go.sum to leverage Docker caching
# This allows Go to pre-download dependencies only when these files change,
# speeding up the build process in subsequent builds
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the entire application source code into the working directory
COPY . .

# Build the selected service's binary
RUN test -n "$SERVICE" || (echo "SERVICE build argument is required" && exit 1) \
    && go build -o /app/dist/main ./cmd/${SERVICE}

# Runner Stage
FROM alpine:latest AS runner

WORKDIR /app

# Copy the built binary from the builder stage to the runner stage
COPY --from=builder /app/dist .

ENTRYPOINT ["/app/main"]
