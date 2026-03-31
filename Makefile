VERSION ?= $(shell tag=$$(git describe --tags --exact-match 2>/dev/null || true); if [ -n "$$tag" ]; then echo "$$tag"; else echo dev; fi)

.PHONY: build test vet clean

build:
	go build -ldflags "-X main.version=$(VERSION)" -o codex-switch ./cmd/codex-switch

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f codex-switch
