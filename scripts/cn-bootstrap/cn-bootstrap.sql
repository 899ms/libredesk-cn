-- [cn-fork] 中国市场部署的一次性初始化（K7）
-- 可以单独执行：psql -v site_name='在线客服' -v root_url='https://chat.example.com' -f cn-bootstrap.sql
-- 幂等：重复执行不报错、不重复建。

\set ON_ERROR_STOP on

-- 变量缺省值，允许不传参直接跑
\if :{?site_name}
\else
\set site_name '在线客服'
\endif
\if :{?root_url}
\else
\set root_url 'http://localhost:9000'
\endif

BEGIN;

-- ── 1. 联系人自定义属性 ────────────────────────────────────────────────
-- key 必须是英文：网站后端签发 JWT 时，contact_custom_attributes 用的就是这些 key。
-- name 是中文：客服在联系人侧栏看到的就是它。
-- 唯一约束是 (key, applies_to)，所以用 ON CONFLICT 做幂等。
-- 已存在的条目只更新中文显示名和描述，不动 data_type——避免覆盖管理员事后的调整。

INSERT INTO custom_attribute_definitions ("name", description, applies_to, key, data_type)
VALUES
  ('会员等级',   '网站侧的会员/VIP 等级，登录用户由 JWT 带入。',       'contact', 'vip_level',      'text'),
  ('注册时间',   '该用户在网站的注册日期。',                            'contact', 'register_at',    'date'),
  ('最近订单号', '最近一笔订单的编号，方便客服直接查单。',              'contact', 'last_order_no',  'text'),
  ('来源渠道',   '用户从哪个渠道进来，如官网、小程序、广告投放。',      'contact', 'source_channel', 'text'),
  ('手机归属地', '手机号归属省市，用于判断时区与话术。',                'contact', 'phone_region',   'text')
ON CONFLICT (key, applies_to) DO UPDATE
  SET "name"     = EXCLUDED."name",
      description = EXCLUDED.description,
      updated_at  = NOW();

-- ── 2. 站点名称与地址 ──────────────────────────────────────────────────
-- settings 表是 key/value(jsonb)，用 to_jsonb 保证字符串被正确编码（中文、引号都安全）。

UPDATE settings SET value = to_jsonb(:'site_name'::text) WHERE "key" = 'app.site_name';
UPDATE settings SET value = to_jsonb(:'root_url'::text)  WHERE "key" = 'app.root_url';
UPDATE settings SET value = to_jsonb((:'root_url' || '/favicon.ico')::text) WHERE "key" = 'app.favicon_url';

-- 语言与时区在 schema.sql 里已经是 zh-CN / Asia/Shanghai（见 [cn-fork] 注释）。
-- 这里再写一次，是为了让从英文实例升级上来的库也能对齐。
UPDATE settings SET value = '"zh-CN"'::jsonb        WHERE "key" = 'app.lang';
UPDATE settings SET value = '"Asia/Shanghai"'::jsonb WHERE "key" = 'app.timezone';

COMMIT;

-- ── 3. 结果自检 ────────────────────────────────────────────────────────
\echo ''
\echo '已预置的联系人自定义属性：'
SELECT key AS "键（JWT 用）", "name" AS "显示名", data_type AS "类型"
FROM custom_attribute_definitions
WHERE applies_to = 'contact'
  AND key IN ('vip_level','register_at','last_order_no','source_channel','phone_region')
ORDER BY id;

\echo ''
\echo '当前设置：'
SELECT "key", value FROM settings
WHERE "key" IN ('app.site_name','app.root_url','app.favicon_url','app.lang','app.timezone')
ORDER BY "key";
