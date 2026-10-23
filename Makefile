BIN      := bin
PLAYERS  := $(BIN)/players
EDITOR   := cmd/editor
WAILS    := wails
LINUXTAG := webkit2_41

.PHONY: test bench template brand cli players hello dragonfall editor-dev editor-linux editor-windows clean

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
	go build -o $(BIN)/launcher ./cmd/launcher

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
	cd $(EDITOR) && $(WAILS) build -tags $(LINUXTAG) -clean

editor-windows: players
	cd $(EDITOR) && $(WAILS) build -platform windows/amd64 -clean

clean:
	rm -rf $(BIN) $(EDITOR)/build/bin
