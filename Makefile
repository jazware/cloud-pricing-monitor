.PHONY: build test clean run docker-build docker-run fmt lint

# Binary name
BINARY_NAME=cloud-pricing-monitor
VERSION?=dev

# Build the application
build:
	go build -ldflags="-X 'main.version=$(VERSION)'" -o $(BINARY_NAME)

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

# Run the application
run:
	go run .

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Build Docker image
docker-build:
	docker build -t $(BINARY_NAME):$(VERSION) .

# Run Docker container
docker-run:
	docker run -p 9090:9090 \
		-e AWS_REGIONS=$(AWS_REGIONS) \
		-e AWS_INSTANCE_TYPES=$(AWS_INSTANCE_TYPES) \
		-e GCP_REGIONS=$(GCP_REGIONS) \
		-e GCP_INSTANCE_TYPES=$(GCP_INSTANCE_TYPES) \
		$(BINARY_NAME):$(VERSION)

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -ldflags="-X 'main.version=$(VERSION)'" -o $(BINARY_NAME)-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X 'main.version=$(VERSION)'" -o $(BINARY_NAME)-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -ldflags="-X 'main.version=$(VERSION)'" -o $(BINARY_NAME)-darwin-arm64
	GOOS=windows GOARCH=amd64 go build -ldflags="-X 'main.version=$(VERSION)'" -o $(BINARY_NAME)-windows-amd64.exe