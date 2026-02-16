#!/bin/sh
# scripts/completions.sh
set -eux

rm -rf completions
mkdir completions

for sh in bash zsh fish powershell; do
    ext="$sh"
    if [ "$sh" = "powershell" ]; then
        ext="ps1"
    fi
    go run main.go completion "$sh" >"completions/gup.$ext"
done
