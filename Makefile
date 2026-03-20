DOCS_DIR := docs
SWAGGER_FILE := $(DOCS_DIR)/swagger.yaml
OPENAPI_FILE := $(DOCS_DIR)/openapi3.yaml

.PHONY: swag generate test run_py
swag:
	@echo "📚 Formatting Swagger documentation..."
	@swag fmt
	@echo "📝 Generating Swagger documentation..."
	@swag init
	@echo "✅ Swagger documentation generated: $(SWAGGER_FILE)"
	@echo "🔄 Converting to OpenAPI3..."
	@swagger2openapi $(SWAGGER_FILE) -o $(OPENAPI_FILE) --patch
	@echo "✅ OpenAPI 3 documentation generated: $(OPENAPI_FILE)"

generate:
	@echo "🔧 Generating code..."
	go generate ./...
	@echo "✅ Code generation complete"

test:
	@echo "🧪 Running tests..."
	go test ./... -v
	@echo "✅ Tests complete"

run_py:
	@echo "🐍 Running Python script..."
	@cd embdservice && \
	trap 'echo "\n🛑 Stopping Python..."; exit 0' INT TERM && \
	.venv/bin/python main.py