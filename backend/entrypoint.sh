#!/bin/sh
set -e

echo "Running database migrations..."
migrate -path /db/migrations -database "$DATABASE_URL" up

echo "Starting $1..."
exec "$@"
