# Build Stage
# Use the official Go image based on Alpine for a lightweight build environment
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container for the build process
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
# Start a new stage for running the application, using the previous builder stage
FROM builder AS runner

# Set the working directory for the runtime environment
WORKDIR /app

# Copy the built binary from the builder stage to the runner stage
# This transfers the compiled application to the new image
COPY --from=builder /app/dist .

# Expose port 7007 to allow external access to the application
EXPOSE 7007

# Set the entry point for the container to execute the compiled Go binary
# This is the command that will run when the container starts
ENTRYPOINT [ "/app/main" ]
