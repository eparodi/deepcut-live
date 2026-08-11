FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates ffmpeg
COPY --from=builder /server /server
COPY db/migrations /db/migrations
EXPOSE 8081
CMD ["/server"]
