# Veristack · 验迹

> **让每一次软件交付，都有可以独立核验的依据。**

Veristack 是一个独立于 Git、CI、测试框架和 Agent 平台的**软件交付证据层**。它消费这些工具产生的证据，把客户需求、开发过程、代码变更、测试结果和客户验收，整理成一份双方都能独立核验的交付记录。

| 需求有依据 | 变更有确认 | 交付有凭据 | 验收可追溯 |
| --- | --- | --- | --- |
| 原始需求与验收条件 | 每次调整都留下客户决定 | 代码版本与开发记录 | 客户确认与异议记录 |
| 冻结的范围基线不可被改写 | 批准后生成新的基线版本 | 清单生成 SHA-256 摘要 | 争议可以逐条回溯 |

**核心对象是一次软件交付，而不是一个 Agent Session。** 一个交付版本对应一个不可随意覆盖的证据快照。

---

## 目录

- [它解决什么问题](#它解决什么问题)
- [核心能力](#核心能力)
- [技术栈](#技术栈)
- [快速开始](#快速开始)
- [项目结构](#项目结构)
- [Evidence Pack：可离线核验的交付证据](#evidence-pack可离线核验的交付证据)
- [第三方平台接入](#第三方平台接入)
- [AI 的边界](#ai-的边界)
- [常用命令](#常用命令)
- [文档](#文档)
- [路线图](#路线图)
- [许可证](#许可证)

---

## 它解决什么问题

软件外包与定制开发中，最常见的纠纷不是「代码写没写」，而是**「当初约定的是什么」**：

- 客户说「我以为包含导出功能」，开发者说「需求单里没写」。
- 需求改过好几轮，没有人说得清最终范围。
- 交付完了，客户说「这个没达到要求」，开发者说「已经做过了」。
- 最后双方只能凭记忆和截图争论。

Veristack 把这件事变成一条**可核验的记录链**：

```
需求基线 v1 ──冻结──▶ 变更申请 ──客户确认──▶ 需求基线 v2
                                                      │
                                    交付版本（清单冻结 + SHA-256）
                                                      │
                              证据（提交、CI 报告、截图、测试报告）
                                                      │
                              客户逐条验收 ──▶ 确认 / 异议与回复闭环
```

每一步都留下时间、操作人和内容摘要，双方都可以独立复现当时发生了什么。

## 核心能力

### 1. 需求范围基线与变更确认

需求条目带验收条件，基线一经冻结即不可编辑。任何调整都必须通过变更申请，由客户确认后生成新的基线版本，旧版本原样保留，随时可对比差异。

### 2. 交付版本与不可覆盖的清单

创建交付时自动把当前基线的需求与验收条件冻结为清单快照，并生成 SHA-256 摘要。交付之后补充证据只会以关联方式追加，不会改写历史清单。

### 3. 证据时间线

Git 提交、CI 报告、截图、录屏、测试报告、文档与人工说明统一归档，按时间倒序形成证据链。附件落盘即计算内容摘要。

### 4. 客户验收门户

客户通过签名分享链接进入（无需注册），逐条对验收条件给出「符合 / 不符合」并填写说明，可对任一条提出异议。开发者回复后客户复核，形成闭环。

### 5. Evidence Pack 离线核验

导出交付快照与证据附件为 Evidence Pack，配置签名私钥时附带 Ed25519 签名。客户用 `veristack verify` 即可在本地核验内容是否被篡改，**无需信任服务端**。

### 6. 自动化接入

- **Webhook**：GitHub、GitLab、Gitee 事件按项目维度接入，先落库再异步处理
- **提交自动归档**：按提交信息中的 `REQ-xxx` 编号关联到需求，生成证据
- **CI 报告导入**：解析标准 JUnit XML，汇总通过、失败、跳过与耗时
- **AI 建议**：需求提取、变更影响分析、交付摘要，全部为待确认草稿

## 技术栈

| 层 | 选型 | 说明 |
| --- | --- | --- |
| 后端 | Go 1.27 + Chi | 模块化单体，路由与中间件链清晰 |
| 数据访问 | PostgreSQL + pgx + sqlc | 从 SQL 生成类型安全代码，Goose 管理迁移 |
| 存储 | S3 兼容接口（一期为本地文件系统） | 证据附件与交付快照 |
| 前端 | React 19 + TypeScript + Vite | TanStack Query / Router / Table |
| 样式 | Tailwind CSS 4 + shadcn/ui | 语义化 CSS 变量驱动的明暗双主题 |
| 完整性 | SHA-256 + Ed25519 | 内容摘要与交付清单签名 |
| 契约 | OpenAPI 3.1 | 39 个接口的完整描述 |

## 快速开始

### 前置条件

- Go 1.27+
- Node.js 20+
- PostgreSQL 17（或使用 Docker Compose 一键启动）

### 使用 Docker Compose（推荐）

```bash
cp .env.example .env
# 修改 .env 中的 VERISTACK_AUTH_SECRET 与数据库密码

docker compose -f deploy/compose/docker-compose.yml up -d --build
```

访问 <http://localhost:5173>，数据库迁移会在 API 启动时自动执行。

### 本地开发

```bash
# 后端
cd api
export VERISTACK_DATABASE_DSN='postgres://veristack:veristack@localhost:5432/veristack?sslmode=disable'
export VERISTACK_AUTH_SECRET='change-me-in-production'
go run ./cmd/server

# 前端（另开终端）
cd web
npm install
npm run dev
```

前端开发服务器会把 `/api` 请求代理到 `http://127.0.0.1:8080`。

### 首次使用

1. 打开控制台，选择「注册组织」创建工作室与 owner 账号
2. 新建项目，录入需求并冻结为基线 v1
3. 需要调整范围时发起变更申请，由客户确认后生成 v2
4. 从基线创建交付版本，补充证据后生成客户验收链接

## 项目结构

```text
veristack/
├── api/                          # Go 后端
│   ├── cmd/
│   │   ├── server/               # API 服务入口
│   │   └── veristack/            # Evidence Pack 核验 CLI
│   ├── contracts/
│   │   ├── openapi/              # OpenAPI 3.1 契约
│   │   └── evidence-pack/        # Evidence Pack 格式契约
│   ├── internal/
│   │   ├── acceptance/           # 客户验收与异议闭环
│   │   ├── change/               # 变更申请
│   │   ├── delivery/             # 交付版本与清单
│   │   ├── evidence/             # 证据归档
│   │   ├── identity/             # 注册与登录
│   │   ├── integration/          # Webhook、CI 报告、AI 建议
│   │   ├── milestone/            # 里程碑
│   │   ├── platform/             # 数据库、配置、认证、存储、模型等基础设施
│   │   ├── project/              # 项目与成员
│   │   └── scope/                # 需求范围基线
│   ├── migrations/               # Goose 迁移脚本
│   └── sql/queries/              # sqlc 查询源
├── web/                          # React 前端
│   └── src/
│       ├── app/                  # 入口、路由与全局状态
│       ├── components/ui/        # 通用组件
│       ├── features/             # 按业务模块组织
│       ├── layouts/              # 布局
│       ├── services/             # API 请求封装
│       └── shared/               # 类型与工具
├── deploy/
│   ├── compose/                  # Docker Compose 编排
│   ├── docker/                   # 镜像构建
│   └── docs/                     # 部署文档
├── docs/                         # 产品与实现规划
└── agents.md                     # 项目开发约定
```

## Evidence Pack：可离线核验的交付证据

Evidence Pack 是一份可以脱离 Veristack 服务端独立核验的交付快照：

```
delivery-pack/
├── pack.json      # 元数据：清单摘要、证据清单、签名
├── manifest.json  # 交付清单快照
└── files/         # 证据附件
```

服务端导出：

```bash
curl -X POST http://localhost:8080/api/v1/deliveries/{id}/pack \
  -H "Authorization: Bearer <token>"
```

客户本地核验：

```bash
veristack verify ./delivery-pack
```

```
交付：官网改版 v1（基线 v2）
清单摘要：a08b27008e6aaae1...

[通过] 交付清单摘要：清单内容与元数据一致
[通过] 交付清单签名：签名有效

结论：全部核验通过，交付内容与签名均有效
```

内容被改动时，摘要与签名两项都会失败，CLI 返回非零退出码，可直接接入流水线做交付卡口。

格式契约见 [`api/contracts/evidence-pack/v1.md`](api/contracts/evidence-pack/v1.md)。

## 第三方平台接入

为每个项目配置回调地址：

```
POST https://your-veristack/api/v1/webhooks/{github|gitlab|gitee}/projects/{projectId}
```

| 平台 | 校验方式 | 环境变量 |
| --- | --- | --- |
| GitHub | `X-Hub-Signature-256` HMAC-SHA256 | `VERISTACK_WEBHOOK_GITHUBSECRET` |
| GitLab | `X-Gitlab-Token` | `VERISTACK_WEBHOOK_GITLABSECRET` |
| Gitee | `X-Gitee-Token` | `VERISTACK_WEBHOOK_GITEESECRET` |

事件先完整落库，再由后台 Worker 异步处理，不阻塞第三方回调。

## AI 的边界

Veristack 里 AI 只承担它擅长的工作：提取候选需求、分析变更影响、生成交付摘要。

**AI 不会做这些事：**

- 修改正式需求基线
- 判定客户已经验收
- 把未运行的测试标记为通过
- 把推测的开发行为记录为真实事实

所有 AI 产出都是待确认草稿，必须经人工采纳或忽略后才改变状态。采纳建议只标记建议本身生效，不会借此改写基线或交付结论。未配置模型密钥时，系统使用本地确定性规则，功能依然可用且结果可复现。

## 常用命令

```bash
make help          # 查看全部命令
make api-run       # 本地运行 API
make api-vet       # go vet 静态检查
make cli-build     # 编译 Evidence Pack 核验 CLI
make web-dev       # 启动前端开发服务器
make web-lint      # 前端 ESLint
make sqlc-gen      # 根据 SQL 重新生成类型安全代码
make compose-up    # 启动完整容器环境
```

## 文档

| 文档 | 内容 |
| --- | --- |
| [`docs/implementation-plan.md`](docs/implementation-plan.md) | 产品定位、需求清单、数据模型、API 设计与实现路线 |
| [`docs/chatGPT-chat-history.md`](docs/chatGPT-chat-history.md) | 产品定位的推演过程与技术选型依据 |
| [`deploy/docs/deployment.md`](deploy/docs/deployment.md) | 私有化部署、环境变量、备份恢复与常见问题 |
| [`api/contracts/openapi/openapi.yaml`](api/contracts/openapi/openapi.yaml) | OpenAPI 3.1 接口契约 |
| [`api/contracts/evidence-pack/v1.md`](api/contracts/evidence-pack/v1.md) | Evidence Pack 格式契约 |
| [`agents.md`](agents.md) | 项目开发约定（通用原则 + 产品领域规则） |
| [`api/agents.md`](api/agents.md) | 后端分层、持久化与安全约定 |
| [`web/agents.md`](web/agents.md) | 前端组件、状态与视觉设计约定 |

## 路线图

- [x] **阶段 0 · 基础层**：Chi + PostgreSQL + sqlc + Goose、认证、对象存储、双主题前端、容器编排
- [x] **阶段 1 · 交付业务核心**：组织与用户、项目与成员、需求基线、变更申请与确认、里程碑
- [x] **阶段 2 · 证据与验收**：交付清单、证据归档、客户分享链接、逐条验收、异议闭环、审计事件
- [x] **阶段 3 · 自动化**：Webhook 接入、Outbox 任务队列、提交自动关联、CI 报告导入、AI 建议层
- [x] **阶段 4 · 开放生态**：Evidence Pack 与独立核验 CLI、OpenAPI 契约、Gitee 接入、私有化部署文档

## 许可证

Apache-2.0
