#!/usr/bin/env bash
# 中华共和实录·野 —— 卸载脚本
# 用法：sudo bash uninstall.sh          仅移除程序，保留数据与配置
#       sudo bash uninstall.sh --purge  连同数据库、配置和系统用户一起删除
set -euo pipefail

PREFIX="${PREFIX:-/opt/gonghe}"
DATA_DIR="${DATA_DIR:-/var/lib/gonghe}"
CONF_DIR="${CONF_DIR:-/etc/gonghe}"
SERVICE_USER="${SERVICE_USER:-gonghe}"
SERVICE_NAME="gonghe"
PURGE=0

if [[ "${1:-}" == "--purge" ]]; then
  PURGE=1
fi

[[ "${EUID:-$(id -u)}" -eq 0 ]] || { echo "请用 root 运行：sudo bash uninstall.sh" >&2; exit 1; }

if command -v systemctl >/dev/null 2>&1; then
  systemctl stop "$SERVICE_NAME" 2>/dev/null || true
  systemctl disable "$SERVICE_NAME" 2>/dev/null || true
fi

rm -f "/etc/systemd/system/$SERVICE_NAME.service"
command -v systemctl >/dev/null 2>&1 && systemctl daemon-reload || true
rm -rf "$PREFIX"

echo "已停止并移除程序：$PREFIX"

if [[ "$PURGE" == "1" ]]; then
  rm -rf "$DATA_DIR" "$CONF_DIR"
  if id -u "$SERVICE_USER" >/dev/null 2>&1; then
    userdel "$SERVICE_USER" 2>/dev/null || true
  fi
  echo "已删除数据、配置与用户 $SERVICE_USER。"
else
  echo "数据与配置已保留："
  echo "  数据：$DATA_DIR"
  echo "  配置：$CONF_DIR"
  echo "如需彻底删除，执行：sudo bash uninstall.sh --purge"
fi
