<p align="center">
  <img src="./docs/readme-hero.svg" alt="Veristack · 验迹：让每一次软件交付，都有可以独立核验的依据" width="100%" />
</p>

<h1 align="center">Veristack · 验迹</h1>

<p align="center">
  <strong>独立于 Git、CI 和 Agent 平台的软件交付证据层。</strong><br />
  把需求基线、变更确认、交付清单、证据和客户验收，串成一条双方都能独立核验的记录链。
</p>

<p align="center">
  <a href="https://github.com/jiangjiangbozan/Veristack/actions/workflows/ci.yml">
    <img alt="CI" src="https://github.com/jiangjiangbozan/Veristack/actions/workflows/ci.yml/badge.svg?branch=main" />
  </a>
  <a href="LICENSE">
    <img alt="License: Apache-2.0" src="https://img.shields.io/badge/license-Apache--2.0-blue.svg" />
  </a>
  <a href="https://github.com/jiangjiangbozan/Veristack/blob/main/api/go.mod">
    <img alt="Go 1.27.2" src="https://img.shields.io/badge/Go-1.27.2-00ADD8?logo=go&amp;logoColor=white" />
  </a>
  <a href="https://github.com/jiangjiangbozan/Veristack/blob/main/web/package.json">
    <img alt="React 19" src="https://img.shields.io/badge/React-19-61DAFB?logo=react&amp;logoColor=white" />
  </a>
  <a href="https://github.com/jiangjiangbozan/Veristack">
    <img alt="Status: Alpha" src="https://img.shields.io/badge/status-alpha-E8A33D" />
  </a>
</p>

<p align="center">
  <a href="#快速开始">快速开始</a> ·
  <a href="#核心能力">核心能力</a> ·
  <a href="#架构">架构</a> ·
  <a href="#evidence-pack可离线核验的交付证据">Evidence Pack</a> ·
  <a href="#贡献">贡献</a>
</p>

> **Status: Alpha.** 核心交付闭环、Evidence Pack v2 和客户验收已完成实现并通过本地集成验证；接口、数据模型与部署方式仍在收敛。生产使用前请先阅读 [部署指南](deploy/docs/deployment.md) 并完成备份与升级演练。

## 为什么是验迹

Git 记录代码，CI 记录测试，聊天工具记录沟通，但它们都不负责回答一个问题：

> **这次交付到底答应了什么，客户又是基于什么确认的？**

Veristack 不替代这些工具。它消费它们产生的证据，把“约定 → 变更 → 交付 → 验收”固定成一条只追加的记录链。

| 交付现场的问题 | 只靠 Git / CI / 聊天记录 | Veristack |
| --- | --- | --- |
| 当初承诺了什么？ | 散落在需求文档和聊天里 | 冻结的需求基线与验收条件 |
| 中途改过什么？ | 提交记录里看不出来 | 变更申请、客户确认与新基线版本 |
| 这次交付包含什么？ | 一个压缩包和一句“已完成” | 不可覆盖的交付清单与 SHA-256 摘要 |
| 客户怎么确认的？ | 聊天截图，难以归因 | 客户验收门户，逐条确认并保留异议闭环 |
| 事后怎么证明？ | 只能继续信任服务端 | Evidence Pack v2 + Ed25519，可离线核验 |

**核心对象是一次软件交付，而不是一个 Agent Session。** 一个交付版本对应一个不可随意覆盖的证据快照。

## 工作原理

```mermaid
flowchart LR
  A["需求与验收条件"] --> B["冻结基线 v1"]
  B --> C["变更申请"]
  C -->|"客户确认"| D["新基线 v2"]
  D --> E["交付清单<br/>SHA-256 快照"]
  E --> F["证据时间线<br/>提交 / CI / 截图"]
  F --> G["客户验收<br/>逐条确认 / 异议闭环"]
  G --> H["Evidence Pack v2<br/>Ed25519 签名"]
  H --> I["veristack verify<br/>离线核验"]
```

每一步都记录时间、操作人和内容摘要。冻结后的基线与交付清单只读；后续补充证据只能关联追加，不能改写历史。

## 核心能力

| 能力 | 说明 |
| --- | --- |
| **需求基线与变更确认** | 需求条目带验收条件，基线冻结后不可编辑；调整必须走变更申请，客户确认后生成新版本，旧版本原样保留。 |
| **不可覆盖的交付清单** | 创建交付时冻结当前基线为清单快照并生成 SHA-256 摘要；补充证据只追加，不改写历史清单。 |
| **证据时间线** | Git 提交、CI 报告、截图、录屏、测试报告、文档与人工说明统一归档，按时间倒序形成证据链。 |
| **客户验收门户** | 客户通过签名分享链接进入，无需注册；逐条给出「符合 / 不符合」，可提出异议，开发者回复后客户复核。 |
| **Evidence Pack 离线核验** | 导出交付快照与附件，附带 Ed25519 签名；客户用 `veristack verify` 在本地核验，无需信任服务端。 |
| **自动化与 AI 建议** | GitHub / GitLab / Gitee Webhook、提交按 `REQ-xxx` 关联需求、JUnit XML 报告导入；AI 产出全部为待确认草稿。 |

> Veristack 不是又一个 Jira，也不替代 Git、CI 或测试平台。它只负责把这些系统里的证据固定下来，并让客户可以独立核验。

## 快速开始

### 一键启动（推荐）

需要 **Docker 24+** 与 **Docker Compose v2**。

```bash
git clone https://github.com/jiangjiangbozan/Veristack.git
cd Veristack
cp .env.example .env

# 必须修改 VERISTACK_AUTH_SECRET：至少 32 位，且不能使用 change-me-in-production
# 同时建议修改 POSTGRES_PASSWORD
# 生成随机密钥：openssl rand -base64 48

docker compose -f deploy/compose/docker-compose.yml up -d --build
```

首次启动时 API 会自动执行 Goose 数据库迁移。启动完成后：

| 服务 | 地址 |
| --- | --- |
| Web 控制台 | <http://localhost:5173> |
| API | <http://localhost:8080> |
| 存活检查 | <http://localhost:8080/healthz> |
| 就绪检查（含数据库） | <http://localhost:8080/readyz> |

查看启动日志：

```bash
docker compose -f deploy/compose/docker-compose.yml logs -f api
```

### 本地开发

需要 **Go 1.27.2+**、**Node.js 22+**、**PostgreSQL 17**。

```bash
# 只启动 PostgreSQL
docker compose -f deploy/compose/docker-compose.yml up -d postgres

# 终端 1：API
cd api
export VERISTACK_DATABASE_DSN='postgres://veristack:veristack@localhost:5432/veristack?sslmode=disable'
export VERISTACK_AUTH_SECRET='<至少 32 位随机值>'
go run ./cmd/server

# 终端 2：Web
cd web
npm ci
npm run dev
```

前端开发服务器会把 `/api` 请求代理到 `http://127.0.0.1:8080`。

<details>
<summary>Windows PowerShell</summary>

```powershell
Copy-Item .env.example .env

cd api
$env:VERISTACK_DATABASE_DSN = 'postgres://veristack:veristack@localhost:5432/veristack?sslmode=disable'
$env:VERISTACK_AUTH_SECRET = '<至少 32 位随机值>'
go run ./cmd/server

# 另开终端
cd web
npm ci
npm run dev
```

</details>

### 首次使用

1. 打开控制台，选择「注册组织」创建工作室与 owner 账号。
2. 新建项目，录入需求并冻结为基线 v1。
3. 需要调整范围时发起变更申请，由客户确认后生成 v2。
4. 从基线创建交付版本，补充证据后生成客户验收链接。

## 架构

| 层 | 选型 | 职责 |
| --- | --- | --- |
| 后端 | Go 1.27 + Chi | 模块化单体；Handler / Service / Repository 分层 |
| 数据访问 | PostgreSQL 17 + pgx + sqlc + Goose | 类型安全查询、版本化迁移、租户隔离 |
| 存储 | 本地文件系统 | 证据附件与 Evidence Pack；S3 驱动在路线图中 |
| 前端 | React 19 + TypeScript + Vite | TanStack Router / Query / Table，明暗双主题 |
| 样式 | Tailwind CSS 4 + 语义化 CSS 变量 | 组件内不写死色值，统一主题 token |
| 完整性 | SHA-256 + Ed25519 | 清单摘要、交付快照签名、离线核验 |
| 契约 | OpenAPI 3.1 | 57 条路径、69 个操作，逐字段契约测试已接入 CI |

```mermaid
flowchart TB
  Web["Web 控制台<br/>React 19"] --> API["HTTP API<br/>Go + Chi"]
  CLI["veristack CLI<br/>离线核验"] --> Pack["Evidence Pack v2<br/>清单 + 附件 + 签名"]
  API --> PG[("PostgreSQL 17")]
  API --> Store["证据存储<br/>本地文件系统"]
  API --> Worker["Outbox Worker"]
  Worker --> SCM["GitHub / GitLab / Gitee"]
  Pack --> Store
```

## 仓库结构

```text
veristack/
├── api/                  # Go API、Evidence Pack CLI、迁移与 OpenAPI 契约
├── web/                  # React 控制台
├── deploy/               # Docker Compose、镜像与部署文档
├── docs/                 # 数据库、测试规范、路线图与 README 资产
├── AGENTS.md             # 全仓库开发约定
├── CONTRIBUTING.md       # 贡献流程
├── SECURITY.md           # 安全问题报告渠道
└── README.md
```

后端与前端的分层约定分别见 [`api/agents.md`](api/agents.md) 和 [`web/agents.md`](web/agents.md)。

## Evidence Pack：可离线核验的交付证据

Evidence Pack 是一份可以脱离 Veristack 服务端独立核验的交付快照：

```text
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
# 在仓库根目录构建 CLI；输出为 api/veristack，Windows 为 api\veristack.exe
make cli-build
./api/veristack verify ./delivery-pack
```

```text
交付：官网改版 v1（基线 v2）
清单摘要：a08b27008e6aaae1...

[通过] 交付清单摘要：清单内容与元数据一致
[通过] 交付清单签名：签名有效

结论：全部核验通过，交付内容与签名均有效
```

修改、删除或替换签名范围内的任一内容，摘要与签名校验都会失败，CLI 返回非零退出码，可直接接入流水线做交付卡口。

格式契约见 [`api/contracts/evidence-pack/v2.md`](api/contracts/evidence-pack/v2.md)。

## 第三方平台接入

为每个项目配置回调地址：

```text
POST https://your-veristack/api/v1/webhooks/{github|gitlab|gitee}/projects/{projectId}
```

| 平台 | 校验方式 | 环境变量 |
| --- | --- | --- |
| GitHub | `X-Hub-Signature-256` HMAC-SHA256 | `VERISTACK_WEBHOOK_GITHUBSECRET` |
| GitLab | `X-Gitlab-Token` | `VERISTACK_WEBHOOK_GITLABSECRET` |
| Gitee | `X-Gitee-Token` | `VERISTACK_WEBHOOK_GITEESECRET` |

事件先完整落库，再由后台 Worker 异步处理，不阻塞第三方回调；重复投递通过幂等约束去重。

## AI 的边界

Veristack 里 AI 只承担它擅长的工作：提取候选需求、分析变更影响、生成交付摘要。

**AI 不会做这些事：**

- 修改正式需求基线
- 判定客户已经验收
- 把未运行的测试标记为通过
- 把推测的开发行为记录为真实事实

所有 AI 产出都是待确认草稿，必须经人工采纳或忽略后才改变状态。未配置模型密钥时，系统使用本地确定性规则，功能依然可用且结果可复现。

## 开发与测试

```bash
make help          # 查看全部命令
make api-run       # 本地运行 API
make api-vet       # go vet 静态检查
make api-test      # 后端单元测试
make cli-build     # 编译 Evidence Pack 核验 CLI
make web-install   # 安装前端依赖
make web-lint      # ESLint
make web-build     # 类型检查 + 生产构建
make sqlc-gen      # 根据 SQL 重新生成类型安全代码
make compose-up    # 启动完整容器环境
```

提交 PR 前至少运行：

```bash
make api-vet
make api-test
make web-lint
make web-build
```

CI 还会检查 `gofmt`、OpenAPI 逐字段契约、sqlc 生成代码无漂移。工作流见 [`.github/workflows/ci.yml`](.github/workflows/ci.yml)。

## 文档

| 文档 | 内容 |
| --- | --- |
| [部署指南](deploy/docs/deployment.md) | Docker、环境变量、HTTPS、备份恢复与常见问题 |
| [数据库说明](docs/database.md) | 本地迁移、注释核验与演示数据写入 |
| [单元测试规范](docs/unit-test.md) | 测试范围、覆盖率要求与豁免流程 |
| [OpenAPI 3.1](api/contracts/openapi/openapi.yaml) | 57 条路径、69 个操作的接口契约 |
| [Evidence Pack v2](api/contracts/evidence-pack/v2.md) | Pack 结构与签名载荷 |
| [路线图](docs/next-steps.md) | P0 / P1 / P2 任务、验收标准与后续计划 |

## 状态与路线图

**已具备：**

- [x] 组织、项目、成员、需求基线、变更审批
- [x] 交付清单、证据时间线、客户验收、异议闭环、审计事件
- [x] GitHub / GitLab / Gitee Webhook、JUnit XML 导入、Outbox 异步任务
- [x] Evidence Pack v2、Ed25519 签名、独立核验 CLI、OpenAPI 契约测试
- [x] React 控制台、明暗双主题、Docker Compose 私有化部署

**接下来：**

- [ ] 生产级 S3 存储驱动
- [ ] PostgreSQL 集成测试与 Playwright 端到端测试
- [ ] 正式 Release、CHANGELOG、多架构镜像与备份恢复演练
- [ ] OIDC / SSO、监控告警与更完整的通知事件
- [ ] 真实团队试点与商业验证

## 贡献

欢迎提交 Issue 和 Pull Request。开始之前请阅读：

- [CONTRIBUTING.md](CONTRIBUTING.md)：开发流程、提交规范与 PR 检查项
- [AGENTS.md](AGENTS.md)：全仓库通用开发约定
- [api/agents.md](api/agents.md) / [web/agents.md](web/agents.md)：后端与前端的分层约定

安全问题请不要公开提交 Issue，按 [SECURITY.md](SECURITY.md) 私下报告。

## 许可证

[Apache License 2.0](LICENSE) © 2026 Veristack contributors
