#!/bin/sh

apk add --no-cache ca-certificates wget
addgroup -S leoxy
adduser -S -G leoxy leoxy
mkdir -p /app/leoxy /app/log /app/docs/html
chown -R leoxy:leoxy /app
