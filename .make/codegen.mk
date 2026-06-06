.PHONY: generate

# Regenerate the chi strict-server from the spec.
generate: openapi-oas30
	go tool -modfile=./tools/go.mod oapi-codegen -config internal/api/oapi-codegen.yaml $(OPENAPI_OAS30)
