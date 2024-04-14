SERVICE_NAME=alignment-research-feed
DOCKER_COMMAND=docker-compose --env-file .env -p ${SERVICE_NAME} -f docker/dev/docker-compose.yaml

.PHONY setup-tools:
setup-tools: setup-files
	go install github.com/joho/godotenv/cmd/godotenv@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

.PHONY generate:
generate:
	go generate ./...

.PHONY docker-up:
docker-up:
	${DOCKER_COMMAND} up

.PHONY docker-down:
docker-down:
	${DOCKER_COMMAND} -v down

.PHONY docker-migrate:
docker-migrate:
	godotenv bash -c 'docker run -v $${PWD}/migrations/dataset:/migrations --network host migrate/migrate -path=/migrations/ -database mysql://$${MYSQL_URI} up'

.PHONY docker-build:
docker-build:
	docker build -t alignment-research-feed -f docker/release/Dockerfile .

.PHONY docker-run:
docker-run:
	godotenv bash -c 'docker run --env-file .env --expose $${HTTP_TLS_DISABLED_PORT} -p $${HTTP_TLS_DISABLED_PORT}:$${HTTP_TLS_DISABLED_PORT} alignment-research-feed'

.PHONY docker-mysql:
docker-mysql:
	godotenv bash -c 'docker run -it --network ${SERVICE_NAME}_default --rm mysql mysql -hdatabase -u$${DEV_MYSQL_USER} -p$${DEV_MYSQL_PASSWORD}'

.PHONY setup-files:
setup-files: .env

.env:
	cp .env.dist .env
