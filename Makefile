.PHONY: build run docker-build docker-run test clean migrate up down logs

build:
	go build -o gateway ./cmd/gateway

run: build
	./gateway

docker-build:
	docker compose build

docker-run:
	docker compose up

test:
	go test ./...

clean:
	rm -f gateway

migrate:
	go run ./cmd/migrate

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f gateway
