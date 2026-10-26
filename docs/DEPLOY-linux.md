# 中华共和实录·野 —— Linux 部署手册

版本：**1.0.0**　构建日期：2026-10-26
适用：x86_64 / arm64 的 Linux 服务器（systemd），无需安装 Go、Node 或数据库。

---

## 一、交付物

发布包 `gonghe-linux-1.0.0.tar.gz` 解压后：

| 路径 | 说明 |
| --- | --- |
| `bin/gonghe-amd64` | Linux x86_64 可执行文件（静态编译，无外部依赖） |
| `bin/gonghe-arm64` | Linux arm64 可执行文件 |
| `web/` | 已构建的前端静态文件（Vue 3 + Element Plus） |
| `deploy/gonghe.service` | systemd 单元模板 |
| `deploy/gonghe.env.example` | 环境变量模板 |
| `deploy/nginx-gonghe.conf` | nginx 反向代理示例 |
| `install.sh` | 一键安装 / 升级 |
| `uninstall.sh` | 卸载 |
| `backup.sh` | SQLite 在线备份 |
| `DEPLOY.md` | 本文件 |
| `VERSION` | 版本号 |
| `SHA256SUMS` | 二进制校验和 |

程序是单个静态链接的 Go 二进制，数据库为纯 Go 实现的 SQLite（WAL 模式），**不依赖 glibc、libsqlite3 或任何运行时**。

---

## 二、环境要求

| 项目 | 最低 | 建议 |
| --- | --- | --- |
| 系统 | 任意带 systemd 的 Linux 发行版 | Debian 12 / Ubuntu 22.04+ / Rocky 9 |
| 架构 | amd64 或 arm64 | amd64 |
| 内存 | 256 MB | 512 MB |
| 磁盘 | 300 MB | 2 GB（含备份） |
| 端口 | 8090（应用）、80/443（nginx） | 仅暴露 80/443 |

检查架构：

```bash
uname -m
# x86_64 → amd64；aarch64 → arm64
```

---

## 三、快速安装

### 1. 上传并解压

在本地执行（把文件换成实际路径）：

```bash
scp gonghe-linux-1.0.0.tar.gz root@your-server:/tmp/
```

登录服务器：

```bash
cd /tmp
tar -xzf gonghe-linux-1.0.0.tar.gz
cd gonghe-linux-1.0.0
sha256sum -c SHA256SUMS        # 可选，校验二进制
```

### 2. 一键安装

```bash
sudo bash install.sh
```

如果是**公开部署**，建议首次安装就关闭演示账号：

```bash
sudo SEED_DEMO=false bash install.sh
```

脚本会自动：

1. 识别 amd64 / arm64，选用对应二进制；
2. 创建系统用户 `gonghe`（不可登录）；
3. 安装程序到 `/opt/gonghe`，静态文件到 `/opt/gonghe/web`；
4. 创建数据目录 `/var/lib/gonghe`；
5. 生成 `/etc/gonghe/gonghe.env`，并写入随机 `JWT_SECRET`（权限 600）；
6. 安装并启动 `gonghe.service`；
7. 轮询 `/healthz` 做健康检查。

看到 `安装完成，服务已启动` 即可访问 `http://服务器IP:8090`。

> 如果服务商有安全组，请先放行 8090（验证后再改为只放行 80/443，见第七节）。

### 3. 验证

```bash
curl -s http://127.0.0.1:8090/healthz
# ok

curl -s http://127.0.0.1:8090/api/v1/stats
# {"code":0,...,"totalEvents":52,"totalBranches":8,...}

systemctl status gonghe --no-pager
journalctl -u gonghe -n 50 --no-pager
```

浏览器打开 `http://服务器IP:8090`，应看到「中华共和实录」报头与谱系图。

---

## 四、安装后的目录

| 路径 | 内容 | 权限 |
| --- | --- | --- |
| `/opt/gonghe/gonghe` | 可执行文件 | root:root 755 |
| `/opt/gonghe/web/` | 前端静态文件 | root:root 755 |
| `/opt/gonghe/backup.sh` | 备份脚本 | root:root 755 |
| `/etc/gonghe/gonghe.env` | 环境变量（含 JWT 密钥） | root:root 600 |
| `/var/lib/gonghe/gonghe.db` | SQLite 数据库 | gonghe:gonghe 640 |
| `/var/lib/gonghe/gonghe.db-wal` | WAL 日志 | gonghe:gonghe |
| `/etc/systemd/system/gonghe.service` | systemd 单元 | root:root 644 |

程序以 `gonghe` 用户运行；systemd 开启 `ProtectSystem=strict`，除 `/var/lib/gonghe` 外整个文件系统只读。

---

## 五、配置

配置文件：`/etc/gonghe/gonghe.env`。修改后执行 `sudo systemctl restart gonghe`。

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `APP_ENV` | `prod` | `prod` 关闭 Gin 调试输出；`dev` 会打印路由 |
| `ADDR` | `:8090` | 监听地址。仅本机反代时建议 `127.0.0.1:8090` |
| `DATA_DIR` | `/var/lib/gonghe` | 数据目录 |
| `DB_PATH` | `/var/lib/gonghe/gonghe.db` | SQLite 文件路径 |
| `STATIC_DIR` | `/opt/gonghe/web` | 前端产物目录；留空则只提供 API |
| `JWT_SECRET` | 安装时随机生成 | JWT 签名密钥，**必须保密**；更换后所有登录失效 |
| `JWT_TTL` | `168h` | 登录有效期（7 天） |
| `ALLOW_REGISTER` | `true` | 是否开放注册 |
| `SEED_DEMO` | `true` | 空库首次启动是否写入演示数据，见下 |
| `DEFAULT_PAGE_SIZE` | `20` | 默认分页大小 |
| `MAX_PAGE_SIZE` | `100` | 最大分页大小 |
| `DB_MAX_OPEN_CONNS` | `4` | SQLite 连接池上限 |
| `DB_MAX_IDLE_CONNS` | `4` | 空闲连接数 |
| `DB_CACHE_KB` | `16384` | SQLite page cache（16 MB） |
| `DB_BUSY_TIMEOUT` | `5s` | 写锁等待时间 |
| `DB_MMAP_MB` | `32` | mmap 大小 |

### 关于 `SEED_DEMO`

- `true`（默认）：空库首次启动写入 6 位演示史官、8 条史枝、52 条实录、53 个赞、13 条评论。演示账号密码统一为 `123456`，**仅适合内网试用或演示**。
- `false`：仍写入正史主干与 8 条史枝、52 条实录（保证有新用户可分叉的「主流分支」），但不写演示点赞与评论，并把演示账号的口令设为随机值（无法登录）。适合公开部署。

该开关只在**空库**生效；已有数据时不会重复写入。要重新灌种子，见第十节「重置数据」。

> 公开部署建议：`sudo SEED_DEMO=false bash install.sh`，或首次启动前把 `SEED_DEMO=false` 写进配置；随后自行注册账号并从正史主干分叉。

### 改为仅本机监听

```bash
sudo sed -i 's/^ADDR=.*/ADDR=127.0.0.1:8090/' /etc/gonghe/gonghe.env
sudo systemctl restart gonghe
```

---

## 六、systemd 运维

```bash
sudo systemctl status gonghe        # 状态
sudo systemctl restart gonghe       # 重启
sudo systemctl stop gonghe          # 停止
sudo systemctl start gonghe         # 启动
sudo systemctl enable gonghe        # 开机自启（安装时已做）
sudo systemctl disable gonghe       # 取消自启
journalctl -u gonghe -f             # 实时日志
journalctl -u gonghe --since today  # 今日日志
```

健康检查：

```bash
curl -fsS http://127.0.0.1:8090/healthz && echo OK
curl -fsS http://127.0.0.1:8090/api/v1/stats | head -c 300
```

---

## 七、nginx 与 HTTPS

### 1. 安装 nginx

```bash
# Debian / Ubuntu
sudo apt update && sudo apt install -y nginx

# Rocky / Alma / CentOS
sudo dnf install -y nginx
```

### 2. 配置反代

```bash
sudo cp deploy/nginx-gonghe.conf /etc/nginx/conf.d/gonghe.conf
sudo sed -i 's/server_name example.com;/server_name your-domain.com;/' /etc/nginx/conf.d/gonghe.conf
```

Debian/Ubuntu 若使用 `sites-enabled`：

```bash
sudo mv /etc/nginx/conf.d/gonghe.conf /etc/nginx/sites-available/gonghe
sudo ln -s /etc/nginx/sites-available/gonghe /etc/nginx/sites-enabled/gonghe
sudo rm -f /etc/nginx/sites-enabled/default
```

测试并重载：

```bash
sudo nginx -t
sudo systemctl reload nginx
```

此时应能用域名访问，且 8090 可以只监听 `127.0.0.1`。

### 3. 防火墙

```bash
# ufw
sudo ufw allow 'Nginx Full'
sudo ufw delete allow 8090/tcp 2>/dev/null || true

# firewalld
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
```

**不要**把 8090 长期暴露在公网；它没有 TLS，且限流按 IP 设计。

### 4. HTTPS

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.com
sudo certbot renew --dry-run
```

证书自动续期由 `certbot.timer` 负责。

### 5. SELinux（仅 RHEL 系）

若 nginx 返回 502 且 `audit.log` 有 `httpd_t` 拒绝：

```bash
sudo setsebool -P httpd_can_network_connect 1
```

---

## 八、升级与回滚

### 升级

```bash
# 1. 先备份
sudo /opt/gonghe/backup.sh

# 2. 上传并解压新版本
tar -xzf gonghe-linux-1.1.0.tar.gz
cd gonghe-linux-1.1.0

# 3. 重跑安装脚本（保留配置与数据）
sudo bash install.sh
```

`install.sh` 会替换二进制和 `web/`，**不会覆盖** `/etc/gonghe/gonghe.env`，随后重启服务。数据库结构在启动时自动迁移（只增列，不删列）。

### 回滚

```bash
sudo systemctl stop gonghe
sudo cp /path/to/old/gonghe /opt/gonghe/gonghe
sudo chmod 755 /opt/gonghe/gonghe
sudo systemctl start gonghe
```

若新版本已写入新列，旧版本通常仍可运行（GORM 忽略未知列），但仍建议先恢复升级前的备份。

---

## 九、备份、恢复与迁移

### 备份

```bash
sudo /opt/gonghe/backup.sh
# 或指定目录
sudo /opt/gonghe/backup.sh /data/backups/gonghe
```

脚本优先用 `sqlite3 .backup`，否则用 `python3` 的 `sqlite3.backup()`，都是**在线一致性备份**，不会拷贝到写了一半的 WAL。默认保留 14 天（`KEEP_DAYS` 可改）。

定时任务：

```bash
sudo crontab -e
# 每天 03:15
15 3 * * * /opt/gonghe/backup.sh >/var/log/gonghe-backup.log 2>&1
```

### 恢复

```bash
sudo systemctl stop gonghe

sudo gunzip -c /var/backups/gonghe/gonghe-20261026-031500.db.gz > /tmp/restore.db
sudo install -o gonghe -g gonghe -m 640 /tmp/restore.db /var/lib/gonghe/gonghe.db
sudo rm -f /var/lib/gonghe/gonghe.db-wal /var/lib/gonghe/gonghe.db-shm

sudo systemctl start gonghe
curl -fsS http://127.0.0.1:8090/healthz
```

**必须删除旧的 `-wal` / `-shm`**，否则 SQLite 可能把旧日志重放到新库上。

### 迁移到新服务器

1. 旧机 `backup.sh`；
2. 新机按第三节装好程序（先别启动）；
3. 用上面的恢复步骤导入数据库；
4. 启动并验证。

---

## 十、重置数据

```bash
sudo systemctl stop gonghe
sudo rm -rf /var/lib/gonghe/*
sudo systemctl start gonghe
```

启动时会按当前 `SEED_DEMO` 重新灌种子。**此操作不可恢复**，先备份。

---

## 十一、手动运行（无 systemd / 容器内）

```bash
cd /opt/gonghe
sudo -u gonghe env \
  APP_ENV=prod \
  ADDR=127.0.0.1:8090 \
  DATA_DIR=/var/lib/gonghe \
  STATIC_DIR=/opt/gonghe/web \
  JWT_SECRET='请替换为随机长串' \
  SEED_DEMO=false \
  /opt/gonghe/gonghe
```

前台运行，`Ctrl+C` 退出。容器内同理，把 `/var/lib/gonghe` 挂成 volume 即可。

> 容器里没有 systemd 时，`install.sh` 会提示不可用；可以只用它准备目录，然后按上面的命令启动，或自行写 Dockerfile。

---

## 十二、源码构建（可选）

其他架构或需要自行编译时：

```bash
# 后端
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -buildvcs=false -ldflags="-s -w" -o gonghe ./cmd/server

# 前端
cd ../web
pnpm install
pnpm build          # 产物在 web/dist
```

把 `gonghe` 与 `dist/` 一起拷到服务器，参照第三节手动放置即可。Go 版本要求见 `backend/go.mod`。

---

## 十三、验收清单

部署完成后逐项确认：

- [ ] `systemctl is-active gonghe` 返回 `active`
- [ ] `curl http://127.0.0.1:8090/healthz` 返回 `ok`
- [ ] `/api/v1/stats` 返回 `code: 0`，史枝与实录数量符合预期
- [ ] 浏览器能看到首页报头、谱系图、史枝总目
- [ ] 注册 / 登录成功，顶栏显示自己的名号
- [ ] 能打开任一史枝并「自此分叉」
- [ ] 能续写条目、点赞、评论
- [ ] 刷新深链接（如 `/branches/1`）不 404
- [ ] 重启服务后登录态仍有效（`JWT_SECRET` 未变）
- [ ] nginx 域名访问正常，证书有效
- [ ] 备份脚本能生成 `.db.gz`，且可解压
- [ ] 日志无持续报错：`journalctl -u gonghe -p err --since today`

---

## 十四、常见问题

**502 Bad Gateway**
`systemctl status gonghe` 看服务是否在跑；`curl 127.0.0.1:8090/healthz`；确认 nginx 的 `proxy_pass` 端口与 `ADDR` 一致；RHEL 系检查 SELinux。

**首页 404 / 白屏，但 API 正常**
`STATIC_DIR` 不对或 `web/index.html` 缺失。确认 `/opt/gonghe/web/index.html` 存在，配置指向 `/opt/gonghe/web`。

**重启后所有人都要重新登录**
`JWT_SECRET` 为空或被改动。安装脚本会生成并保存在 `/etc/gonghe/gonghe.env`，不要删除。

**`database is locked`**
同一个数据库被两个进程打开（例如 systemd 与手动命令同时跑），或备份工具直接 `cp` 了 WAL 库。确保只运行一个实例，备份用 `backup.sh`。

**端口被占用**
`ss -lntp | grep 8090`，改 `ADDR` 或停掉占用进程。

**首次启动慢 1–3 秒**
空库要写种子并计算 bcrypt，属正常。日志出现 `[seed] 已写入 …` 即完成。

**「今日新记」日期不对**
systemd 单元已设 `TZ=Asia/Shanghai`；手动运行时自行加 `TZ=Asia/Shanghai`。

**注册关闭了怎么加人**
把 `ALLOW_REGISTER=true` 改回并重启；或直接在数据库中处理（不推荐）。

**想换域名 / 加 CDN**
静态资源带 hash，可长缓存；HTML 与 API 不要缓存。CDN 回源到 nginx 即可。

---

## 十五、安全清单

- [ ] `JWT_SECRET` 为随机 64 位十六进制，文件权限 600
- [ ] `SEED_DEMO=false`（公开站点），或已确认演示账号不可用
- [ ] 服务以 `gonghe` 非 root 用户运行
- [ ] 8090 只监听 `127.0.0.1`，公网仅 80/443
- [ ] 已启用 HTTPS，HTTP 自动跳转
- [ ] 按需设置 `ALLOW_REGISTER=false`
- [ ] 防火墙仅放行 22/80/443
- [ ] 备份文件不在 Web 可访问目录
- [ ] 定期 `apt upgrade` / `dnf update` 系统组件

---

## 十六、演示账号（仅 `SEED_DEMO=true`）

| 名号 | 登录名 | 口令 |
| --- | --- | --- |
| 说书人 | `shuoshuren` | `123456` |
| 白山客 | `baishanke` | `123456` |
| 听雨者 | `tingyuzhe` | `123456` |
| 金陵旧客 | `jinling` | `123456` |
| 岭南生 | `lingnan` | `123456` |
| 史馆 | `shiguan` | `123456` |

> 这些账号只用于体验。公开部署请使用 `SEED_DEMO=false`。

---

## 十七、卸载

```bash
# 保留数据与配置
sudo bash uninstall.sh

# 连数据、配置、系统用户一起删除
sudo bash uninstall.sh --purge
```

---

*中华共和实录·野 —— 正史一树，野史千枝。*
