#!/bin/sh
set -eu
if [ "$#" -lt 1 ]; then
  echo "Usage: sh scripts/publish-image.sh YOUR_DOCKERHUB_USERNAME [VERSION]" >&2
  exit 1
fi
namespace=$1
version=${2:-1.0.0}
case "$namespace" in *[!a-z0-9_-]*|'') echo 'Invalid Docker Hub username' >&2; exit 1;; esac
case "$version" in *[!a-zA-Z0-9_.-]*|'') echo 'Invalid image version' >&2; exit 1;; esac
cd "$(dirname "$0")/.."
docker buildx build --platform linux/amd64,linux/arm64 \
  -t "$namespace/geofence-api:$version" --push backend
printf '\nPublished image: https://hub.docker.com/r/%s/geofence-api\nImage tag: %s\n' "$namespace" "$version"
