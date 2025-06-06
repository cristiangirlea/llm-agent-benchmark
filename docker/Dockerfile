# -------- Stage 1: Builder --------
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install Git for fetching Go modules
RUN apk add --no-cache git

# Install Air for hot reload
RUN go install github.com/air-verse/air@latest

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy full project
COPY . .

# Optional: Prebuild one of the entrypoints
RUN go build -o main ./cmd/web

# -------- Stage 2: Final Dev Container --------
FROM golang:1.24-alpine

WORKDIR /app

# Install runtime deps
RUN apk add --no-cache git tzdata

# Copy dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy app source
COPY . .

# Copy air binary
COPY --from=builder /go/bin/air /usr/bin/air

EXPOSE 8080

CMD ["air"]
