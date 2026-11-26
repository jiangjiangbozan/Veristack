# Veristack（验迹）需求分析与实现规划

## 1. 产品定位

来源：`docs/chatGPT-chat-history.md`。

产品中文名为「**验迹**」，英文名为 `Veristack`。代码、文档、API 路径与界面标识以英文 `Veristack` 为主，中文名用于面向中文用户的介绍文案与品牌展示，不再使用「交付有据」「有据」等旧称。

Veristack 是一个**独立于 Git、CI、测试框架和 Agent 平台的软件交付证据层**。它消费这些工具产生的证据，整理成客户能够理解、审核和确认的交付记录。

- 核心用户：2～20 人的软件定制工作室、外包团队、自由职业开发者及其客户。
- 核心对象：**一次软件交付**，而不是一个 Agent Session。
- 核心价值：需求有边界、变更有确认、交付有凭据、验收可追溯。
- 明确不做：Agent 任务调度器、自动化测试框架、通用 AI 平台、一期桌面应用。

产品要形成闭环：开发者证明交付了什么，客户确认验收了什么，双方对尚未满足的条件留下明确记录。

## 2. 当前框架现状盘点

> 本节于 Alpha 收尾时更新，与仓库实际代码一致。历史阶段划分见第 9 节。

| 目录 | 现状 | 缺口 |
| --- | --- | --- |
| `web/` | React 19.3 + TypeScript 6 + Vite 8 + Tailwind 4，TanStack Query/Router/Table，9 个业务功能模块，明暗双主题 | 缺 7 个功能入口：项目编辑、成员管理、AI 交付摘要、AI 变更影响、基线对比、CI 报告列表、证据附件下载 |
| `api/` | Go 1.27.2 + Chi，10 个业务模块，约 60 个接口，19 张表，96 条 sqlc 查询，Goose 迁移 | 缺审计查询接口、Pack 下载接口、通知模块；0 个测试文件；无 CI 工作流 |
| `openspec/` | 已初始化 `config.yaml`，`changes/` 与 `specs/` 为空 | 尚未沉淀任何规格 |
| `docs/` | `chatGPT-chat-history.md`、`implementation-plan.md`、`todo.md`（商业化审查）、`next-steps.md`（可执行待办） | 后续按需补充架构与接口文档 |
| `web/agents.md` | 已按 React + TypeScript + Vite 重写，并补充 UI/UX 视觉约定 | 无 |
| `api/agents.md` | 已按 Go + Chi + PostgreSQL 重写 | 无 |
| 根目录 `agents.md` | 已改为语言无关的通用原则 + 产品领域约定（34 条） | 无 |

当前成熟度为 **Alpha**：核心业务链路已跑通，但存在 7 项 P0 级可信性缺陷与 0 个自动化测试，尚不能正式商用。完整待办与执行顺序见 [`next-steps.md`](next-steps.md)。

## 3. 关键决策

以下决策已确认，作为后续实现的基础约束。

### 3.1 后端 HTTP 框架：Chi

- 现状：`go.mod` 已引入 `github.com/go-chi/chi/v5 v5.5.0`。
- 结论：**使用 Chi**，与对话历史建议一致。路由按资源组织，中间件链统一处理请求 ID、鉴权、错误恢复与日志。
- OpenAPI 契约代码生成可在 Chi 之上叠加 `oapi-codegen`，不冲突。

### 3.2 数据库：PostgreSQL

- 结论：**统一使用 PostgreSQL，不使用 MongoDB**。
- 理由：核心价值是「版本化基线与验收链」，需要强一致的外键与事务；证据项、Webhook 原始事件、CI 报告等异构数据用 `JSONB` 字段承载即可，无需引入第二套存储。
- 访问方式：`pgx` 驱动 + `sqlc` 生成类型安全代码，表结构变更通过 Goose 迁移脚本管理。
- 私有化部署只需一个 PostgreSQL 服务，降低运维成本。

### 3.3 认证与授权

- 一期：邮箱 + 密码登录，JWT 会话；客户通过带签名 token 的分享链接访问验收门户，不占用组织席位。
- 二期：接入 OIDC（对话历史已有规划）。
- 权限模型：组织 owner/admin、项目 developer/reviewer、外部 customer。所有写接口必须鉴权，默认最小范围授权。

### 3.4 文件存储

- 一期：通过 `ObjectStore` 接口抽象，默认本地文件系统实现，便于 Docker Compose 私有化部署。
- 二期：实现 S3 兼容实现（MinIO / AWS S3）。
- 所有文件写入即计算 SHA-256；交付清单可选 Ed25519 签名。

### 3.5 命名

- 产品中文名「**验迹**」，英文名 `Veristack`。
- 代码、文档、API 路径与界面标识以英文 `Veristack` 为主；中文名用于面向中文用户的介绍文案与品牌展示。
- 不再使用「交付有据」「有据」等旧称。

## 4. 功能性需求（补充后）

### 4.1 组织与用户

- 组织创建、成员邀请（邮箱）、成员移除。
- 用户注册、登录、退出、修改密码。
- 角色：owner、admin、developer、reviewer、customer。customer 只能访问被分享的项目/交付。
- 用户列表按姓名升序展示，禁用用户不可登录。

### 4.2 项目管理

- 项目字段：名称、客户名称、合同编号、描述、开始日期、结束日期、状态、归档标记。
- 项目成员及角色分配。
- 项目列表：按更新时间倒序，支持按状态筛选与关键词搜索。
- 项目概览：需求数、待确认变更数、交付数、待验收交付数。

### 4.3 需求范围基线（核心）

- 需求条目字段：编号（`REQ-001`）、标题、描述、优先级、验收条件列表、状态。
- 基线版本：`v1`、`v2`……冻结后不可编辑，修改必须通过变更申请生成新版本。
- 基线对比：展示两个版本之间新增、删除、修改的需求。
- 需求状态：待开发、开发中、已完成、已验收。

### 4.4 变更申请

- 字段：关联需求、变更类型（新增/修改/删除）、原因、影响说明、发起人、状态。
- 客户确认或拒绝，并填写意见；拒绝后可撤回重新编辑。
- 确认后自动生成新的范围基线版本，原版本保持只读。
- 状态：待客户确认、已确认、已拒绝、已撤回。
- AI 仅提供影响分析建议，不得自动修改正式基线。

### 4.5 里程碑

- 字段：名称、目标日期、关联需求、状态、所属项目。
- 状态：未开始、进行中、已完成。
- 一个交付版本可关联一个或多个已完成里程碑。

### 4.6 交付版本（核心）

- 字段：版本号、关联基线版本、关联里程碑、Git 提交/标签、构建号、创建人、创建时间、状态。
- 交付清单：需求快照 + 验收条件快照 + 关联证据快照，创建后不可覆盖。
- 交付摘要：可由 AI 生成自然语言描述，但必须经人工确认后发布。
- 状态：草稿、待验收、已通过、有异议、已拒绝。

### 4.7 证据

- 类型：Git 提交、CI 报告、截图、录屏、测试报告、文档、人工说明。
- 支持文件上传与外链；上传时限制大小与 MIME 类型，超限返回 400。
- 每条证据计算 SHA-256，保存原始文件名、大小、上传时间、上传人和来源。
- 可关联到需求、验收条件或交付版本，形成时间线。

### 4.8 客户验收门户

- 客户通过签名分享链接进入，无需注册组织账号。
- 逐条验收条件判定：通过/不通过，并填写说明。
- 支持上传异议附件。
- 交付级确认：确认通过或提出异议。
- 异议处理闭环：开发者回复 → 客户复核 → 关闭或再次提出。

### 4.9 审计事件

- 所有写操作记录审计事件：操作人、时间、动作、对象类型与 ID、变更前后摘要、来源 IP。
- 审计事件只追加，不提供删除或更新接口。

### 4.10 通知

- 变更申请待客户确认、交付待验收、异议待处理时发送站内通知。
- 邮件通知作为可选实现，接口留出扩展点。

## 5. 非功能性需求

- **时区**：所有日期时间处理显式使用 `Asia/Shanghai`，不依赖运行环境默认时区。
- **权限**：每个写接口都要鉴权；涉及资源归属的校验放在 Handler 层调用 Service 校验方法完成。
- **边界校验**：所有外部输入在入口处校验，错误以 400 返回，不得泄漏数据库异常。
- **完整性**：文件 SHA-256；交付清单可 Ed25519 签名，供外部独立验证。
- **分页**：所有列表接口统一分页，参数与响应结构一致。
- **日志**：结构化日志，包含请求 ID、用户 ID、动作与耗时。
- **部署**：Docker Compose 一键启动；前端构建产物为静态文件。
- **响应式**：管理端和客户门户在移动端可用。

### UI/UX 设计要求（用户补充需求）

UI 必须现代化、精致、美观，符合「可信、严谨、专业」的产品定位，避免模板化后台管理系统的观感。

- **视觉参考**：对标 Trae、CodeBuddy 等主流 Agent Web 端以及 ChatGPT 网页的设计感——大面积留白、克制的中性色、清晰的信息层级、精确的间距与圆角、轻量阴影与细腻过渡。靠排版与空间关系建立秩序，而非粗边框和强色块。
- **明暗双主题**：必须同时支持浅色与深色两套配色，通过语义化 CSS 变量统一定义色值，禁止在组件里写死颜色；主题选择持久化到本地并在首屏加载时生效，避免闪烁。两套主题都要满足 WCAG AA 对比度，状态色（成功、警告、错误、进行中）在两套主题下均可辨识。
- **图标与字体**：图标风格统一（线性、同尺寸、同视觉重量），禁止混用不同风格图标库；中文字体使用系统无衬线字体栈，数字与代码使用等宽字体。
- **关键界面**：项目概览突出待确认变更与待验收交付；基线对比用并排或逐条高亮展示差异并明确标记冻结版本；交付证据以时间轴串联 Git、CI、截图与测试报告；客户验收门户面向非技术用户，步骤清晰、文案通俗。
- **状态与反馈**：空状态提供指导性文案，加载态使用骨架屏，禁止空白页面或生硬提示；关键写操作（冻结基线、提交验收、确认交付）必须有二次确认与明确结果反馈。
- **动效**：过渡动画控制在 150～250ms，仅用于状态变化与层级切换，不做装饰性动画；表单错误紧邻字段展示，不用 toast 代替内联校验。

详细执行约定见 `web/agents.md` 的「视觉与交互设计」章节。

## 6. 数据模型

使用 PostgreSQL 表结构。`JSONB` 字段承载异构扩展数据（验收条件、CI 报告、Webhook 原始事件），核心业务字段保持关系类型与外键约束。

| 表 | 关键字段 | 说明 |
| --- | --- | --- |
| `organizations` | name, created_at, deleted_at | 团队与组织 |
| `users` | email, password_hash, display_name, role, status, organization_id | 用户 |
| `projects` | name, client_name, contract_no, description, start_date, end_date, status, archived, organization_id | 项目 |
| `project_members` | project_id, user_id, role | 项目成员 |
| `scope_baselines` | project_id, version, requirements_json (JSONB), frozen_at, change_request_id | 冻结的版本化基线 |
| `requirements` | baseline_id, code, title, description, priority, status, acceptance_criteria (JSONB) | 需求条目 |
| `change_requests` | project_id, baseline_id, type, reason, impact, status, requester_id, decided_by_id | 需求变更 |
| `milestones` | project_id, name, target_date, status, requirement_ids (JSONB) | 里程碑 |
| `deliveries` | project_id, version, baseline_version, milestone_ids, git_ref, build_no, status, manifest_hash, manifest_json | 交付版本 |
| `evidence_items` | project_id, delivery_id, requirement_id, type, title, file_key, sha256, source, note, occurred_at | 证据 |
| `approvals` | delivery_id, customer_id, decision, comment, decided_at | 客户确认 |
| `objections` | delivery_id, criterion_id, raised_by_id, content, status, resolved_at | 异议 |
| `acceptance_verdicts` | delivery_id, criterion_key, requirement_code, requirement_title, verdict, comment, decided_by_id, decided_at | 逐条验收裁定 |
| `objection_replies` | objection_id, author_id, content, created_at | 异议回复 |
| `audit_events` | actor_id, action, object_type, object_id, before, after, ip, occurred_at | 审计事件 |
| `share_tokens` | delivery_id, token_hash, expires_at, revoked_at | 客户分享链接 |
| `outbox_tasks` | event_type, payload, project_id, delivery_id, status, attempts, next_run_at | 异步任务 |
| `webhooks` | provider, event_type, project_id, delivery_id, signature, payload, status, received_at | 外部事件原始记录 |
| `ci_reports` | project_id, delivery_id, provider, build_no, git_ref, total_count, passed_count, failed_count, skipped_count, duration_ms, summary | CI 测试报告汇总 |
| `ai_suggestions` | project_id, kind, status, content, model, applied_at | AI 建议草稿 |

设计约束：

- 需求基线冻结后禁止直接修改，变更只能通过新增 `change_requests` 并生成新 `scope_baselines` 完成；整个流程放在同一数据库事务中保证一致性。
- 交付清单一旦生成即只读，后续补充证据以关联方式追加，不改写历史清单。
- 软删除统一使用 `deleted_at timestamptz NULL`，部分索引条件为 `WHERE deleted_at IS NULL`，查询条件与索引保持一致。
- 时间字段统一使用 `timestamptz`，写入与比较时显式指定 `Asia/Shanghai`，禁止依赖数据库或服务器默认时区。
- 版本化表通过 `(project_id, version)` 唯一约束防止重复版本。

## 7. API 设计

统一前缀 `/api/v1`。URL 表示资源，HTTP 方法表示动作。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/auth/login` | 登录 |
| POST | `/organizations` | 创建组织 |
| GET/POST | `/projects` | 项目列表 / 创建 |
| GET/PUT | `/projects/{id}` | 项目详情 / 更新 |
| GET | `/projects/{id}/overview` | 项目概览统计 |
| GET/POST | `/projects/{id}/scope-baselines` | 基线列表 / 冻结新基线 |
| GET | `/scope-baselines/{id}/diff` | 基线对比 |
| GET/POST | `/projects/{id}/change-requests` | 变更列表 / 发起变更 |
| POST | `/change-requests/{id}/decision` | 客户确认或拒绝 |
| GET/POST | `/projects/{id}/milestones` | 里程碑列表 / 创建 |
| GET/POST | `/projects/{id}/deliveries` | 交付列表 / 创建交付 |
| GET | `/deliveries/{id}/manifest` | 交付清单 |
| POST | `/deliveries/{id}/submit` | 提交客户验收 |
| GET/POST | `/deliveries/{id}/evidence` | 证据列表 / 上传证据 |
| POST | `/deliveries/{id}/decision` | 客户确认或提出异议 |
| GET | `/deliveries/{id}/audit-events` | 审计事件 |
| POST | `/projects/{id}/deliveries` | 创建交付版本并冻结清单 |
| GET | `/deliveries/{id}` | 交付详情 |
| GET | `/deliveries/{id}/manifest` | 交付清单快照 |
| POST | `/deliveries/{id}/submit` | 提交客户验收 |
| GET/POST | `/deliveries/{id}/evidence` | 证据时间线 / 登记证据 |
| GET | `/evidence/{id}/file` | 下载证据附件 |
| POST | `/deliveries/{id}/share-token` | 生成客户分享链接 |
| GET | `/deliveries/{id}/acceptance` | 验收概览（裁定与异议） |
| POST | `/objections/{id}/replies` | 回复客户异议 |
| POST | `/objections/{id}/resolve` | 关闭客户异议 |
| GET | `/share/{token}` | 客户门户读取交付 |
| POST | `/share/{token}/verdicts/{criterionKey}` | 保存单条验收裁定 |
| POST | `/share/{token}/decision` | 客户提交最终验收结论 |
| POST | `/webhooks/{provider}/projects/{projectId}` | 接收第三方事件（签名校验，公开接口） |
| GET | `/projects/{id}/webhooks` | Webhook 事件列表 |
| GET/POST | `/projects/{id}/ci-reports` | CI 报告列表 / 导入测试报告 |
| GET | `/projects/{id}/evidence` | 项目全部证据时间线 |
| POST | `/projects/{id}/ai/requirements` | 从文档提取候选需求建议 |
| POST | `/projects/{id}/ai/change-impact` | 生成变更影响分析建议 |
| POST | `/projects/{id}/ai/delivery-summary` | 生成交付摘要建议 |
| GET | `/projects/{id}/ai/suggestions` | AI 建议列表 |
| POST | `/ai/suggestions/{id}/apply` | 采纳 AI 建议 |
| POST | `/ai/suggestions/{id}/dismiss` | 忽略 AI 建议 |

分页统一使用 `page`、`pageSize`，响应统一包含 `items`、`total`、`page`、`pageSize`。

## 8. 前端结构与页面

```text
web/src/
├── app/                 # 应用入口、路由、Providers
├── layouts/             # 管理端布局、客户门户布局
├── components/ui/       # 通用按钮、表单、表格等封装
├── features/
│   ├── auth/
│   ├── projects/
│   ├── scope/
│   ├── changes/
│   ├── milestones/
│   ├── deliveries/
│   ├── evidence/
│   └── acceptance/
├── services/            # API 请求封装与类型定义
├── shared/              # 常量、工具、类型
└── routes/              # 路由声明
```

页面按用户操作路径组织：

1. 登录 → 项目列表 → 项目概览
2. 项目概览 → 需求范围基线 → 基线对比
3. 项目概览 → 变更申请 → 客户确认
4. 项目概览 → 里程碑 → 交付版本 → 证据时间线
5. 交付详情 → 提交验收 → 客户门户逐条验收 → 确认/异议

## 9. 实施路线

### 阶段 0：基础层（本次文档产出后可立即开始）

- 后端：程序入口、配置、日志、错误处理、Chi 中间件链、鉴权、pgx + sqlc 数据访问层、Goose 迁移、统一响应与分页、`ObjectStore` 接口。
- 前端：路由与布局、明暗双主题与 UI 基础组件、TanStack Query、API 客户端骨架、类型定义。
- 基础设施：Docker Compose（API + Web + PostgreSQL + MinIO 可选）、Makefile、OpenAPI 契约骨架。

### 阶段 1：交付业务核心

- 组织与用户、项目与成员、需求范围基线、变更申请及确认、里程碑。
- 客户门户的最小可读版本（查看基线与变更）。

### 阶段 2：证据与验收（已完成）

- 交付版本与清单：创建交付时冻结基线需求与验收条件，生成 SHA-256 清单摘要，清单不可覆盖。
- 证据上传与关联：附件经对象存储落盘并计算内容摘要，人工说明型证据以备注承载，形成按时间倒序的证据时间线。
- 客户分享链接：生成一次性明文令牌，库中只保存摘要，链接可设置有效期。
- 逐条验收：客户对每条验收条件给出通过或不通过并填写说明，裁定记录裁定人与时间。
- 确认与异议闭环：客户确认交付或提出异议，开发者回复，客户复核后关闭异议。
- 审计事件：交付、证据、验收与异议的全部写操作均已记录审计事件。

### 阶段 3：自动化（已完成）

- Webhook 接入：GitHub/GitLab 事件按项目维度配置回调地址，HMAC-SHA256 签名校验，先落库再异步处理。
- Outbox 任务队列：事件写入即建任务，后台 Worker 批量领取处理，失败按次数退避重试。
- 提交自动归档：解析 push 事件，按提交信息中的需求编号关联到需求，生成 Git 提交类型证据。
- CI 报告导入：解析标准 JUnit XML，汇总通过、失败、跳过与耗时，作为交付证据来源。
- AI 建议层：模型适配接口 + OpenAI 兼容实现 + 本地确定性兜底，产出需求提取、变更影响与交付摘要建议。
- 人工确认闭环：所有 AI 产出均为草稿，采纳或忽略后才改变状态，绝不自动修改正式基线或判定验收。

### 阶段 4：开放生态（已完成）

- Evidence Pack 格式契约（`api/contracts/evidence-pack/v1.md`）：可脱离服务端独立核验的交付证据快照，独立版本化。
- Evidence Pack 导出：`POST /deliveries/{id}/pack` 打包交付清单与证据附件，生成 SHA-256 摘要并按配置附带 Ed25519 签名。
- 独立核验 CLI（`api/cmd/veristack`）：`veristack verify` 校验清单摘要、附件摘要与签名，异常时返回非零退出码，便于接入流水线卡口；`veristack keygen` 生成签名密钥对。
- 第三方平台扩展：Webhook 支持 GitHub、GitLab 与 Gitee，Gitee 使用 `X-Gitee-Token` 校验。
- OpenAPI 3.1 完整契约（`api/contracts/openapi/openapi.yaml`）：覆盖 39 个接口、17 个数据模型与 10 个资源分组。
- 私有化部署文档（`deploy/docs/deployment.md`）：环境要求、环境变量、HTTPS、第三方接入、备份恢复与常见问题。

## 10. 验收口径

每个阶段结束时，必须用一条真实业务链路验证，而不是只看接口是否返回 200：

1. 阶段 1：创建项目 → 录入需求并冻结 v1 → 发起变更 → 客户确认 → 生成 v2，且 v1 未被改写。
2. 阶段 2：从 v2 创建交付 → 上传截图和测试报告 → 生成清单 → 客户逐条验收 → 对一条不通过项提出异议 → 开发者回复 → 客户复核关闭。
3. 阶段 3：推送一个 Git 提交，自动关联到对应需求，并在交付证据时间线中出现。
4. 全链路必须能在审计事件中按时间顺序还原，且所有时间显示为上海时区。
5. 每一阶段的前端验收都要在浅色与深色两套主题下逐一走查：无硬编码色值、无对比度不足、无布局错位，关键写操作有二次确认与结果反馈。
