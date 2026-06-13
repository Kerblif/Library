.PHONY: test

# Run the Go test suite (needs generated code; run `make generate` first).
test:
	go test ./...
