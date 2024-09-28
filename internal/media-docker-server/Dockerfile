# Build Stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Pre-copy go.mod and go.sum to leverage Docker caching
# This allows Go to pre-download dependencies only when these files change,
# speeding up the build process in subsequent builds
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the entire application source code into the working directory
# This includes all necessary files for building the application
COPY . .

# Build the Go application
# Change directory to where the main.go file is located and build the binary
RUN cd cmd/media-docker-server && go build -o ../../dist/main main.go

# Runner Stage
FROM builder AS runner

WORKDIR /app

# Copy the built binary from the builder stage to the runner stage
# This transfers the compiled application to the new image
COPY --from=builder /app/dist .

EXPOSE 7007

ENTRYPOINT [ "/app/main" ]
