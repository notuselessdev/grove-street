BIN       := grove-street
BIN_DIR   := $(HOME)/bin
DATA_DIR  := $(HOME)/.grove-street
OVERLAY   := $(DATA_DIR)/grove-notify
SWIFT_SRC := scripts/grove-notify.swift

.PHONY: build install overlay dev test lint

## build: compile the Go binary to ./grove-street
build:
	go build -o $(BIN) ./cmd/grove-street

## install: build and install binary to ~/bin
install: build
	cp $(BIN) $(BIN_DIR)/$(BIN)
	@echo "Installed to $(BIN_DIR)/$(BIN)"

## overlay: compile and sign the macOS notification overlay
overlay:
	swiftc -O -o $(OVERLAY) $(SWIFT_SRC) -framework Cocoa
	@identity=$$(security find-identity -v -p codesigning 2>/dev/null | awk '/Apple Development:/{print $$2; exit}'); \
	if [ -n "$$identity" ]; then \
		codesign --force --sign "$$identity" $(OVERLAY); \
	else \
		codesign --sign - --force $(OVERLAY); \
	fi
	@echo "Overlay installed and signed at $(OVERLAY)"

## dev: build, install binary, and recompile overlay (full local dev cycle)
dev: install overlay

## test: run all tests
test:
	go test ./...

## lint: vet the code
lint:
	go vet ./...
