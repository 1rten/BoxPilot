#!/bin/sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

IMAGE_TAG="${1:-boxpilot:latest}"

make build
docker build -f docker/Dockerfile.prebuilt -t "$IMAGE_TAG" .
