VERSION ?= dev

.PHONY: build test vet clean

build:
	go build -ldflags "-X main.version=$(VERSION)" -o codex-switch ./cmd/codex-switch

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f codex-switch
