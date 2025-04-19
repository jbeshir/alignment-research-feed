FROM golang:1.24.2-alpine3.21 AS builder
WORKDIR /source
COPY ../../go.mod ../../go.sum ./
COPY ../../internal ./internal
COPY ../../cmd ./cmd
COPY ../../migrations ./migrations
COPY ../../docker ./docker

ARG RUN_BUILD=1
RUN sh /source/docker/run-build.sh

FROM alpine:latest
WORKDIR /
COPY --from=builder /app /app

RUN adduser -D -g '' service
USER service:service
ENTRYPOINT ["/app"]