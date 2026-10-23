BIN      := bin
PLAYERS  := $(BIN)/players
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

# Players are the ascdii binary per platform; builds and the editor append projects to them.
players:
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o $(PLAYERS)/ascdii-linux-amd64 ./cmd/ascdii
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o $(PLAYERS)/ascdii-windows-amd64.exe ./cmd/ascdii
	cp $(PLAYERS)/ascdii-linux-amd64 internal/editor/players/bin/ascdii-linux-amd64
	cp $(PLAYERS)/ascdii-windows-amd64.exe internal/editor/players/bin/ascdii-windows-amd64.exe

hello:
	go run ./cmd/hello

dragonfall:
	go run ./cmd/ascdii run internal/dragonfall/template

editor-dev: players
	cd $(EDITOR) && WAILS_DEV_ALLOWED_HOSTS=wails.localhost $(WAILS) dev -tags $(LINUXTAG)

editor-linux: players
	cd $(EDITOR) && $(WAILS) build -tags $(LINUXTAG)

editor-windows: players
	cd $(EDITOR) && $(WAILS) build -platform windows/amd64

# dist packs what someone without the source needs: the editor and the
# ascdii command, one archive per platform.
VERSION = $(shell go run ./cmd/ascdii version | cut -d' ' -f3)
dist: editor-linux editor-windows
	rm -rf $(BIN)/dist && mkdir -p $(BIN)/dist/ascdii-$(VERSION)-linux-amd64 $(BIN)/dist/ascdii-$(VERSION)-windows-amd64
	cp $(EDITOR)/build/bin/ascdii-editor $(PLAYERS)/ascdii-linux-amd64 $(BIN)/dist/ascdii-$(VERSION)-linux-amd64/
	mv $(BIN)/dist/ascdii-$(VERSION)-linux-amd64/ascdii-linux-amd64 $(BIN)/dist/ascdii-$(VERSION)-linux-amd64/ascdii
	cp $(EDITOR)/build/bin/ascdii-editor.exe $(BIN)/dist/ascdii-$(VERSION)-windows-amd64/
	cp $(PLAYERS)/ascdii-windows-amd64.exe $(BIN)/dist/ascdii-$(VERSION)-windows-amd64/ascdii.exe
	cd $(BIN)/dist && tar czf ascdii-$(VERSION)-linux-amd64.tar.gz ascdii-$(VERSION)-linux-amd64
	cd $(BIN)/dist && python3 -m zipfile -c ascdii-$(VERSION)-windows-amd64.zip ascdii-$(VERSION)-windows-amd64

clean:
	rm -rf $(BIN) $(EDITOR)/build/bin
