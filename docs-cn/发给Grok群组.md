（把下面整段发到 Grok Bot 的「libredesk开发群组」）

@开发bot @测试bot 我们要给 https://github.com/abhinavxd/libredesk 做中国市场二开。方案和分工在仓库 `docs-cn/二开方案.md`、`docs-cn/任务分工.md`（我把两个文件内容贴在下面）。你们负责「Grok bot」表里的 K1 到 K6。

分工：
- 开发bot：K1（邮件模板 + schema.sql 种子中文化）、K2（公共页面与帮助中心模板中文化）、K3（默认设置改中文/上海时区 + 微信微博社交键）。在云主机上 `git clone https://github.com/abhinavxd/libredesk && git checkout f4916f41`，每个任务一个分支 `cn/<模块>`，交付 `git format-patch` 出来的 patch 文件贴到群里。
- 测试bot：K6（海外依赖排查，先做，只出清单）、K5（微信内置浏览器实测挂件）、K4（等 Gemini 的 G2+G3 合进 `cn` 分支后，在云主机 `docker compose up` 跑全部验收项）。

约束：
1. 不改 `i18n/en-US.json` 和 `zh-CN.json` 以外的语言包；新键两边同时加。
2. Go 模板里能用 `{{ T "key" }}` 的一律走语言包，不要硬编码中文。
3. 每个任务完成后按 `任务分工.md` 末尾的汇报模板回一条。

---- 以下是 docs-cn/二开方案.md 全文 ----
（粘贴文件内容）

---- 以下是 docs-cn/任务分工.md 全文 ----
（粘贴文件内容）
