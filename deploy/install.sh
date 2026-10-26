#!/usr/bin/env bash
# 中华共和实录·野 —— Linux 一键安装 / 升级脚本
# 用法：sudo bash install.sh
# 可选环境变量：PREFIX、DATA_DIR、CONF_DIR、SERVICE_USER
set -euo pipefail

PREFIX="${PREFIX:-/opt/gonghe}"
DATA_DIR="${DATA_DIR:-/var/lib/gonghe}"
CONF_DIR="${CONF_DIR:-/etc/gonghe}"
SERVICE_USER="${SERVICE_USER:-gonghe}"
SERVICE_NAME="gonghe"
ENV_FILE="$CONF_DIR/gonghe.env"

die() { echo "错误：$*" >&2; exit 1; }
info() { echo "==> $*"; }

[[ "${EUID:-$(id -u)}" -eq 0 ]] || die "请用 root 运行：sudo bash install.sh"
command -v systemctl >/dev/null 2>&1 || die "未检测到 systemd。请参考 DEPLOY.md 的「手动运行」一节。"
command -v install >/dev/null 2>&1 || die "缺少 install 命令（coreutils）"

SRC="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$SRC/web/index.html" ]] || die "包内缺少 web/index.html，请重新解压完整发布包"

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) BIN="gonghe-amd64" ;;
  aarch64|arm64) BIN="gonghe-arm64" ;;
  *) die "不支持的架构：$ARCH（发布包仅含 amd64 / arm64）" ;;
esac
[[ -f "$SRC/bin/$BIN" ]] || die "找不到二进制：$SRC/bin/$BIN"
info "检测到 $ARCH，使用 bin/$BIN"

if ! id -u "$SERVICE_USER" >/dev/null 2>&1; then
  info "创建系统用户 $SERVICE_USER"
  useradd --system --home-dir "$DATA_DIR" --create-home --shell /usr/sbin/nologin "$SERVICE_USER" 2>/dev/null \
    || useradd --system --home-dir "$DATA_DIR" --create-home --shell /bin/false "$SERVICE_USER"
fi

info "安装程序到 $PREFIX"
install -d -m 0755 "$PREFIX"
install -m 0755 "$SRC/bin/$BIN" "$PREFIX/gonghe"
rm -rf "$PREFIX/web"
install -d -m 0755 "$PREFIX/web"
cp -a "$SRC/web/." "$PREFIX/web/"
if [[ -f "$SRC/DEPLOY.md" ]]; then install -m 0644 "$SRC/DEPLOY.md" "$PREFIX/DEPLOY.md"; fi
if [[ -f "$SRC/VERSION" ]]; then install -m 0644 "$SRC/VERSION" "$PREFIX/VERSION"; fi
if [[ -f "$SRC/backup.sh" ]]; then install -m 0755 "$SRC/backup.sh" "$PREFIX/backup.sh"; fi

info "准备数据目录 $DATA_DIR"
install -d -m 0750 -o "$SERVICE_USER" -g "$SERVICE_USER" "$DATA_DIR"

info "写入配置 $ENV_FILE"
install -d -m 0755 "$CONF_DIR"
if [[ -f "$ENV_FILE" ]]; then
  info "配置已存在，保留原文件（不会覆盖 JWT_SECRET 与数据库路径）"
else
  SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"
  DEMO="${SEED_DEMO:-true}"
  sed -e "s|__JWT_SECRET__|$SECRET|" \
      -e "s|__SEED_DEMO__|$DEMO|g" \
      -e "s|__PREFIX__|$PREFIX|g" \
      -e "s|__DATA_DIR__|$DATA_DIR|g" \
      "$SRC/deploy/gonghe.env.example" > "$ENV_FILE"
  chmod 0600 "$ENV_FILE"
  info "已生成随机 JWT_SECRET（$ENV_FILE，权限 600）"
  if [[ "$DEMO" == "false" ]]; then
    info "SEED_DEMO=false：不写入演示点赞/评论，演示账号口令随机化"
  fi
fi

info "安装 systemd 单元 /etc/systemd/system/$SERVICE_NAME.service"
sed -e "s|__PREFIX__|$PREFIX|g" \
    -e "s|__DATA_DIR__|$DATA_DIR|g" \
    -e "s|__CONF_DIR__|$CONF_DIR|g" \
    -e "s|__USER__|$SERVICE_USER|g" \
    "$SRC/deploy/gonghe.service" > "/etc/systemd/system/$SERVICE_NAME.service"
chmod 0644 "/etc/systemd/system/$SERVICE_NAME.service"

systemctl daemon-reload
systemctl enable "$SERVICE_NAME" >/dev/null 2>&1
systemctl restart "$SERVICE_NAME"

ADDR="$(sed -n 's/^ADDR=//p' "$ENV_FILE" | tail -n 1 | tr -d '"')"
ADDR="${ADDR:-:8090}"
PORT="${ADDR##*:}"
HOST="${ADDR%:*}"
if [[ -z "$HOST" || "$HOST" == "$ADDR" ]]; then
  HOST="127.0.0.1"
fi

if command -v curl >/dev/null 2>&1; then
  ok=0
  for _ in $(seq 1 20); do
    if curl -fsS "http://$HOST:$PORT/healthz" >/dev/null 2>&1; then
      ok=1
      break
    fi
    sleep 1
  done
  echo
  if [[ "$ok" == "1" ]]; then
    echo "安装完成，服务已启动：http://$HOST:$PORT"
  else
    echo "服务已安装，但健康检查未通过。请执行："
    echo "  journalctl -u $SERVICE_NAME -n 80 --no-pager"
    exit 1
  fi
else
  echo
  echo "安装完成（未安装 curl，跳过健康检查）：http://$HOST:$PORT"
fi

EFFECTIVE_DEMO="$(sed -n 's/^SEED_DEMO=//p' "$ENV_FILE" | tail -n 1)"
if [[ "$EFFECTIVE_DEMO" == "false" ]]; then
  DEMO_NOTE="演示账号口令已随机化，请直接注册新账号，再从正史主干分叉。"
else
  DEMO_NOTE="首次访问会写入 6 位演示史官与 8 枝示例（密码均为 123456）。"
fi

cat <<EOF

配置文件：$ENV_FILE
数据目录：$DATA_DIR
静态文件：$PREFIX/web
管理命令：systemctl status|restart|stop $SERVICE_NAME
查看日志：journalctl -u $SERVICE_NAME -f

$DEMO_NOTE
对外提供服务前，请配置 nginx 与 HTTPS，见 DEPLOY.md。
EOF
