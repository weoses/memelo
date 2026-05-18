#!/usr/bin/env bash
set -euo pipefail

if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "Error: uncommitted changes — commit or stash them first" >&2
    exit 1
fi

usage() {
    echo "Usage: $0              # create next dev tag  (v00.000.00-N-dev)" >&2
    echo "       $0 <version>    # create release tag   (v<version>)" >&2
    echo "  <version> must be formatted as XX.YYY.ZZ, e.g. 11.222.33" >&2
    exit 1
}

if [ $# -eq 0 ]; then
    LAST_DEV=$(git tag -l 'v*-dev' | sort -V | tail -n 1)

    if [ -z "$LAST_DEV" ]; then
        NEXT=1
    else
        LAST_NUM=$(echo "$LAST_DEV" | sed -E 's/.*-([0-9]+)-dev$/\1/')
        NEXT=$((LAST_NUM + 1))
    fi

    NEW_TAG="v00.000.00-${NEXT}-dev"
    echo "Creating dev tag: $NEW_TAG"
    git tag "$NEW_TAG"
    #git push origin "$NEW_TAG"

elif [ $# -eq 1 ]; then
    VERSION="$1"

    if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
        echo "Error: version must be formatted as XX.YYY.ZZ (e.g. 11.222.33)" >&2
        exit 1
    fi

    NEW_TAG="v${VERSION}"

    if git tag -l | grep -qx "$NEW_TAG"; then
        echo "Error: tag $NEW_TAG already exists" >&2
        exit 1
    fi

    echo "Creating release tag: $NEW_TAG"
    git tag "$NEW_TAG"
    #git push origin "$NEW_TAG"

else
    usage
fi