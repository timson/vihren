BINARY_NAME=vihren
DOCKER_IMAGE?=ghcr.io/timson/vihren
DOCKER_TAG?=latest
DOCKER_PLATFORMS?=linux/amd64,linux/arm64
GOLANGCI_LINT_VERSION?=v2.10.1

build: check
	go build -o ${BINARY_NAME} ./cmd/vihren

run: build
	./${BINARY_NAME}

check: lint test test-ui

lint: ensure-golangci-lint
	go vet ./...
	golangci-lint run

ensure-golangci-lint:
	@if ! golangci-lint version 2>/dev/null | grep -q "${GOLANGCI_LINT_VERSION}"; then \
		echo "Installing golangci-lint ${GOLANGCI_LINT_VERSION}..."; \
		curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $$(go env GOPATH)/bin ${GOLANGCI_LINT_VERSION}; \
	fi

test:
	go test ./...

test-ui:
	cd frontend && bun run test

docker: check
	docker buildx build --platform ${DOCKER_PLATFORMS} -t ${DOCKER_IMAGE}:${DOCKER_TAG} --push .

clean:
	rm -f ${BINARY_NAME}

.PHONY: build run check lint ensure-golangci-lint test test-ui docker clean
