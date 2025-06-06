FROM golang:1.24-alpine

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -o server ./cmd/web/main.go

CMD ["./server"]
