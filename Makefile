.PHONY: build test vet race run
build:
	go build -trimpath -o bin/imgo ./cmd/imgo
test:
	go test ./...
vet:
	go vet ./...
race:
	go test -race ./...
run:
	go run ./cmd/imgo
