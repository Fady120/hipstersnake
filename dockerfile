# Stage 1: Build the Go app and compile frontend assets]
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install Node.js
RUN apk add --no-cache nodejs npm

# Install CoffeeScript compiler
RUN npm install -g coffeescript

# Copy go.mod and go.sum separately first for caching
COPY go.mod go.sum ./

RUN go mod tidy

# Copy the rest of the source
COPY . .

# Compile frontend assets
RUN coffee --compile static

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o snake ./src/snake/main/main.go


# Stage 2: Minimal runtime container
FROM debian:bookworm-slim

WORKDIR /app

# Update and install security patches
RUN apt-get update && apt-get upgrade -y && apt-get clean

# Copy only the final binary and assets
COPY --from=builder /app/snake /app/snake
COPY --from=builder /app/static /app/static
COPY --from=builder /app/index.html /app/index.html

EXPOSE 8000

CMD ["./snake"]