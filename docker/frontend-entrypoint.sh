#!/bin/sh
set -eu

: "${FRONTEND_AGENT_URL:=http://localhost:17443}"
: "${FRONTEND_BACKEND_URL:=http://localhost:8000}"

envsubst '${FRONTEND_AGENT_URL} ${FRONTEND_BACKEND_URL}' \
  < /usr/share/nginx/html/config.template.js \
  > /usr/share/nginx/html/config.js
