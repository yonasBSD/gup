#!/bin/sh
# scripts/completions.sh
set -eux

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

rm -rf completions
mkdir -p completions

tmp_bin="$(mktemp "${TMPDIR:-/tmp}/gup-completion.XXXXXX")"
trap 'rm -f "$tmp_bin"' EXIT

go build -o "$tmp_bin" .
chmod +x "$tmp_bin"

for sh in bash zsh fish powershell; do
    ext="$sh"
    if [ "$sh" = "powershell" ]; then
        ext="ps1"
    fi
    "$tmp_bin" completion "$sh" >"completions/gup.$ext"
done
