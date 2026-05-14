.PHONY: build test lint docker fuzz

build:
	go build ./...

test:
	go test -race ./...

lint:
	go vet ./...

fuzz:
	go test -fuzz=FuzzStringToOnionLayer ./node_server/model/

docker:
	docker compose up --build
