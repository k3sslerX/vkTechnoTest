FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY subscribeservice/ ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o subsservice ./cmd_server/main.go

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/subsservice .

RUN chmod +x /app/subsservice

EXPOSE 1313
CMD ["/app/subsservice"]