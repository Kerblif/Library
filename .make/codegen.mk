.PHONY: generate generate-api generate-db

# Regenerate everything from the spec and SQL.
generate: generate-api generate-db

# chi strict-server from the OpenAPI spec.
generate-api: openapi-oas30
	go tool -modfile=./tools/go.mod oapi-codegen -config internal/api/oapi-codegen.yaml $(OPENAPI_OAS30)

# Typed query layer from sql/migrations + sql/queries.
generate-db:
	go tool -modfile=./tools/go.mod sqlc generate -f sqlc.yaml
