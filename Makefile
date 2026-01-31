SERVICE_NAME=alignment-research-feed
DOCKER_COMMAND=docker compose --env-file .env -p ${SERVICE_NAME} -f docker/dev/docker-compose.yaml -f docker/dev/docker-compose.dev.yaml

.PHONY setup-tools:
setup-tools: setup-files
	go install github.com/joho/godotenv/cmd/godotenv@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/vektra/mockery/v2@latest
	go install github.com/daveshanley/vacuum@latest

.PHONY generate:
generate:
	go generate ./...

.PHONY mocks:
mocks:
	mockery

.PHONY test-short:
test-short:
	go test -v -short ./...

.PHONY docker-up:
docker-up:
	${DOCKER_COMMAND} up

.PHONY docker-down:
docker-down:
	${DOCKER_COMMAND} -v down

.PHONY docker-test:
docker-test:
	docker compose --env-file .env.dist -p ${SERVICE_NAME}-test down -v
	docker compose --env-file .env.dist -p ${SERVICE_NAME}-test -f docker/dev/docker-compose.yaml -f docker/dev/docker-compose.test.yaml up --build --abort-on-container-exit --exit-code-from testrunner

.PHONY docker-migrate:
docker-migrate:
	godotenv bash -c 'docker run -v $${PWD}/migrations/dataset:/migrations --network host migrate/migrate -path=/migrations/ -database mysql://$${MYSQL_URI} up'

.PHONY docker-build:
docker-build:
	docker build -t alignment-research-feed -f docker/Dockerfile .

.PHONY docker-run:
docker-run:
	godotenv bash -c 'docker run --env-file .env --expose $${PORT} -p $${PORT}:$${PORT} alignment-research-feed'

.PHONY docker-mysql:
docker-mysql:
	godotenv bash -c 'docker run -it --network ${SERVICE_NAME}_default --rm mysql mysql -hdatabase -u$${DEV_MYSQL_USER} -p$${DEV_MYSQL_PASSWORD}'

.PHONY setup-files:
setup-files: .env

.env:
	cp .env.dist .env

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: lint-openapi
lint-openapi:
	vacuum lint openapi/api.yaml

.PHONY: build-openapi-docs
build-openapi-docs:
	npx @redocly/cli build-docs openapi/api.yaml -o openapi/index.html

.PHONY: fmt
fmt:
	go fmt ./...
	goimports -w .

.PHONY: build-mcp
build-mcp:
	go build -o bin/alignment-feed-mcp ./cmd/mcp

.PHONY: build-mcp-all
build-mcp-all:
	GOOS=darwin GOARCH=amd64 go build -o bin/alignment-feed-mcp-darwin-amd64 ./cmd/mcp
	GOOS=darwin GOARCH=arm64 go build -o bin/alignment-feed-mcp-darwin-arm64 ./cmd/mcp
	GOOS=linux GOARCH=amd64 go build -o bin/alignment-feed-mcp-linux-amd64 ./cmd/mcp
	GOOS=windows GOARCH=amd64 go build -o bin/alignment-feed-mcp-windows-amd64.exe ./cmd/mcp
