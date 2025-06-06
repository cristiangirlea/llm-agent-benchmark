# --- Stage 1: Build ---
FROM golang:1.24 AS builder

WORKDIR /app
COPY . .

# Ensure modules are tidy
RUN go mod tidy

# Build the server binary for Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/web

# --- Stage 2: Run ---
FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /app/server .

# Make sure the binary is executable
RUN chmod +x ./server

EXPOSE 8080
ENTRYPOINT ["./server"]
