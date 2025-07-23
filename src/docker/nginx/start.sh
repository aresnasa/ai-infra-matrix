#!/bin/sh
# 启动nginx with custom config
nginx -c /etc/nginx/nginx.conf -g 'daemon off;'
