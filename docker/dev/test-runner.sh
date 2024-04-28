#!/bin/sh
go mod download &
DOWNLOAD_PID=$!

go install -tags mysql github.com/golang-migrate/migrate/v4/cmd/migrate@latest

DOCKERIZE_VERSION=v0.7.0
wget -O - https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz | tar xzf - -C /usr/local/bin
dockerize -wait tcp://database:3306 -timeout 60s

/go/bin/migrate -source file:///source/migrations/dataset -database mysql://${DEV_MYSQL_USER}:${DEV_MYSQL_PASSWORD}@tcp\(database:3306\)/${DEV_MYSQL_DATABASE} up

wait $DOWNLOAD_PID
go test -v ./...