set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

fmt:
    gofmt -w cmd internal

fmt-check:
    test -z "$(gofmt -l cmd internal)"

lint:
    go vet ./...

test:
    go test ./...

run:
    go run ./cmd/request-info

ci: fmt-check lint test
