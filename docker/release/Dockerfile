FROM golang:1.22.2-alpine3.19 AS builder
WORKDIR /source
COPY ../../go.mod ../../go.sum ./
COPY ../../internal ./internal
COPY ../../cmd ./cmd
COPY ../../migrations ./migrations
RUN go build -o /app ./cmd/app/app.go

FROM alpine:latest
WORKDIR /
COPY --from=builder /app /app

RUN adduser -D -g '' service
USER service:service
ENTRYPOINT ["/app"]