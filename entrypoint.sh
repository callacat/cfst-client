#!/bin/bash

CONFIG_DIR="/app/config"
DEFAULT_CONFIG_DIR="/app/default_config"

# 检查 config.yml 是否存在，如果不存在则从默认位置复制
if [ ! -f "${CONFIG_DIR}/config.yml" ]; then
  echo "config.yml not found in ${CONFIG_DIR}, copying default config..."
  cp "${DEFAULT_CONFIG_DIR}/config.yml" "${CONFIG_DIR}/config.yml"
fi

# 检查 ip.txt 是否存在，如果不存在则从默认位置复制
if [ ! -f "${CONFIG_DIR}/ip.txt" ]; then
  echo "ip.txt not found in ${CONFIG_DIR}, copying default ip.txt..."
  cp "${DEFAULT_CONFIG_DIR}/ip.txt" "${CONFIG_DIR}/ip.txt"
fi

# 检查 ipv6.txt 是否存在，如果不存在则从默认位置复制
if [ ! -f "${CONFIG_DIR}/ipv6.txt" ]; then
  echo "ipv6.txt not found in ${CONFIG_DIR}, copying default ipv6.txt..."
  cp "${DEFAULT_CONFIG_DIR}/ipv6.txt" "${CONFIG_DIR}/ipv6.txt"
fi

# 执行主程序
exec /app/test-client