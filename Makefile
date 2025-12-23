.PHONY: help build run test clean docs swagger postman install-tools

# Default target
help:
	@echo "Available targets:"
	@echo "  make build         - Build the API binary"
	@echo "  make run           - Run the API server"
	@echo "  make test          - Run all tests"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make clean         - Clean build artifacts and generated files"
	@echo "  make docs          - Generate OpenAPI documentation"
	@echo "  make swagger       - Alias for 'make docs'"
	@echo "  make postman       - Generate Postman collection from OpenAPI spec"
	@echo "  make install-tools - Install required development tools"
	@echo "  make lint          - Run golangci-lint"
	@echo "  make migrate-up    - Run database migrations"
	@echo "  make migrate-down  - Rollback database migrations"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/swaggo/swag/v2/cmd/swag@latest
	go install github.com/golang/mock/mockgen@latest
	@echo "Tools installed successfully"

# Generate OpenAPI documentation
docs: swagger

swagger:
	@echo "Generating OpenAPI 3.1 documentation..."
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal -ot yaml,json --v3.1
	@echo "Documentation generated successfully in ./docs"

# Generate Postman collection from OpenAPI spec
postman: docs
	@echo "Generating Postman collection..."
	@if command -v npx > /dev/null 2>&1; then \
		npx --yes openapi-to-postmanv2 -s docs/swagger.json -o postman/Array-Banking-API.postman_collection.json -p; \
		echo "Postman collection generated successfully"; \
		echo "Applying collection enhancements..."; \
		node scripts/fix-postman-collection.js; \
		node scripts/organize-postman-collection.js; \
		node scripts/add-auth-scripts.js; \
		node scripts/add-happy-path-tests.js; \
		node scripts/add-error-tests.js; \
		node scripts/create-test-suites.js; \
		node scripts/add-performance-assertions.js; \
		echo "Postman collection ready in ./postman"; \
	else \
		echo "Error: npx not found. Install Node.js to generate Postman collection."; \
		exit 1; \
	fi

# Build the application (includes documentation generation)
build: docs
	@echo "Building application..."
	go build -o api cmd/api/main.go
	@echo "Build complete: ./api"

# Run the application
run: build
	@echo "Starting API server..."
	./api

# Run tests
test:
	@echo "Running tests..."
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter (requires golangci-lint to be installed)
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Clean build artifacts and generated documentation
clean:
	@echo "Cleaning build artifacts..."
	rm -f api
	rm -f coverage.out coverage.html
	rm -rf docs/docs.go docs/swagger.json docs/swagger.yaml
	@echo "Clean complete"

# Database migrations (requires GOOSE or similar)
migrate-up:
	@echo "Running database migrations..."
	# Add your migration command here

migrate-down:
	@echo "Rolling back database migrations..."
	# Add your migration rollback command here
