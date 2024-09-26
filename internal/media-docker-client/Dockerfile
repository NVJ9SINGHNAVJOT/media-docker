# Build Stage
# Use the official Go image as the build stage with Alpine Linux for a lightweight image
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Pre-copy go.mod and go.sum to leverage Docker cache
# This helps in downloading dependencies only when these files change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the rest of the application source code into the working directory
COPY . .

# Build the Go application
# Navigate to the specific directory where the main.go file is located and build the binary
RUN cd cmd/media-docker-client && go build -o ../../dist/main main.go

# Runner Stage
# Start a new stage for the runtime environment
FROM builder AS runner

# Set the working directory in the runner image
WORKDIR /app

# Copy the built binary from the builder stage to the runner stage
COPY --from=builder /app/dist .

# Expose port 7000 to allow external access to the application
EXPOSE 7000

# Set the entry point for the container to execute the compiled Go binary
ENTRYPOINT [ "/app/main" ]
