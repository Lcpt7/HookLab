#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
if [ -e .env ]; then echo '.env exists; kept unchanged.'; exit 0; fi
umask 077
{ printf 'MYSQL_PASSWORD=%s\n' "$(openssl rand -hex 24)"; printf 'MYSQL_ROOT_PASSWORD=%s\n' "$(openssl rand -hex 24)"; printf 'APP_ORIGIN=http://localhost:8080\nCOOKIE_SECURE=false\nPORT=8080\nMAX_BODY_BYTES=1048576\n'; } > .env
echo 'Created .env with random database passwords.'
