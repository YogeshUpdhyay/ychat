OUT ?= bin/ypoker
RELAY_OUT ?= bin/relay
ICON ?= internal/ui/assets/yChat.png
CLIENT_PKG ?= ./cmd/client
RELAY_PKG ?= ./cmd/relay

.PHONY: build build-client build-relay client relay run dev test package-windows package-linux

build: build-client

build-client:
	@mkdir -p $(dir $(OUT))
	@go build -o $(OUT) $(CLIENT_PKG)

build-relay:
	@mkdir -p $(dir $(RELAY_OUT))
	@go build -o $(RELAY_OUT) $(RELAY_PKG)

client: build-client

relay: build-relay

run: build
	@./$(OUT)

dev:
	@air

test:
	go test ./...

package-windows:
	@mkdir -p dist
	@CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=amd64 fyne package -os windows -icon $(ICON)
	@mv *.exe dist/ || true

package-linux:
	@mkdir -p dist
	@fyne package -os linux -icon $(ICON)
	@mv *.tar.xz dist/ || true
