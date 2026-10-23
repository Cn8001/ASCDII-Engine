BIN      := bin
EDITOR   := cmd/editor
WAILS    := wails
LINUXTAG := webkit2_41

.PHONY: test bench template brand cli players hello dragonfall editor-dev editor-linux editor-windows dist clean

test:
	go vet ./...
	go test -race ./...

bench:
	go test -run '^$$' -bench . ./internal/dragonfall/

template:
	go run ./internal/dragonfall/gen

brand:
	go run ./cmd/brand

cli:
	go build -o $(BIN)/ascdii ./cmd/ascdii

# Players are ascdii per platform; builds append the game to them. The editor
# uses the ascdii beside it for its own platform and these for the others.
players:
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o $(BIN)/ascdii-linux-amd64 ./cmd/ascdii
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o $(BIN)/ascdii-windows-amd64.exe ./cmd/ascdii

hello:
	go run ./cmd/hello

dragonfall:
	go run ./cmd/ascdii run internal/dragonfall/template

editor-dev:
	cd $(EDITOR) && ASCDII_PLAYERS=$(CURDIR)/$(BIN) WAILS_DEV_ALLOWED_HOSTS=wails.localhost $(WAILS) dev -tags $(LINUXTAG)

# The editor lands next to bin/ascdii, where the launcher looks for it.
editor-linux:
	cd $(EDITOR) && $(WAILS) build -tags $(LINUXTAG)
	cp $(EDITOR)/build/bin/ascdii-editor $(BIN)/

editor-windows:
	cd $(EDITOR) && $(WAILS) build -platform windows/amd64
	cp $(EDITOR)/build/bin/ascdii-editor.exe $(BIN)/

# dist packs what someone without the source needs: the editor and the
# ascdii command, one archive per platform.
VERSION = $(shell go run ./cmd/ascdii version | cut -d' ' -f3)
# Each archive also carries the other platform's player, so either editor
# builds games for both.
dist: players editor-linux editor-windows
	rm -rf $(BIN)/dist && mkdir -p $(BIN)/dist/ascdii-$(VERSION)-linux-amd64 $(BIN)/dist/ascdii-$(VERSION)-windows-amd64
	cp $(EDITOR)/build/bin/ascdii-editor $(BIN)/ascdii-windows-amd64.exe $(BIN)/dist/ascdii-$(VERSION)-linux-amd64/
	cp $(BIN)/ascdii-linux-amd64 $(BIN)/dist/ascdii-$(VERSION)-linux-amd64/ascdii
	cp $(EDITOR)/build/bin/ascdii-editor.exe $(BIN)/ascdii-linux-amd64 $(BIN)/dist/ascdii-$(VERSION)-windows-amd64/
	cp $(BIN)/ascdii-windows-amd64.exe $(BIN)/dist/ascdii-$(VERSION)-windows-amd64/ascdii.exe
	cd $(BIN)/dist && tar czf ascdii-$(VERSION)-linux-amd64.tar.gz ascdii-$(VERSION)-linux-amd64
	cd $(BIN)/dist && python3 -m zipfile -c ascdii-$(VERSION)-windows-amd64.zip ascdii-$(VERSION)-windows-amd64

clean:
	rm -rf $(BIN) $(EDITOR)/build/bin
