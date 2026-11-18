#!/usr/bin/env bash
# freemagpie: independent distribution; never dispatch upstream signing jobs.
set -euo pipefail
version=${1:?usage: package.sh VERSION macos|linux|windows}
target=${2:?missing target}
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'expected X.Y.Z' >&2; exit 1; }
ldflags="-X github.com/yetone/magpie/internal/appdir.Name=freemagpie -s -w -X main.version=$version -X github.com/yetone/magpie/internal/update.defaultFeed=https://freemagpie.com/api/latest -X github.com/yetone/magpie/internal/update.defaultNotesFeed=https://freemagpie.com/api/notes -X github.com/yetone/magpie/internal/update.releasePagePrefix=https://github.com/cq0206/freemagpie/releases/tag/freemagpie-v"
arch=$(go env GOARCH)
mkdir -p dist
smoke_home=$(mktemp -d)
case "$target" in
  macos)
    make app VERSION="$version" LDFLAGS="$ldflags"
    # Preserve magpie's executable, bundle identifier and configuration paths.
    /usr/libexec/PlistBuddy -c 'Set :CFBundleDisplayName FreeMagpie' magpie.app/Contents/Info.plist
    /usr/libexec/PlistBuddy -c 'Set :CFBundleName FreeMagpie' magpie.app/Contents/Info.plist
    /usr/libexec/PlistBuddy -c 'Set :CFBundleIdentifier com.freemagpie.app' magpie.app/Contents/Info.plist
    cp LICENSE magpie.app/Contents/Resources/LICENSE
    if [[ -n "${FREEMAGPIE_SIGNING_IDENTITY:-}${FREEMAGPIE_NOTARY_PROFILE:-}" ]]; then
      bash build/freemagpie/notarize.sh magpie.app
    else
      codesign --force --deep --sign - magpie.app
    fi
    codesign --verify --deep --strict magpie.app
    env HOME="$smoke_home" ./magpie.app/Contents/MacOS/magpie --version
    ditto -c -k --sequesterRsrc --keepParent magpie.app "dist/magpie-darwin-$arch.zip"
    branded=$(mktemp -d)
    ditto magpie.app "$branded/FreeMagpie.app"
    ditto -c -k --sequesterRsrc --keepParent "$branded/FreeMagpie.app" "dist/freemagpie-darwin-$arch.zip"
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
# User-facing downloads use the distribution name. Legacy assets remain for
# installed versions whose updater expects magpie-* and magpie.app.
for asset in dist/magpie-*; do
  [[ "$asset" == *.zip ]] && continue
  cp "$asset" "dist/free${asset#dist/}"
done
