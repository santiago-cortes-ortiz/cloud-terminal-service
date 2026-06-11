#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-ShAd0W20/aws-terminal}"
BINARY="aws-terminal"
INSTALL_DIR="${INSTALL_DIR:-}"
VERSION="${VERSION:-latest}"

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

download() {
  local url="$1"
  local output="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$output"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$output"
  else
    fail "required command not found: curl or wget"
  fi
}

normalize_os() {
  case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
    darwin) printf 'darwin' ;;
    linux) printf 'linux' ;;
    *) fail "unsupported OS: $(uname -s)" ;;
  esac
}

normalize_arch() {
  case "$(uname -m | tr '[:upper:]' '[:lower:]')" in
    x86_64|amd64) printf 'amd64' ;;
    arm64|aarch64) printf 'arm64' ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
  esac
}

latest_version() {
  local metadata="$1"
  download "https://api.github.com/repos/${REPO}/releases/latest" "$metadata"
  sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$metadata" | head -n 1
}

sha256_file() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
  else
    fail "required command not found: sha256sum or shasum"
  fi
}

select_install_dir() {
  if [[ -n "$INSTALL_DIR" ]]; then
    printf '%s' "$INSTALL_DIR"
    return
  fi

  if [[ -d /usr/local/bin && -w /usr/local/bin ]]; then
    printf '/usr/local/bin'
    return
  fi

  printf '%s/.local/bin' "$HOME"
}

os="$(normalize_os)"
arch="$(normalize_arch)"

if [[ "$os" == "linux" && "$arch" != "amd64" ]]; then
  fail "unsupported target: ${os}/${arch}; release assets are currently published for linux/amd64 only"
fi

need_cmd awk
need_cmd grep
need_cmd install
need_cmd mkdir
need_cmd sed
need_cmd tar

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

if [[ "$VERSION" == "latest" ]]; then
  VERSION="$(latest_version "$tmpdir/release.json")"
  [[ -n "$VERSION" ]] || fail "could not determine latest release version"
fi

asset="${BINARY}_${VERSION}_${os}_${arch}.tar.gz"
base_url="https://github.com/${REPO}/releases/download/${VERSION}"
archive="$tmpdir/$asset"
checksums="$tmpdir/checksums.txt"

log "Installing ${BINARY} ${VERSION} for ${os}/${arch}..."
download "${base_url}/${asset}" "$archive"
download "${base_url}/checksums.txt" "$checksums"

expected="$(awk -v asset="$asset" '$2 == asset { print $1; exit }' "$checksums")"
[[ -n "$expected" ]] || fail "checksum for ${asset} not found"
actual="$(sha256_file "$archive")"
[[ "$actual" == "$expected" ]] || fail "checksum mismatch for ${asset}"

mkdir -p "$tmpdir/extract"
tar -xzf "$archive" -C "$tmpdir/extract"
[[ -f "$tmpdir/extract/$BINARY" ]] || fail "archive did not contain ${BINARY}"

install_dir="$(select_install_dir)"
mkdir -p "$install_dir"
install -m 0755 "$tmpdir/extract/$BINARY" "$install_dir/$BINARY"

log "Installed ${BINARY} to ${install_dir}/${BINARY}"
case ":$PATH:" in
  *":${install_dir}:"*) ;;
  *) log "Note: ${install_dir} is not on your PATH." ;;
esac
