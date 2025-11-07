.PHONY: build run docker-build docker-run test clean

build:
	go build -o gateway ./cmd/gateway

run: build
	./gateway

docker-build:
	docker-compose build

docker-run:
	docker-compose up

test:
	go test ./...

clean:
	rm -f gateway
