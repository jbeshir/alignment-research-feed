SERVICE_NAME=alignment-research-feed
DOCKER_COMMAND=docker compose --env-file .env -p ${SERVICE_NAME} -f docker/dev/docker-compose.yaml -f docker/dev/docker-compose.dev.yaml

.PHONY setup-tools:
setup-tools: setup-files
	go install github.com/joho/godotenv/cmd/godotenv@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install golang.org/x/tools/cmd/goimports@latest

.PHONY generate:
generate:
	go generate ./...

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

.PHONY: fmt
fmt:
	go fmt ./...
	goimports -w .
