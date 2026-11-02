#!/bin/sh
# doctor.sh checks what a make target needs before it starts, so a missing
# tool is named together with how to install it, instead of surfacing as a
# compiler error halfway through. Required versions are read from go.mod and
# the frontend's dependencies, so they cannot drift from the code.
#
#   scripts/doctor.sh cli|editor-linux|editor-windows|editor-dev|version|all|dist|runtime
#
# Exit status is 1 when anything the target needs is missing. What only a
# running game needs is printed as a note.

set -u
cd "$(dirname "$0")/.." || exit 1

target=${1:-all}
fail=0

ok() { printf '  ok       %s\n' "$1"; }
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

go_min=$(sed -n 's/^go \([0-9.]*\).*/\1/p' go.mod)
wails_want=$(sed -n 's|.*github.com/wailsapp/wails/v2 \(v[^ ]*\).*|\1|p' go.mod)

# Vite refuses older Node; its engines field is "^20.19.0 || >=22.12.0".
node_ok() {
	v=$1
	major=${v%%.*}
	if [ "$major" -eq 20 ]; then version_ge "$v" 20.19.0
	elif [ "$major" -eq 22 ]; then version_ge "$v" 22.12.0
	else [ "$major" -gt 22 ]
	fi
}

check_go() {
	if has go; then
		v=$(go env GOVERSION 2>/dev/null | sed 's/^go//')
		if version_ge "$v" "$go_min"; then ok "go $v"
		else bad "go $v is older than $go_min (go.mod)" "Install Go $go_min or newer: https://go.dev/dl/"
		fi
	else
		bad "go" "Install Go $go_min or newer: https://go.dev/dl/"
	fi
}

check_editor_tools() {
	if has node; then
		v=$(node --version | sed 's/^v//')
		if node_ok "$v"; then ok "node $v"
		else bad "node $v is too old for Vite (needs 20.19+ or 22.12+)" "Install Node 22 LTS: https://nodejs.org/"
		fi
	else
		bad "node" "Install Node 22 LTS: https://nodejs.org/"
	fi
	if has npm; then ok "npm $(npm --version 2>/dev/null)"
	else bad "npm" "It comes with Node: https://nodejs.org/"
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
		bad "wails $wails_want" "Install it: make tools" \
			"(go install github.com/wailsapp/wails/v2/cmd/wails@$wails_want, then put \$(go env GOPATH)/bin on PATH)"
	fi
}

check_python() {
	if has python3; then ok "python3 $(python3 --version 2>&1 | cut -d' ' -f2)"
	else bad "python3 (writes the docs and the version into files)" "Install Python 3: https://www.python.org/downloads/"
	fi
}

# The editor is Wails v2 built with -tags webkit2_41, so WebKitGTK 4.1.
editor_packages() {
	case $(distro) in
	debian) echo "sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev" ;;
	fedora) echo "sudo dnf install gcc pkgconf-pkg-config gtk3-devel webkit2gtk4.1-devel" ;;
	arch) echo "sudo pacman -S --needed base-devel gtk3 webkit2gtk-4.1" ;;
	suse) echo "sudo zypper install gcc pkg-config gtk3-devel webkit2gtk3-devel" ;;
	*) echo "Install a C compiler, pkg-config and the development files of gtk+-3.0 and webkit2gtk-4.1" ;;
	esac
}

# The window player is Ebitengine, whose GLFW is compiled against X11.
window_packages() {
	case $(distro) in
	debian) echo "sudo apt install build-essential pkg-config libx11-dev libxrandr-dev libxinerama-dev libxi-dev libxrender-dev libxcursor-dev" ;;
	fedora) echo "sudo dnf install gcc pkgconf-pkg-config libX11-devel libXrandr-devel libXinerama-devel libXi-devel libXrender-devel libXcursor-devel" ;;
	arch) echo "sudo pacman -S --needed base-devel libx11 libxrandr libxinerama libxi libxrender libxcursor" ;;
	suse) echo "sudo zypper install gcc pkg-config libX11-devel libXrandr-devel libXinerama-devel libXi-devel libXrender-devel libXcursor-devel" ;;
	*) echo "Install a C compiler, pkg-config and the X11 development files: x11 xrandr xinerama xi xrender xcursor" ;;
	esac
}

# check_cgo_linux WHAT INSTALL MODULES...: the Linux programs that use cgo
# can only be built on Linux, where they need a compiler and these libraries.
check_cgo_linux() {
	what=$1
	install=$2
	shift 2
	if [ "$(host)" != linux ]; then
		bad "a Linux machine for the $what" "It uses cgo, which does not cross-compile from $(host); build it on Linux"
		return
	fi
	if has gcc || has clang || has cc; then ok "C compiler"
	else bad "a C compiler (the $what uses cgo)" "$install"
	fi
	if ! has pkg-config; then
		bad "pkg-config" "$install"
		return
	fi
	for m in "$@"; do
		if pkg-config --exists "$m"; then ok "$m $(pkg-config --modversion "$m")"
		else bad "$m development files (for the $what)" "$install"
		fi
	done
}

section_cli() {
	echo "CLI and players (make cli)"
	check_go
	check_cgo_linux "window player" "$(window_packages)" x11 xrandr xinerama xi xrender xcursor
	ok "Windows programs need no C compiler (cross-compiled without cgo)"
}

section_editor_linux() {
	echo "Linux editor (make editor-linux)"
	check_go
	check_editor_tools
	check_cgo_linux "Linux editor" "$(editor_packages)" gtk+-3.0 webkit2gtk-4.1
}

section_editor_windows() {
	echo "Windows editor (make editor-windows)"
	check_go
	check_editor_tools
	note "The machine that runs ascdii-editor.exe needs the WebView2 runtime." \
		"Windows 10 and 11 ship it; older systems: https://developer.microsoft.com/microsoft-edge/webview2/"
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
}

case $target in
cli) section_cli ;;
editor-linux) section_editor_linux ;;
editor-windows) section_editor_windows ;;
editor-dev)
	section_cli
	echo
	section_editor_linux
	;;
all | dist)
	section_cli
	echo
	section_editor_linux
	echo
	section_editor_windows
	echo
	echo "Docs and version (make version)"
	check_python
	if [ "$target" = dist ]; then
		if has tar; then ok "tar"; else bad "tar" "Install tar from your package manager"; fi
	fi
	;;
version)
	echo "Docs and version (make version)"
	check_go
	check_python
	;;
runtime) section_runtime ;;
*)
	echo "usage: $0 cli|editor-linux|editor-windows|editor-dev|version|all|dist|runtime" >&2
	exit 2
	;;
esac

if [ "$fail" -ne 0 ]; then
	echo
	echo "Install what is MISSING above, then run make again."
	exit 1
fi
