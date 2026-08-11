FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1

FROM alpine:3.21
RUN apk add --no-cache ca-certificates ffmpeg
COPY --from=builder /server /server
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY db/migrations /db/migrations
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 8081
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/server"]
