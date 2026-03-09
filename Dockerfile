# Multi-stage build; pure Go SQLite (no cgo)
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bot ./cmd/bot

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /bot .
ENV DB_PATH=/data/app.db
VOLUME /data
ENTRYPOINT ["./bot"]
