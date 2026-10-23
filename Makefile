BIN      := bin
EDITOR   := cmd/editor
WAILS    := wails
LINUXTAG := webkit2_41

.PHONY: all test bench template brand cli hello dragonfall editor-dev editor-linux editor-windows dist clean

test:
	go vet ./...
	go test -race ./...

bench:
	go test -run '^$$' -bench . ./internal/dragonfall/

template:
	go run ./internal/dragonfall/gen

brand:
	go run ./cmd/brand

# bin/linux and bin/windows are each a complete install holding only that
# platform's programs: copied to a machine of the platform they run as they
# are. ascdii opens the editor (and is the CLI), ascdii-editor is the editor,
# ascdii-player runs games and is what builds start from.
LINUX   := $(BIN)/linux
WINDOWS := $(BIN)/windows
GOREL   := go build -trimpath -ldflags "-s -w"

all: cli editor-linux editor-windows

# brand also writes the Windows icon resources go build links in.
cli: brand
	GOOS=linux GOARCH=amd64 $(GOREL) -o $(LINUX)/ascdii ./cmd/ascdii
	GOOS=linux GOARCH=amd64 $(GOREL) -o $(LINUX)/ascdii-player ./cmd/player
	GOOS=windows GOARCH=amd64 $(GOREL) -o $(WINDOWS)/ascdii.exe ./cmd/ascdii
	GOOS=windows GOARCH=amd64 $(GOREL) -o $(WINDOWS)/ascdii-player.exe ./cmd/player

hello:
	go run ./cmd/hello

dragonfall:
	go run ./cmd/ascdii play internal/dragonfall/template

editor-dev: cli
	cd $(EDITOR) && ASCDII_PLAYERS=$(CURDIR)/$(BIN) WAILS_DEV_ALLOWED_HOSTS=wails.localhost $(WAILS) dev -tags $(LINUXTAG)

# Wails always writes build/bin; the editor moves next to its ascdii, where the
# launcher looks for it.
editor-linux: brand
	cd $(EDITOR) && $(WAILS) build -tags $(LINUXTAG)
	mkdir -p $(LINUX) && mv $(EDITOR)/build/bin/ascdii-editor $(LINUX)/

editor-windows: brand
	cd $(EDITOR) && $(WAILS) build -platform windows/amd64
	mkdir -p $(WINDOWS) && mv $(EDITOR)/build/bin/ascdii-editor.exe $(WINDOWS)/

VERSION = $(shell go run ./cmd/ascdii version | cut -d' ' -f3)
# dist archives the two folders as they are.
dist: all
	rm -rf $(BIN)/dist && mkdir -p $(BIN)/dist
	cp -r $(LINUX) $(BIN)/dist/ascdii-$(VERSION)-linux-amd64
	cp -r $(WINDOWS) $(BIN)/dist/ascdii-$(VERSION)-windows-amd64
	cd $(BIN)/dist && tar czf ascdii-$(VERSION)-linux-amd64.tar.gz ascdii-$(VERSION)-linux-amd64
	cd $(BIN)/dist && python3 -m zipfile -c ascdii-$(VERSION)-windows-amd64.zip ascdii-$(VERSION)-windows-amd64

clean:
	rm -rf $(BIN)/* $(EDITOR)/build/bin
