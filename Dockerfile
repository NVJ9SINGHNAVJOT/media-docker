FROM node:20-alpine AS base

# Install dependencies only when needed
FROM base AS deps

WORKDIR /app

# Install dependencies
COPY package.json ./
COPY package-lock.json ./
RUN npm ci

# Rebuild the source code only when needed
FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Install FFmpeg
RUN apk add --no-cache ffmpeg
RUN npm run build

EXPOSE 7000

ENTRYPOINT  ["node", "dist/index.js"]