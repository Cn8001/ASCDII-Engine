BIN      := bin
EDITOR   := cmd/editor
WAILS    := wails
LINUXTAG := webkit2_41

# The key that encrypts game data comes from .env, which git ignores, so the
# source opens no game: games open only in players built with the same key.
# The first make writes a random one; keep it private.
-include .env
# MASK blanks the key in the output of builds that print their flags, such
# as wails.
MASK := sed -u 's/pakSecret=[^ "]*/pakSecret=<.env>/g'

ifeq ($(ASCDII_PAK_SECRET),)
LDKEY :=
else
LDKEY := -X gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/bundle.pakSecret=$(ASCDII_PAK_SECRET)
endif

.PHONY: all version test bench template brand cli hello dragonfall editor-dev editor-linux editor-windows dist clean

test:
	go vet ./...
	go test -race ./...

bench:
	go test -run '^$$' -bench . ./internal/dragonfall/

template:
	go run ./internal/dragonfall/gen

# The engine version is set only in internal/engine/runtime/version/version.go.
# This copies it to what cannot read Go: wails.json and the Windows resources
# (brand), the template manifest and the docs. The frontend reads it itself.
version: brand
	python3 docs/nav.py
	@if [ -f docs/internals/build.py ]; then python3 docs/internals/build.py; fi

brand:
	go run ./cmd/brand

# bin/linux and bin/windows are each a complete install holding only that
# platform's programs: copied to a machine of the platform they run as they
# are. ascdii opens the editor (and is the CLI), ascdii-editor is the editor,
# ascdii-player runs games and is what builds start from.
LINUX   := $(BIN)/linux
WINDOWS := $(BIN)/windows
GOREL   := go build -trimpath -ldflags "-s -w $(LDKEY)"

all: version cli editor-linux editor-windows

# brand also writes the Windows icon resources go build links in.
# Build lines stay quiet: printed, they would put the key into logs.
cli: brand
	@echo "build $(LINUX)/ascdii $(LINUX)/ascdii-player $(WINDOWS)/ascdii.exe $(WINDOWS)/ascdii-player.exe"
	@GOOS=linux GOARCH=amd64 $(GOREL) -o $(LINUX)/ascdii ./cmd/ascdii
	@GOOS=linux GOARCH=amd64 $(GOREL) -o $(LINUX)/ascdii-player ./cmd/player
	@GOOS=windows GOARCH=amd64 $(GOREL) -o $(WINDOWS)/ascdii.exe ./cmd/ascdii
	@GOOS=windows GOARCH=amd64 $(GOREL) -o $(WINDOWS)/ascdii-player.exe ./cmd/player

hello:
	go run ./cmd/hello

dragonfall:
	go run ./cmd/ascdii play internal/dragonfall/template

editor-dev: cli
	@cd $(EDITOR) && ASCDII_PLAYERS=$(CURDIR)/$(BIN) WAILS_DEV_ALLOWED_HOSTS=wails.localhost $(WAILS) dev -tags $(LINUXTAG) -ldflags "$(LDKEY)" 2>&1 | $(MASK)

# Wails always writes build/bin; the editor moves next to its ascdii, where
# the launcher looks for it. -trimpath keeps -ldflags, and so the key, out of
# the editor's build info.
editor-linux: brand
	@cd $(EDITOR) && $(WAILS) build -trimpath -tags $(LINUXTAG) -ldflags "$(LDKEY)" > build.log 2>&1; s=$$?; $(MASK) build.log; rm -f build.log; exit $$s
	mkdir -p $(LINUX) && mv $(EDITOR)/build/bin/ascdii-editor $(LINUX)/

editor-windows: brand
	@cd $(EDITOR) && $(WAILS) build -trimpath -platform windows/amd64 -ldflags "$(LDKEY)" > build.log 2>&1; s=$$?; $(MASK) build.log; rm -f build.log; exit $$s
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

# make reads .env above and runs this first when it is missing.
.env:
	@echo "ASCDII_PAK_SECRET=$$(od -An -tx1 -N32 /dev/urandom | tr -d ' \n')" > $@
	@echo "wrote .env with a new game data key"
