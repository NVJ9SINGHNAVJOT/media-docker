# Build Stage
FROM golang:1.22-alpine AS builder

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
FROM builder AS runner

WORKDIR /app

# Copy the built binary from the builder stage to the runner stage
COPY --from=builder /app/dist .

EXPOSE 7000

ENTRYPOINT [ "/app/main" ]
