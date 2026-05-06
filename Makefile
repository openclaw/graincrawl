.PHONY: test vet tidy check build run

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

check: tidy vet test
	git diff --exit-code -- go.mod go.sum

build:
	go build -o bin/graincrawl ./cmd/graincrawl

run:
	go run ./cmd/graincrawl --help
