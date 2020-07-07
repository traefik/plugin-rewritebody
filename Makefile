export GO111MODULE=on

.PHONY: lint test clean

default: lint test

lint:
	golangci-lint run

test:
	go test -v -cover ./...