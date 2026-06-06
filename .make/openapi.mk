OPENAPI_SRC    := api/openapi.yaml
OPENAPI_BUNDLE := api/openapi.bundle.yaml
OPENAPI_OAS30  := api/openapi.oas30.yaml

.PHONY: openapi-format openapi-bundle openapi-oas30

# Reformat and split the spec into the multi-file authoring tree.
openapi-format:
	npx openapi-format $(OPENAPI_SRC) -o $(OPENAPI_SRC) --split

# Bundle the tree into a single 3.1 document (source for MCP and the down-convert).
openapi-bundle:
	npx openapi-format $(OPENAPI_SRC) -o $(OPENAPI_BUNDLE)

# Down-convert 3.1 -> 3.0; oapi-codegen does not support 3.1 yet (issue #373).
openapi-oas30: openapi-bundle
	npx @apiture/openapi-down-convert --input $(OPENAPI_BUNDLE) --output $(OPENAPI_OAS30)
