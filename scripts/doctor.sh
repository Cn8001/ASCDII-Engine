#!/bin/sh
# doctor.sh checks what a make target needs before it starts, and installs it
# when asked. Someone new to the repository runs plain make, which runs the
# "guide" mode: it either lets the build go ahead or stops with a numbered
# list of what to do next. Required versions are read from go.mod and the
# frontend's dependencies, so they cannot drift from the code.
#
#   scripts/doctor.sh guide                 what plain make runs
#   scripts/doctor.sh deps                  install what is missing (make deps)
#   scripts/doctor.sh cli|editor-linux|editor-windows|editor-dev|version|all|dist
#   scripts/doctor.sh runtime               what running games need
#
# Exit status is 1 when anything required is missing.

set -u
cd "$(dirname "$0")/.." || exit 1

mode=${1:-guide}
fail=0
quiet=0

# What the next-steps list and make deps act on.
want_pkgs=0
want_go=0
want_node=0
want_wails=0
want_python=0
want_linux=0

ok() { [ "$quiet" -eq 1 ] || printf '  ok       %s\n' "$1"; }
bad() {
	printf '  MISSING  %s\n' "$1"
	shift
	for line in "$@"; do printf '           %s\n' "$line"; done
	fail=1
}
note() {
	printf '  note     %s\n' "$1"
	shift
	for line in "$@"; do printf '           %s\n' "$line"; done
}
head_line() { [ "$quiet" -eq 1 ] || echo "$1"; }
has() { command -v "$1" >/dev/null 2>&1; }

# version_ge A B: is version A at least B? sort -V knows 1.10 > 1.9.
version_ge() {
	[ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n 1)" = "$2" ]
}

host() {
	case "$(uname -s 2>/dev/null)" in
	Linux) echo linux ;;
	Darwin) echo darwin ;;
	MINGW* | MSYS* | CYGWIN*) echo windows ;;
	*) echo unknown ;;
	esac
}

distro() {
	[ -r /etc/os-release ] || { echo unknown; return; }
	# shellcheck disable=SC1091
	. /etc/os-release
	case " ${ID:-} ${ID_LIKE:-} " in
	*" debian "* | *" ubuntu "*) echo debian ;;
	*" fedora "* | *" rhel "*) echo fedora ;;
	*" arch "*) echo arch ;;
	*" opensuse"* | *" suse "*) echo suse ;;
	*) echo unknown ;;
	esac
}

distro_name() {
	[ -r /etc/os-release ] || { host; return; }
	# shellcheck disable=SC1091
	. /etc/os-release
	echo "${PRETTY_NAME:-${NAME:-linux}}"
}

go_min=$(sed -n 's/^go \([0-9.]*\).*/\1/p' go.mod)
wails_want=$(sed -n 's|.*github.com/wailsapp/wails/v2 \(v[^ ]*\).*|\1|p' go.mod)

# Every system package any Linux target needs, per distribution: a compiler
# and pkg-config for cgo, GTK 3 and WebKitGTK 4.1 for the editor (Wails v2
# built with -tags webkit2_41), the X11 libraries Ebitengine's GLFW is built
# against for the window player, and python3 for make version.
linux_packages() {
	case $(distro) in
	debian) echo "build-essential pkg-config python3 libgtk-3-dev libwebkit2gtk-4.1-dev libx11-dev libxrandr-dev libxinerama-dev libxi-dev libxrender-dev libxcursor-dev" ;;
	fedora) echo "gcc pkgconf-pkg-config python3 gtk3-devel webkit2gtk4.1-devel libX11-devel libXrandr-devel libXinerama-devel libXi-devel libXrender-devel libXcursor-devel" ;;
	arch) echo "base-devel pkgconf python gtk3 webkit2gtk-4.1 libx11 libxrandr libxinerama libxi libxrender libxcursor" ;;
	suse) echo "gcc pkg-config python3 gtk3-devel webkit2gtk3-devel libX11-devel libXrandr-devel libXinerama-devel libXi-devel libXrender-devel libXcursor-devel" ;;
	esac
}

install_command() {
	pkgs=$(linux_packages)
	[ -n "$pkgs" ] || return 1
	case $(distro) in
	debian) echo "sudo apt-get update && sudo apt-get install -y $pkgs" ;;
	fedora) echo "sudo dnf install -y $pkgs" ;;
	arch) echo "sudo pacman -S --needed --noconfirm $pkgs" ;;
	suse) echo "sudo zypper install -y $pkgs" ;;
	esac
}

# Where the package manager's Go and Node are new enough to use. Debian,
# Ubuntu, Fedora and openSUSE ship versions older than this module needs, so
# there the official installers are the way.
manager_has_current_toolchain() {
	case $(host) in
	darwin) has brew ;;
	windows) has winget ;;
	linux) [ "$(distro)" = arch ] ;;
	*) return 1 ;;
	esac
}

# Vite refuses older Node; its engines field is "^20.19.0 || >=22.12.0".
node_ok() {
	v=$1
	major=${v%%.*}
	if [ "$major" -eq 20 ]; then version_ge "$v" 20.19.0
	elif [ "$major" -eq 22 ]; then version_ge "$v" 22.12.0
	else [ "$major" -gt 22 ]
	fi
}

# Each tool is checked once however many sections need it, so a missing one
# is reported once.
done_go=0
done_editor=0
done_cgo=0
done_pkgconfig=0

check_go() {
	[ "$done_go" -eq 0 ] || return 0
	done_go=1
	if has go; then
		v=$(go env GOVERSION 2>/dev/null | sed 's/^go//')
		if version_ge "$v" "$go_min"; then ok "go $v"
		else
			want_go=1
			bad "go $v is older than $go_min (go.mod)" "Install Go $go_min or newer: https://go.dev/dl/"
		fi
	else
		want_go=1
		bad "go" "Install Go $go_min or newer: https://go.dev/dl/"
	fi
}

check_editor_tools() {
	[ "$done_editor" -eq 0 ] || return 0
	done_editor=1
	if has node; then
		v=$(node --version | sed 's/^v//')
		if node_ok "$v"; then ok "node $v"
		else
			want_node=1
			bad "node $v is too old for Vite (needs 20.19+ or 22.12+)" "Install Node 22 LTS: https://nodejs.org/"
		fi
	else
		want_node=1
		bad "node" "Install Node 22 LTS: https://nodejs.org/"
	fi
	if has npm; then ok "npm $(npm --version 2>/dev/null)"
	else
		want_node=1
		bad "npm" "It comes with Node: https://nodejs.org/"
	fi
	if has wails; then
		# wails prints its banner around the version, partly on stderr.
		v=$(wails version 2>&1 | grep -o 'v2\.[0-9][0-9.]*' | head -n 1)
		if [ "$v" = "$wails_want" ]; then ok "wails $v"
		else
			note "wails is ${v:-of unknown version}, go.mod uses $wails_want" \
				"Builds usually work, but bindings may differ. To match: make tools"
		fi
	else
		want_wails=1
		bad "wails $wails_want" "make tools installs it"
	fi
}

check_python() {
	if has python3; then ok "python3 $(python3 --version 2>&1 | cut -d' ' -f2)"
	else
		want_python=1
		bad "python3 (writes the docs and the version into files)" "make deps installs it"
	fi
}

# check_cgo_linux WHAT MODULES...: the Linux programs that use cgo can only
# be built on Linux, where they need a compiler and these libraries.
check_cgo_linux() {
	what=$1
	shift
	if [ "$(host)" != linux ]; then
		want_linux=1
		bad "a Linux machine for the $what" "It uses cgo, which does not cross-compile from $(host); build it on Linux"
		return
	fi
	if [ "$done_cgo" -eq 0 ]; then
		done_cgo=1
		if has gcc || has clang || has cc; then ok "C compiler"
		else
			want_pkgs=1
			bad "a C compiler (cgo needs one)" "make deps installs it"
		fi
	fi
	if ! has pkg-config; then
		if [ "$done_pkgconfig" -eq 0 ]; then
			done_pkgconfig=1
			want_pkgs=1
			bad "pkg-config" "make deps installs it"
		fi
		return
	fi
	for m in "$@"; do
		if pkg-config --exists "$m"; then ok "$m $(pkg-config --modversion "$m")"
		else
			want_pkgs=1
			bad "$m development files (for the $what)" "make deps installs them"
		fi
	done
}

section_cli() {
	head_line "CLI and players (make cli)"
	check_go
	check_cgo_linux "window player" x11 xrandr xinerama xi xrender xcursor
}

section_editor_linux() {
	head_line "Linux editor (make editor-linux)"
	check_go
	check_editor_tools
	check_cgo_linux "Linux editor" gtk+-3.0 webkit2gtk-4.1
}

section_editor_windows() {
	head_line "Windows editor (make editor-windows)"
	check_go
	check_editor_tools
}

# A built game finds its sound and OpenGL libraries when it starts, so none
# of this stops a build.
section_runtime() {
	echo "Needed where games run (optional for building)"
	if [ "$(host)" = linux ]; then
		if ldconfig -p 2>/dev/null | grep -q 'libasound.so.2'; then ok "libasound.so.2 (sound)"
		else note "libasound.so.2 not found" "Games play without sound. Debian/Ubuntu: sudo apt install libasound2t64 (or libasound2)"
		fi
		if ldconfig -p 2>/dev/null | grep -q 'libGL.so.1'; then ok "libGL.so.1 (window player)"
		else note "libGL.so.1 not found" "The window player cannot open. Debian/Ubuntu: sudo apt install libgl1"
		fi
	else
		ok "nothing extra on $(host)"
	fi
	echo "  note     The machine that runs ascdii-editor.exe needs the WebView2 runtime;"
	echo "           Windows 10 and 11 ship it."
}

# What plain make builds on this machine; the guide checks exactly that.
section_host() {
	case $(host) in
	linux)
		section_cli
		section_editor_linux
		;;
	*) section_editor_windows ;;
	esac
}

# next_steps turns what is missing into a numbered list, in the order the
# steps depend on each other: system packages and Go first, then wails.
next_steps() {
	echo
	echo "What to do next ($(distro_name)):"
	n=1
	step() {
		printf '  %d. %s\n' "$n" "$1"
		shift
		for line in "$@"; do printf '     %s\n' "$line"; done
		n=$((n + 1))
	}
	if [ "$want_linux" -eq 1 ]; then
		step "Build on Linux" "The CLI, the players and the Linux editor use cgo and only build there." \
			"On this machine: make editor-windows"
	fi
	if [ "$want_pkgs" -eq 1 ] || [ "$want_python" -eq 1 ]; then
		if cmd=$(install_command); then
			step "make deps" "Installs the system packages (asks for your sudo password):" "$cmd"
		else
			step "Install a C compiler, pkg-config, python3 and the development files of" \
				"gtk+-3.0, webkit2gtk-4.1, x11, xrandr, xinerama, xi, xrender and xcursor" \
				"(this distribution is not one make deps knows)"
		fi
	fi
	if [ "$want_go" -eq 1 ]; then
		if manager_has_current_toolchain; then step "make deps" "Also installs Go $go_min or newer."
		else
			step "Install Go $go_min or newer from https://go.dev/dl/" \
				"The distribution's own Go package is older than go.mod needs." \
				"Linux: sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go*.linux-amd64.tar.gz" \
				"then add /usr/local/go/bin and \$HOME/go/bin to PATH"
		fi
	fi
	if [ "$want_node" -eq 1 ]; then
		if manager_has_current_toolchain; then step "make deps" "Also installs Node."
		else
			step "Install Node 22 LTS" "https://nodejs.org/ or, without sudo, nvm: https://github.com/nvm-sh/nvm" \
				"nvm install 22"
		fi
	fi
	if [ "$want_wails" -eq 1 ]; then
		step "make tools" "Installs wails $wails_want with go install (needs Go first)." \
			"Then make sure \$(go env GOPATH)/bin is on PATH."
	fi
	echo "  Then run make again."
}

# install_deps is make deps: everything a package manager can do here, then
# the Go tools. Each step says what it runs before running it.
install_deps() {
	case $(host) in
	linux)
		if cmd=$(install_command); then
			echo "Installing system packages for $(distro_name):"
			echo "  $cmd"
			sh -c "$cmd" || return 1
			if [ "$(distro)" = arch ]; then
				echo "  sudo pacman -S --needed --noconfirm go nodejs npm"
				sudo pacman -S --needed --noconfirm go nodejs npm || return 1
			fi
		else
			echo "make deps does not know $(distro_name)."
			echo "Install a C compiler, pkg-config, python3 and the development files of"
			echo "gtk+-3.0, webkit2gtk-4.1, x11, xrandr, xinerama, xi, xrender and xcursor."
			return 1
		fi
		;;
	darwin)
		if ! xcode-select -p >/dev/null 2>&1; then
			echo "xcode-select --install"
			xcode-select --install
		fi
		if has brew; then
			echo "brew install go node python"
			brew install go node python || return 1
		else
			echo "Homebrew not found; install Go, Node and Python from their sites, or Homebrew from https://brew.sh/"
		fi
		;;
	windows)
		if has winget; then
			echo "winget install GoLang.Go OpenJS.NodeJS.LTS Python.Python.3.12"
			winget install --silent --accept-source-agreements --accept-package-agreements GoLang.Go OpenJS.NodeJS.LTS Python.Python.3.12
			echo "Open a new terminal so PATH picks the new tools up."
		else
			echo "winget not found; install Go, Node and Python from their sites."
		fi
		;;
	esac
	if has go; then
		echo "go install github.com/wailsapp/wails/v2/cmd/wails@$wails_want"
		go install "github.com/wailsapp/wails/v2/cmd/wails@$wails_want" || return 1
		gobin=$(go env GOPATH)/bin
		case ":$PATH:" in
		*":$gobin:"*) ;;
		*) echo "Add $gobin to PATH so wails can be found." ;;
		esac
	else
		echo "Go is still missing, so wails was not installed. Install Go, then: make tools"
	fi
}

case $mode in
guide)
	# Silent when the build can go ahead; otherwise only what is missing,
	# then the steps.
	quiet=1
	section_host
	if [ "$fail" -ne 0 ]; then
		next_steps
		exit 1
	fi
	exit 0
	;;
deps)
	install_deps || exit 1
	echo
	quiet=1
	section_host
	if [ "$fail" -ne 0 ]; then
		next_steps
		exit 1
	fi
	echo "Everything the build needs is installed. Run: make"
	exit 0
	;;
cli) section_cli ;;
editor-linux) section_editor_linux ;;
editor-windows) section_editor_windows ;;
editor-dev)
	section_cli
	echo
	section_editor_linux
	;;
version)
	head_line "Docs and version (make version)"
	check_go
	check_python
	;;
all | dist)
	section_cli
	echo
	section_editor_linux
	echo
	section_editor_windows
	echo
	head_line "Docs and version (make version)"
	check_python
	if [ "$mode" = dist ]; then
		if has tar; then ok "tar"; else bad "tar" "Install tar from your package manager"; fi
	fi
	;;
runtime) section_runtime ;;
*)
	echo "usage: $0 guide|deps|cli|editor-linux|editor-windows|editor-dev|version|all|dist|runtime" >&2
	exit 2
	;;
esac

if [ "$fail" -ne 0 ]; then
	next_steps
	exit 1
fi
