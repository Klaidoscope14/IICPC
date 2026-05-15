#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
COMPOSE_DIR="$ROOT_DIR/infrastructure/docker"

if [ ! -f "$COMPOSE_DIR/.env" ] && [ -f "$COMPOSE_DIR/.env.example" ]; then
  cp "$COMPOSE_DIR/.env.example" "$COMPOSE_DIR/.env"
  echo "Created $COMPOSE_DIR/.env from .env.example"
fi

cd "$COMPOSE_DIR"

docker compose --env-file .env up -d --build

"$ROOT_DIR/scripts/healthcheck.sh"
