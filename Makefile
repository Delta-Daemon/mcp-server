BINARY := deltadaemon-mcp
MODULE := github.com/Delta-Daemon/mcp-server
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build test install clean release

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) .

test:
	go test ./...

install: build
	install -d $(DESTDIR)$(HOME)/.local/bin
	install -m 0755 $(BINARY) $(HOME)/.local/bin/$(BINARY)

clean:
	rm -f $(BINARY)

release:
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY)_darwin_arm64 .
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY)_darwin_amd64 .
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY)_linux_amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY)_linux_arm64 .
