#!/usr/bin/env bash
set -euo pipefail

# Grove Street Release Script
# Usage: ./scripts/release.sh [version]
# Example: ./scripts/release.sh 0.2.0

REPO="notuselessdev/grove-street"
TAP_REPO="notuselessdev/homebrew-tap"
FORMULA="Formula/grove-street.rb"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}[CJ]${NC} $1"; }
ok()    { echo -e "${GREEN}[CJ]${NC} $1"; }
error() { echo -e "${RED}[CJ]${NC} $1" >&2; exit 1; }

# Determine version
if [ -n "${1:-}" ]; then
    VERSION="$1"
else
    VERSION=$(cat VERSION 2>/dev/null || echo "")
    [ -z "$VERSION" ] && error "No version specified. Usage: $0 <version>"
fi

TAG="v${VERSION}"
info "Releasing Grove Street ${TAG}"

# Preflight checks
command -v gh &>/dev/null || error "gh CLI not found. Install: https://cli.github.com"
command -v git &>/dev/null || error "git not found"
gh auth status &>/dev/null || error "Not logged in to GitHub. Run: gh auth login"

# Ensure working tree is clean
if [ -n "$(git status --porcelain)" ]; then
    error "Working tree is dirty. Commit or stash changes first."
fi

# Update VERSION file
echo "$VERSION" > VERSION
if [ -n "$(git status --porcelain VERSION)" ]; then
    git add VERSION
    git commit -m "Bump version to ${VERSION}"
fi

# Push latest code
info "Pushing to origin..."
git push origin main

# Delete existing release/tag if present
if gh release view "$TAG" --repo "$REPO" &>/dev/null; then
    info "Deleting existing release ${TAG}..."
    gh release delete "$TAG" --repo "$REPO" --yes
fi
if git rev-parse "$TAG" &>/dev/null 2>&1; then
    git tag -d "$TAG"
    git push origin --delete "$TAG" 2>/dev/null || true
fi

# Tag and push
info "Tagging ${TAG}..."
git tag "$TAG"
git push origin "$TAG"

# Wait for release workflow
info "Waiting for release workflow..."
sleep 5

RUN_ID=$(gh run list --repo "$REPO" --limit 1 --json databaseId --jq '.[0].databaseId')
if [ -z "$RUN_ID" ]; then
    error "Could not find workflow run"
fi

info "Watching workflow run ${RUN_ID}..."
gh run watch "$RUN_ID" --repo "$REPO" --exit-status || error "Release workflow failed"
ok "Release workflow completed"

# Wait for assets to be available
sleep 3

# Compute sha256 checksums
info "Computing checksums..."
declare -A SHAS
for arch in darwin_arm64 darwin_amd64 linux_arm64 linux_amd64; do
    url="https://github.com/${REPO}/releases/download/${TAG}/grove-street_${arch}.tar.gz"
    sha=$(curl -sL "$url" | shasum -a 256 | cut -d' ' -f1)
    SHAS[$arch]="$sha"
    info "  ${arch}: ${sha}"
done

# Verify checksums are unique (not error pages)
unique_shas=$(printf '%s\n' "${SHAS[@]}" | sort -u | wc -l | tr -d ' ')
if [ "$unique_shas" -lt 2 ]; then
    error "All checksums are identical — repo may be private or assets not ready"
fi

# Update local formula
info "Updating Homebrew formula..."
sed -i '' "s|version \".*\"|version \"${VERSION}\"|" "$FORMULA"

# Update sha256 values in order: darwin_arm64, darwin_amd64, linux_arm64, linux_amd64
# The formula has 4 sha256 lines in this exact order
awk -v sha1="${SHAS[darwin_arm64]}" \
    -v sha2="${SHAS[darwin_amd64]}" \
    -v sha3="${SHAS[linux_arm64]}" \
    -v sha4="${SHAS[linux_amd64]}" '
BEGIN { n=0 }
/sha256/ {
    n++
    if (n==1) { sub(/"[^"]*"/, "\"" sha1 "\"") }
    if (n==2) { sub(/"[^"]*"/, "\"" sha2 "\"") }
    if (n==3) { sub(/"[^"]*"/, "\"" sha3 "\"") }
    if (n==4) { sub(/"[^"]*"/, "\"" sha4 "\"") }
}
{ print }
' "$FORMULA" > "${FORMULA}.tmp" && mv "${FORMULA}.tmp" "$FORMULA"

# Commit and push formula update
git add "$FORMULA"
git commit -m "Update Homebrew formula for ${TAG}"
git push origin main

# Update homebrew-tap repo
info "Updating homebrew-tap..."
TAP_DIR=$(mktemp -d)
git clone "git@github.com:${TAP_REPO}.git" "$TAP_DIR" --quiet
mkdir -p "$TAP_DIR/Formula"
cp "$FORMULA" "$TAP_DIR/Formula/grove-street.rb"
cd "$TAP_DIR"
git add Formula/grove-street.rb
git commit -m "Update grove-street to ${TAG}"
git push --quiet
cd -
rm -rf "$TAP_DIR"

ok "Released Grove Street ${TAG}"
echo ""
echo "  Install: brew install notuselessdev/tap/grove-street"
echo "  Release: https://github.com/${REPO}/releases/tag/${TAG}"
echo ""
