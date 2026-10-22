#!/bin/sh
set -eu

tag=$1
archive=$2
tap=$3
version=${tag#v}
if ! printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "Expected a stable appsum release tag, got: $tag" >&2
  exit 1
fi

# Earlier releases can overwrite Homebrew's Cellar during --upgrade.
if [ "$(printf '%s\n' 0.1.2 "$version" | sort -V | tail -1)" != "$version" ]; then
  echo "Skipping $tag: Homebrew support starts with v0.1.2."
  exit 0
fi

formula=$tap/Formula/appsum.rb
if [ -f "$formula" ]; then
  current=$(awk -F '"' '/^  version / { print $2 }' "$formula")
  if ! printf '%s\n' "$current" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "Cannot determine the current Homebrew formula version." >&2
    exit 1
  fi
  if [ "$(printf '%s\n' "$current" "$version" | sort -V | tail -1)" != "$version" ]; then
    echo "Skipping $tag: the tap already has appsum $current."
    exit 0
  fi
fi

checksum=$(shasum -a 256 "$archive")
checksum=${checksum%% *}
mkdir -p "$tap/Formula"
cat > "$formula" <<EOF
class Appsum < Formula
  desc "CPU and memory usage for apps and process trees"
  homepage "https://github.com/sadiksaifi/appsum"
  url "https://github.com/sadiksaifi/appsum/archive/refs/tags/$tag.tar.gz"
  version "$version"
  sha256 "$checksum"

  depends_on "go" => :build

  def install
    ENV["CGO_ENABLED"] = "0"
    ldflags = "-s -w -X github.com/sadiksaifi/appsum/internal/app.version=v#{version}"
    system "go", "build", *std_go_args(ldflags:), "./cmd/appsum"
  end

  test do
    assert_match "v#{version}", shell_output("#{bin}/appsum --version")
    report = JSON.parse(shell_output("#{bin}/appsum #{Process.pid} --json --interval 100ms"))
    assert_equal Process.pid, report.fetch("groups").first.fetch("processes").first.fetch("pid")
  end
end
EOF
