#!/bin/sh
# docker-tui installer for Linux and macOS.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Akib558/docker-tui/main/scripts/install.sh | sh
#
# Environment overrides:
#   BIN_DIR    install directory (default: $HOME/.local/bin)
#   VERSION    release tag to install (default: latest)
set -eu

REPO="Akib558/docker-tui"
BIN="docker-tui"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-latest}"

info() { printf '\033[0;32m==>\033[0m %s\n' "$1"; }
err()  { printf '\033[0;31merror:\033[0m %s\n' "$1" >&2; exit 1; }

# --- detect OS -----------------------------------------------------------
os="$(uname -s)"
case "$os" in
  Linux)  os="linux" ;;
  Darwin) os="darwin" ;;
  *) err "unsupported OS '$os'. Try 'go install github.com/$REPO@latest' instead." ;;
esac

# --- detect arch ---------------------------------------------------------
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) err "unsupported architecture '$arch'. Try 'go install github.com/$REPO@latest' instead." ;;
esac

command -v curl >/dev/null 2>&1 || err "curl is required."
command -v tar  >/dev/null 2>&1 || err "tar is required."

# --- resolve the download URL from the GitHub API ------------------------
if [ "$VERSION" = "latest" ]; then
  api="https://api.github.com/repos/$REPO/releases/latest"
else
  api="https://api.github.com/repos/$REPO/releases/tags/$VERSION"
fi

info "Looking up the $VERSION release of $REPO..."
release="$(curl -fsSL "$api" 2>/dev/null || true)"
[ -n "$release" ] || err "no published release found yet. Install with Go instead:
    go install github.com/$REPO@latest"

url="$(printf '%s\n' "$release" \
  | grep -o '"browser_download_url": *"[^"]*"' \
  | sed 's/.*"browser_download_url": *"\([^"]*\)".*/\1/' \
  | grep -i "_${os}_${arch}\." \
  | head -n1)"
[ -n "$url" ] || err "no prebuilt binary for ${os}/${arch} in the latest release.
    Install with Go instead: go install github.com/$REPO@latest"

# --- download + extract --------------------------------------------------
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM
info "Downloading $(basename "$url")..."
curl -fsSL "$url" -o "$tmp/archive.tgz" || err "download failed."
tar -xzf "$tmp/archive.tgz" -C "$tmp" || err "extraction failed."

src="$(find "$tmp" -type f -name "$BIN" | head -n1)"
[ -n "$src" ] || err "binary '$BIN' not found in the archive."

# --- install -------------------------------------------------------------
mkdir -p "$BIN_DIR"
install -m 0755 "$src" "$BIN_DIR/$BIN" 2>/dev/null || { cp "$src" "$BIN_DIR/$BIN" && chmod 0755 "$BIN_DIR/$BIN"; }
info "Installed $BIN to $BIN_DIR/$BIN"

# --- PATH hint -----------------------------------------------------------
case ":$PATH:" in
  *":$BIN_DIR:"*) : ;;
  *) printf '\033[0;33mnote:\033[0m %s is not on your PATH. Add this to your shell profile:\n    export PATH="%s:$PATH"\n' "$BIN_DIR" "$BIN_DIR" ;;
esac

info "Done. Run '$BIN' to start (Docker daemon must be running)."
