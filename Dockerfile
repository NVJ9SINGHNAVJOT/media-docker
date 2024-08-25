FROM golang:1.23-alpine

WORKDIR /app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -o ./dist/main main.go

# Install FFmpeg
RUN apk add --no-cache ffmpeg

EXPOSE 7000

ENTRYPOINT [ "/dist/main" ]