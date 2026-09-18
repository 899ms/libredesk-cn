# cn-bootstrap —— 中国市场部署的一次性初始化

给刚装好的 Libredesk 实例预置中文环境。**幂等**，重复跑不会出错也不会重复建。

## 它做两件事

**一、建好五个联系人自定义属性**，显示名中文、key 英文：

| key（网站签 JWT 时用） | 显示名（客服看到的） | 类型 |
| --- | --- | --- |
| `vip_level` | 会员等级 | text |
| `register_at` | 注册时间 | date |
| `last_order_no` | 最近订单号 | text |
| `source_channel` | 来源渠道 | text |
| `phone_region` | 手机归属地 | text |

key 必须是英文，因为网站后端签发 JWT 时 `contact_custom_attributes` 传的就是这些 key：

```js
contact_custom_attributes: {
  vip_level: '黄金',
  register_at: '2024-03-15',
  last_order_no: 'SO20260917001',
}
```

客服在联系人侧栏看到的则是「会员等级：黄金」。

**二、写入站点名称、站点地址、语言与时区**。语言和时区在 `schema.sql` 里已经默认是 `zh-CN` / `Asia/Shanghai`，这里再写一次是为了让从英文实例升级上来的库也能对齐。

## 怎么跑

装了 `psql` 的机器上：

```bash
SITE_NAME='在线客服' ROOT_URL='https://chat.example.com' \
PGHOST=localhost PGPORT=5432 PGUSER=libredesk PGPASSWORD=xxx PGDATABASE=libredesk \
./cn-bootstrap.sh
```

docker compose 部署，直接喂给数据库容器：

```bash
docker compose exec -T db psql -U libredesk -d libredesk \
  -v site_name='在线客服' -v root_url='https://chat.example.com' \
  < cn-bootstrap.sql
```

两个变量都可以不传，默认是「在线客服」和 `http://localhost:9000`。

## ⚠️ 跑完要重启

应用启动时把 `settings` 表载入内存（koanf），所以站点名称和地址改完之后，**要重启实例，或者在后台「设置 → 常规」点一次保存**，新值才会生效。自定义属性不受这个限制，立即可见。

## 校验

脚本末尾会自动打印建好的属性和当前设置。也可以手动查：

```sql
SELECT key, name, data_type FROM custom_attribute_definitions WHERE applies_to = 'contact';
SELECT key, value FROM settings WHERE key LIKE 'app.%';
```
