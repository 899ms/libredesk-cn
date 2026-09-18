#!/usr/bin/env bash
# [cn-fork] 中国市场部署的一次性初始化脚本（K7）
#
# 作用：给一个刚装好的 Libredesk 实例预置中文环境——
#   1. 建好客服端联系人侧栏常用的自定义属性，显示名是中文、key 是英文
#      （JWT 的 contact_custom_attributes 传的是 key，所以 key 必须是英文）
#   2. 设置站点名称与站点地址
#
# 幂等：重复执行不会报错、不会重复建。
#
# 用法：
#   SITE_NAME='在线客服' ROOT_URL='https://chat.example.com' \
#   PGHOST=localhost PGPORT=5432 PGUSER=libredesk PGPASSWORD=xxx PGDATABASE=libredesk \
#   ./cn-bootstrap.sh
#
#   跑在 docker compose 部署上时，直接进数据库容器执行：
#   docker compose exec -T db psql -U libredesk -d libredesk < cn-bootstrap.sql
#   （本脚本会先生成 cn-bootstrap.sql，也可只用那个 SQL）
#
# 环境变量：
#   SITE_NAME    站点名称，显示在后台与邮件里。默认「在线客服」
#   ROOT_URL     站点对外地址，必须带协议、不带结尾斜杠。默认 http://localhost:9000
#   PGHOST/PGPORT/PGUSER/PGPASSWORD/PGDATABASE  标准 libpq 变量，psql 直接读
#
# ⚠️ 改完 site_name / root_url 后，运行中的实例需要重启，或在后台「设置 → 常规」里
#    点一次保存，新值才会进内存（应用启动时把 settings 表载入 koanf）。

set -euo pipefail

SITE_NAME="${SITE_NAME:-在线客服}"
ROOT_URL="${ROOT_URL:-http://localhost:9000}"

if ! command -v psql >/dev/null 2>&1; then
  echo "错误：找不到 psql。请安装 postgresql-client，或直接把同目录的 cn-bootstrap.sql 喂给数据库。" >&2
  exit 1
fi

# ROOT_URL 基本校验：必须 http/https 开头，不能有结尾斜杠
case "$ROOT_URL" in
  http://*|https://*) ;;
  *) echo "错误：ROOT_URL 必须以 http:// 或 https:// 开头，当前是 '$ROOT_URL'" >&2; exit 1 ;;
esac
case "$ROOT_URL" in
  */) echo "错误：ROOT_URL 不能以斜杠结尾，当前是 '$ROOT_URL'" >&2; exit 1 ;;
esac

echo "→ 目标库：${PGDATABASE:-libredesk}@${PGHOST:-localhost}:${PGPORT:-5432}"
echo "→ 站点名称：$SITE_NAME"
echo "→ 站点地址：$ROOT_URL"

psql -v ON_ERROR_STOP=1 \
     -v site_name="$SITE_NAME" \
     -v root_url="$ROOT_URL" \
     -f "$(dirname "$0")/cn-bootstrap.sql"

echo
echo "✓ 初始化完成。"
echo "  若实例正在运行，重启它、或在后台「设置 → 常规」点一次保存，新的站点名称与地址才会生效。"
