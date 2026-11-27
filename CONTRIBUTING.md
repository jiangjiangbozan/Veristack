# 贡献指南

感谢你愿意参与 Veristack。这个项目的核心不是“再加一个功能”，而是让软件交付过程变得更可核验，所以提交前请优先保证**行为正确、证据完整、边界清楚**。

## 开始之前

- 先搜索已有 [Issues](https://github.com/jiangjiangbozan/Veristack/issues)，避免重复。
- 较大的功能、接口或数据模型调整，请先开 Issue 说明动机、方案和影响范围，再开始编码。
- 阅读 [`AGENTS.md`](AGENTS.md)、[`api/agents.md`](api/agents.md) 与 [`web/agents.md`](web/agents.md)，遵循已有的分层、命名和主题约定。

## 开发流程

1. Fork 仓库并创建分支：`feat/<topic>`、`fix/<topic>` 或 `docs/<topic>`。
2. 安装依赖并按 [README 快速开始](README.md#快速开始) 启动本地环境。
3. 修改代码；涉及业务规则时补充单元测试。
4. 本地跑完检查项。
5. 提交 Pull Request，说明改动、验证方式、风险和未覆盖项。

## 提交信息

使用 [Conventional Commits](https://www.conventionalcommits.org/)：

```text
feat: 支持导出交付审计时间线
fix: 修复未登录访问受保护路由时的 404
docs: 补充 Evidence Pack 核验说明
test: 覆盖基线冻结后的写入拒绝
```

一个 PR 只做一件逻辑上完整的事。格式化、重命名和功能改动尽量拆开，方便 review。

## 提交前检查

```bash
make api-vet
make api-test
make web-lint
make web-build
```

如果改动涉及以下内容，还需要额外检查：

| 改动类型 | 额外要求 |
| --- | --- |
| HTTP 接口或字段 | 更新 `api/contracts/openapi/openapi.yaml`，契约测试必须通过 |
| 数据库结构 | 新增 Goose 迁移，不修改已发布迁移；迁移需在本机 PostgreSQL 验证 |
| 证据包格式 | 更新 `api/contracts/evidence-pack/` 契约与核验 CLI 测试 |
| 前端主题与组件 | 使用语义化 CSS 变量，同时检查明暗两套主题 |
| 安全相关逻辑 | 补充失败路径测试，并在 PR 中说明威胁模型 |

## Pull Request 验收标准

- 能说明“为什么改”，而不只是“改了什么”。
- 新增业务逻辑有对应测试；没有测试的，需要在 PR 中解释原因。
- 不引入 `any`、静默降级、未处理的外部输入或绕过权限的路径。
- 不提交密钥、令牌、真实客户数据、构建产物或本地环境文件。
- 文档、契约和实现保持同步。

## 安全问题

请不要在公开 Issue 中披露漏洞细节，按 [`SECURITY.md`](SECURITY.md) 的方式私下报告。
