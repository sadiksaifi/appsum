#!/bin/sh
set -eu

script=$(dirname "$0")/update-homebrew.sh
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' 0
printf abc > "$tmp/source.tar.gz"

# Do not expose releases that can overwrite a Homebrew installation.
sh "$script" v0.1.1 "$tmp/source.tar.gz" "$tmp/tap"
test ! -e "$tmp/tap/Formula/appsum.rb"

# The first supported release creates the formula without a manual tap commit.
sh "$script" v0.1.2 "$tmp/source.tar.gz" "$tmp/tap"
formula=$tmp/tap/Formula/appsum.rb
grep -Fq 'url "https://github.com/sadiksaifi/appsum/archive/refs/tags/v0.1.2.tar.gz"' "$formula"
grep -Fq 'version "0.1.2"' "$formula"
grep -Fq 'sha256 "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"' "$formula"

# Retrying a historical release must not downgrade the tap.
sh "$script" v0.1.10 "$tmp/source.tar.gz" "$tmp/tap"
cp "$formula" "$tmp/expected.rb"
sh "$script" v0.1.9 "$tmp/source.tar.gz" "$tmp/tap"
cmp "$tmp/expected.rb" "$formula"
sh "$script" v0.1.10 "$tmp/source.tar.gz" "$tmp/tap"
cmp "$tmp/expected.rb" "$formula"

# A failed download or unrecognized formula must not overwrite the existing file.
if sh "$script" v0.1.11 "$tmp/missing.tar.gz" "$tmp/tap" 2>/dev/null; then
  echo 'Expected a missing source archive to fail.' >&2
  exit 1
fi
cmp "$tmp/expected.rb" "$formula"
printf 'not a versioned formula\n' > "$formula"
if sh "$script" v0.1.11 "$tmp/source.tar.gz" "$tmp/tap" 2>/dev/null; then
  echo 'Expected an unrecognized formula to fail.' >&2
  exit 1
fi
test "$(wc -l < "$formula" | tr -d ' ')" = 1

echo 'Homebrew formula tests passed.'
