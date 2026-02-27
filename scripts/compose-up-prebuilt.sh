#!/bin/sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

make build-prebuilt
docker compose -f docker-compose.yml -f docker-compose.prebuilt.yml up --build
