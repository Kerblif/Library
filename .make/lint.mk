.PHONY: yaml-format yaml-lint

# Format every YAML except the OpenAPI tree (owned by openapi-format) in place.
yaml-format:
	go tool -modfile=./tools/go.mod yamlfmt

# Same check without writing; non-zero exit on any unformatted file.
yaml-lint:
	go tool -modfile=./tools/go.mod yamlfmt -lint
