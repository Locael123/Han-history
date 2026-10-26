#!/usr/bin/env bash
# 中华共和实录·野 —— SQLite 在线备份
# 用法：sudo bash backup.sh [输出目录]
# 环境变量：DB_PATH、KEEP_DAYS（默认保留 14 天）
set -euo pipefail

DB="${DB_PATH:-}"
if [[ -z "$DB" && -r /etc/gonghe/gonghe.env ]]; then
  DB="$(sed -n 's/^DB_PATH=//p' /etc/gonghe/gonghe.env | tail -n 1 | tr -d '"')"
fi
DB="${DB:-/var/lib/gonghe/gonghe.db}"
OUT="${1:-/var/backups/gonghe}"
KEEP_DAYS="${KEEP_DAYS:-14}"
STAMP="$(date +%Y%m%d-%H%M%S)"
TMP="$OUT/gonghe-$STAMP.db"

[[ -f "$DB" ]] || { echo "找不到数据库：$DB" >&2; exit 1; }
mkdir -p "$OUT"

if command -v sqlite3 >/dev/null 2>&1; then
  sqlite3 "$DB" ".backup '$TMP'"
elif command -v python3 >/dev/null 2>&1; then
  python3 - "$DB" "$TMP" <<'PY'
import sqlite3
import sys

src = sqlite3.connect(sys.argv[1])
dst = sqlite3.connect(sys.argv[2])
with dst:
    src.backup(dst)
dst.close()
src.close()
PY
else
  echo "需要 sqlite3 或 python3 才能安全备份（WAL 模式下不能直接 cp）" >&2
  exit 1
fi

gzip -f "$TMP"
find "$OUT" -type f -name 'gonghe-*.db.gz' -mtime "+$KEEP_DAYS" -delete

echo "备份完成：$TMP.gz"
echo "保留最近 $KEEP_DAYS 天，目录：$OUT"
