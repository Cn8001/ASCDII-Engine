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

.PHONY: start build deps help all version test bench template brand cli hello dragonfall editor-dev editor-linux editor-windows dist clean doctor tools

ifeq ($(OS),Windows_NT)
HOST := windows
else
HOST := $(if $(filter Darwin,$(shell uname -s)),darwin,linux)
endif

# Plain make is the way in for someone new: it checks this machine, and
# either builds what the machine can build or stops with numbered steps
# (make deps, make tools, ...) that get it there.
.DEFAULT_GOAL := start
start:
	@sh scripts/doctor.sh guide
	@$(MAKE) --no-print-directory build
	@echo
	@echo "Built into $(BIN)/. Run the editor with: $(BIN)/$(HOST)/ascdii"

# build is what this machine can build: everything on Linux, the Windows
# editor elsewhere (the CLI and players use cgo for Linux).
ifeq ($(HOST),linux)
build: cli editor-linux
else
build: editor-windows
endif

# deps installs the system packages for this distribution (with sudo), Go
# and Node where the package manager has new enough ones, and wails.
deps:
	@sh scripts/doctor.sh deps

help:
	@echo "make              check this machine, then build what it can"
	@echo "make deps         install what is missing (system packages, wails)"
	@echo "make tools        install wails at the version go.mod uses"
	@echo "make doctor       list every requirement and what is missing"
	@echo "make test         vet and run the tests"
	@echo "make cli          CLI and players for Linux and Windows"
	@echo "make editor-linux, make editor-windows, make editor-dev"
	@echo "make all, make dist, make clean"

WAILS_VERSION := $(shell sed -n 's|.*github.com/wailsapp/wails/v2 \(v[^ ]*\).*|\1|p' go.mod)

# Each build first checks what it needs (scripts/doctor.sh). It is silent when
# all is there; otherwise it names what is missing with the command that
# installs it, and stops before the build starts.
doctor-%:
	@out=$$(sh scripts/doctor.sh $* 2>&1) || { printf '%s\n' "$$out"; exit 1; }

# doctor reports every target's requirements and what running games need.
doctor:
	-@sh scripts/doctor.sh all
	@echo
	@sh scripts/doctor.sh runtime

# tools installs the wails CLI at the version go.mod uses.
tools:
	go install github.com/wailsapp/wails/v2/cmd/wails@$(WAILS_VERSION)
	@echo "wails is in $$(go env GOPATH)/bin; that folder must be on PATH."

test:
	go vet ./...
	go vet -tags window ./internal/player ./cmd/player
	go test -race ./...

bench:
	go test -run '^$$' -bench . ./internal/dragonfall/

template:
	go run ./internal/dragonfall/gen

# The engine version is set only in internal/engine/runtime/version/version.go.
# This copies it to what cannot read Go: wails.json and the Windows resources
# (brand), the template manifest and the docs. The frontend reads it itself.
version: doctor-version brand template
	python3 docs/nav.py
	@if [ -f docs/internals/build.py ]; then python3 docs/internals/build.py; fi

brand:
	go run ./cmd/brand

# bin/linux and bin/windows are each a complete install holding only that
# platform's programs: copied to a machine of the platform they run as they
# are. ascdii opens the editor (and is the CLI), ascdii-editor is the editor,
# ascdii-player runs games and is what builds start from. ascdii-player-window
# is the player built with the window display (Ebitengine), which games shown
# in a window start from; the terminal programs do not link it. On Windows it
# is a GUI program, so it opens no console.
LINUX   := $(BIN)/linux
WINDOWS := $(BIN)/windows
GOREL   := go build -trimpath -ldflags "-s -w $(LDKEY)"
GOWIN   := go build -trimpath -tags window

all: doctor-all version cli editor-linux editor-windows

# brand also writes the Windows icon resources go build links in.
# Build lines stay quiet: printed, they would put the key into logs.
cli: doctor-cli brand
	@echo "build $(LINUX)/ascdii $(LINUX)/ascdii-player $(LINUX)/ascdii-player-window $(WINDOWS)/ascdii.exe $(WINDOWS)/ascdii-player.exe $(WINDOWS)/ascdii-player-window.exe"
	@GOOS=linux GOARCH=amd64 $(GOREL) -o $(LINUX)/ascdii ./cmd/ascdii
	@GOOS=linux GOARCH=amd64 $(GOREL) -o $(LINUX)/ascdii-player ./cmd/player
	@GOOS=linux GOARCH=amd64 $(GOWIN) -ldflags "-s -w $(LDKEY)" -o $(LINUX)/ascdii-player-window ./cmd/player
	@GOOS=windows GOARCH=amd64 $(GOREL) -o $(WINDOWS)/ascdii.exe ./cmd/ascdii
	@GOOS=windows GOARCH=amd64 $(GOREL) -o $(WINDOWS)/ascdii-player.exe ./cmd/player
	@GOOS=windows GOARCH=amd64 $(GOWIN) -ldflags "-s -w -H windowsgui $(LDKEY)" -o $(WINDOWS)/ascdii-player-window.exe ./cmd/player

hello:
	go run ./cmd/hello

dragonfall:
	go run ./cmd/ascdii play internal/dragonfall/template

editor-dev: doctor-editor-dev cli
	@cd $(EDITOR) && ASCDII_PLAYERS=$(CURDIR)/$(BIN) WAILS_DEV_ALLOWED_HOSTS=wails.localhost $(WAILS) dev -tags $(LINUXTAG) -ldflags "$(LDKEY)" 2>&1 | $(MASK)

# Wails always writes build/bin; the editor moves next to its ascdii, where
# the launcher looks for it. -trimpath keeps -ldflags, and so the key, out of
# the editor's build info.
editor-linux: doctor-editor-linux brand
	@cd $(EDITOR) && $(WAILS) build -trimpath -tags $(LINUXTAG) -ldflags "$(LDKEY)" > build.log 2>&1; s=$$?; $(MASK) build.log; rm -f build.log; exit $$s
	mkdir -p $(LINUX) && mv $(EDITOR)/build/bin/ascdii-editor $(LINUX)/

editor-windows: doctor-editor-windows brand
	@cd $(EDITOR) && $(WAILS) build -trimpath -platform windows/amd64 -ldflags "$(LDKEY)" > build.log 2>&1; s=$$?; $(MASK) build.log; rm -f build.log; exit $$s
	mkdir -p $(WINDOWS) && mv $(EDITOR)/build/bin/ascdii-editor.exe $(WINDOWS)/

VERSION = $(shell go run ./cmd/ascdii version | cut -d' ' -f3)
# dist archives the two folders as they are.
dist: doctor-dist all
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
