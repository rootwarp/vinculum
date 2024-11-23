.PHONY: build test

build:
	go build -o ./bin/vinculum cmd/vinculum/main.go

test:
	go test ./...
