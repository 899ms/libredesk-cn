（把下面整段粘给 Antigravity 里的 Gemini 会话）

你在 `F:\claude bf\openclash ssh\libredesk` 这个仓库上做中国市场二开。先完整读两份文档：`docs-cn/二开方案.md` 和 `docs-cn/任务分工.md`，然后按「Gemini」表里的顺序做 G2 → G3 → G4 → G5 → G6 → G7（G1 已经做完）。

规则：
1. 每个任务开一个分支 `cn/<模块>`，从 `cn` 分出，做完合回 `cn`。不要碰 `main`。
2. 改到上游文件的每一处都加 `// [cn-fork]`（Go）或 `<!-- [cn-fork] -->`（Vue/HTML）注释。
3. 只允许改 `i18n/en-US.json` 和 `i18n/zh-CN.json` 两个语言包，新键两边同时加。
4. 每个任务结束前跑 `go test ./...`、`go vet ./...`、`cd frontend && pnpm test:run && pnpm lint`，把命令和结果贴出来。
5. 汇报格式按 `任务分工.md` 末尾的模板，写到 `docs-cn/进度.md` 追加一段。
6. 遇到方案里没写清的决定，先按方案第 4.4 节「少碰上游文件」的原则选，再在进度里注明你的选择。

先做 G2，做完停下来汇报。
