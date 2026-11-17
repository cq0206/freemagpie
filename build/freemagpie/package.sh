#!/usr/bin/env bash
# freemagpie: independent distribution; never dispatch upstream signing jobs.
set -euo pipefail
version=${1:?usage: package.sh VERSION macos|linux|windows}
target=${2:?missing target}
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'expected X.Y.Z' >&2; exit 1; }
ldflags="-s -w -X main.version=$version -X github.com/yetone/magpie/internal/update.defaultFeed=https://freemagpie.com/api/latest -X github.com/yetone/magpie/internal/update.defaultNotesFeed=https://freemagpie.com/api/notes -X github.com/yetone/magpie/internal/update.releasePagePrefix=https://github.com/cq0206/freemagpie/releases/tag/freemagpie-v"
arch=$(go env GOARCH)
mkdir -p dist
smoke_home=$(mktemp -d)
case "$target" in
  macos)
    make app VERSION="$version" LDFLAGS="$ldflags"
    # Preserve magpie's executable, bundle identifier and configuration paths.
    /usr/libexec/PlistBuddy -c 'Set :CFBundleDisplayName FreeMagpie' magpie.app/Contents/Info.plist
    cp LICENSE magpie.app/Contents/Resources/LICENSE
    codesign --force --deep --sign - magpie.app
    codesign --verify --deep --strict magpie.app
    env HOME="$smoke_home" ./magpie.app/Contents/MacOS/magpie --version
    ditto -c -k --sequesterRsrc --keepParent magpie.app "dist/magpie-darwin-$arch.zip"
    make release-cli VERSION="$version" LDFLAGS="$ldflags" TARGETS="darwin/$arch"
    ;;
  linux)
    make release-linux VERSION="$version" LDFLAGS="$ldflags"
    env HOME="$smoke_home" "./dist/magpie-linux-$arch" --version
    targets="linux/$arch"
    if [ "$arch" = amd64 ]; then targets="$targets android/arm64 windows/amd64 windows/arm64"; fi
    make release-cli VERSION="$version" LDFLAGS="$ldflags" TARGETS="$targets"
    ;;
  windows)
    make release-windows VERSION="$version" LDFLAGS="$ldflags"
    ;;
  *) echo "unknown target: $target" >&2; exit 1 ;;
esac
cp LICENSE THIRD_PARTY_NOTICES dist/
