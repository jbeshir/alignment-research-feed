#!/bin/sh

if [[ "${RUN_BUILD}" -eq 0 ]]; then
  exit
fi
go build -o /app ./cmd/app/app.go
