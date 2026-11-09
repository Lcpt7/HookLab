#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
if [ -f .env ]; then set -a; . ./.env; set +a; fi
: "${MYSQL_PASSWORD:?Run scripts/setup.sh first}"
export TEST_DATABASE_DSN="${TEST_DATABASE_DSN:-hooklab:${MYSQL_PASSWORD}@tcp(127.0.0.1:3306)/hooklab?parseTime=true&charset=utf8mb4&loc=UTC}"
cd backend
exec "${GO_BIN:-go}" test -race ./... -count=1 "$@"
