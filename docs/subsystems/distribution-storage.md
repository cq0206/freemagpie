# FreeMagpie storage and signing

FreeMagpie's Makefile and release packager set `internal/appdir.Name` to
`freemagpie`. `appdir` owns config, cache, OS cache, logs and Windows WebView
profile paths; Linux GTK/WebKit uses the same name. Upstream unit tests keep
the upstream default; `TestFreeMagpieOwnDirectories` exercises the distribution
namespace. Raw `go run` without the distribution linker flags is an upstream
development build; use `make build` or `make cli` for FreeMagpie.

Installed directories:

| Purpose | Default |
| --- | --- |
| Configuration and persistent data | `~/.config/freemagpie` (`XDG_CONFIG_HOME` honored) |
| Downloadable cache | `~/.cache/freemagpie` (`XDG_CACHE_HOME` honored) |
| macOS logs / OS cache | `~/Library/Logs/freemagpie`, `~/Library/Caches/freemagpie` |
| Linux logs / WebKit data | `~/.local/state/freemagpie`, `~/.local/share/freemagpie` (XDG honored) |
| Windows logs / OS cache | `%LOCALAPPDATA%/freemagpie` |
| Windows WebView profile | config directory's `webview2` subdirectory |
| Windows installation record / folder | `HKCU/Software/freemagpie`, per-user `Programs/freemagpie` |

macOS bundle/login records use `com.freemagpie.app`; daemon records use that
namespace on macOS and `freemagpie.service` on Linux. Agent-owned directories
and protocol keys belong to their existing contracts. Portable builds retain
their explicitly selected `data` folder, with cache/logs/webview2 below it.

Existing Magpie data is not read automatically. Before first FreeMagpie launch,
stop Magpie and run `magpie data import-magpie` using the FreeMagpie binary
(inside `FreeMagpie.app/Contents/MacOS/` on macOS). This explicitly copies the
sibling `magpie` config directory, leaves the original intact and refuses an
existing destination, symbolic links and special files. Files are staged before
publication; an error is reported instead of silently starting from a failed
copy. Caches are downloaded again. Review imported custom absolute paths: user
values are preserved rather than rewritten. Future writes use FreeMagpie paths.

The release packager optionally calls `build/freemagpie/notarize.sh` when
`FREEMAGPIE_SIGNING_IDENTITY` and `FREEMAGPIE_NOTARY_PROFILE` are provided.
The certificate/private key and notarytool credentials live in the user's
Keychain. It requires Developer ID Application signing, hardened runtime,
secure timestamp, Apple's Accepted result, stapling, ticket validation and a
Gatekeeper assessment before producing distribution ZIPs. Any failure aborts
packaging. Without these credentials, packaging remains explicitly ad-hoc and
must not be described as notarized. `brand_macos.py` is an ad-hoc-only repair
tool; never run it over a notarized release.

Verification: `go test ./internal/appdir ./internal/settings ./internal/daemon
./internal/autostart ./internal/update`, CLI/GUI checks, cross-platform vet and
build checks, and a real distribution build under a temporary HOME. See the
FreeMagpie verification workflow for the complete Linux nogui suite.
