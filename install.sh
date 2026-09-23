#!/bin/sh
set -eu

REPO="antihq/forja-cli"

install_dir="${FORJA_INSTALL_DIR:-$HOME/.local/bin}"

kernel="$(uname -s)"
case "$kernel" in
  MINGW* | MSYS* | CYGWIN*)
    echo "forja install: on Windows, download forja-windows-amd64.exe from https://github.com/${REPO}/releases" >&2
    exit 1
    ;;
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    echo "forja install: unsupported operating system: $kernel" >&2
    exit 1
    ;;
esac

machine="$(uname -m)"
case "$machine" in
  x86_64 | amd64) arch="amd64" ;;
  aarch64 | arm64) arch="arm64" ;;
  *)
    echo "forja install: unsupported architecture: $machine" >&2
    exit 1
    ;;
esac

if [ "$(id -u)" = "0" ] && [ -z "${FORJA_INSTALL_DIR:-}" ]; then
  install_dir="/usr/local/bin"
fi

asset="forja-${os}-${arch}"
url="https://github.com/${REPO}/releases/latest/download/${asset}"

mkdir -p "$install_dir"
echo "Installing forja (${os}/${arch}) to ${install_dir}/forja..."
curl -fsSL "$url" -o "${install_dir}/forja"
chmod +x "${install_dir}/forja"

case ":${PATH}:" in
  *":${install_dir}:"*) ;;
  *) echo "Add ${install_dir} to your PATH to start using forja." ;;
esac

"${install_dir}/forja" version
