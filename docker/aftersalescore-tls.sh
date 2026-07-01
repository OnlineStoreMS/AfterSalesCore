#!/bin/sh
set -eu
# 若已挂载 TLS 证书，则启用 HTTPS 配置（供客户端摄像头 getUserMedia 使用）
if [ -f /etc/nginx/certs/server.crt ] && [ -f /etc/nginx/certs/server.key ]; then
  TLS_PORT="${AFTERSALES_TLS_PORT:-5177}"
  sed "s/__TLS_PORT__/${TLS_PORT}/g" /etc/nginx/aftersalescore-web.ssl.conf \
    > /etc/nginx/conf.d/default.conf
fi
