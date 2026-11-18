#!/usr/bin/env bash
# Developer ID certificate/private key and notarytool profile stay in Keychain.
set -euo pipefail
app=${1:?usage: notarize.sh APP_BUNDLE}
identity=${FREEMAGPIE_SIGNING_IDENTITY:?set a Developer ID Application identity}
profile=${FREEMAGPIE_NOTARY_PROFILE:?set an existing notarytool Keychain profile}
[[ "$identity" == 'Developer ID Application: '* ]] || { echo 'Developer ID Application certificate required' >&2; exit 1; }
security find-identity -v -p codesigning | grep -F -- "$identity" >/dev/null
xcrun --find notarytool >/dev/null
codesign --force --options runtime --timestamp --sign "$identity" "$app"
codesign --verify --deep --strict "$app"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
ditto -c -k --sequesterRsrc --keepParent "$app" "$work/app.zip"
xcrun notarytool submit "$work/app.zip" --keychain-profile "$profile" \
  --wait --timeout 20m --output-format json > "$work/result.json"
python3 - "$work/result.json" <<'PY'
import json, sys
result = json.load(open(sys.argv[1]))
if result.get('status') != 'Accepted':
    raise SystemExit(f"Apple notarization not accepted: {result}")
print('Apple notarization accepted:', result['id'])
PY
xcrun stapler staple "$app"
xcrun stapler validate "$app"
spctl --assess --type execute --verbose=2 "$app"
