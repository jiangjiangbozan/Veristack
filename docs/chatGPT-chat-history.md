# **一、先调整一个很重要的产品判断**

我在重新核查项目命名和技术路线时，发现了一个不能忽略的新竞品：

taskproof。

它已经提出了非常接近的产品思路：任务派发给 Agent、通过独立验收命令判断是否完成、记录执行过程，并提供 CLI、本地 API、浏览器看板和桌面控制台。

[image](https://www.google.com/s2/favicons?domain=https://pypi.org\&sz=32)

PyPI+1



另外，Shipmark已经将 AI 编程会话跟踪、工作记录、客户需求确认和交付沟通结合起来。

[image](https://www.google.com/s2/favicons?domain=https://shipmark.app\&sz=32)

Shipmark+1



这意味着我会修改之前的定位：

**不能再把「Agent 执行记录 + 独立验收 + 证据包」本身当作独家卖点。**

我更希望新项目进一步聚焦：

> **软件交付的可验证性：把客户需求、开发过程、代码变更、测试结果和客户验收，变成一份双方都能独立核验的交付记录。**

它不只是服务开发者，而是连接开发团队与最终接收软件的人。

这会带来一个更明确的业务闭环：

开发者证明自己交付了什么，客户证明自己验收了什么，双方能针对尚未满足的验收条件留下明确记录。

这比单纯统计 Agent 操作更有产品意义，也更有机会形成独立的商业场景。

# **二、重新给这个项目取名字**

我建议不要继续使用 ShangShu、Proof 等已有项目的品牌延伸，也不建议名字过度强调 AI。

产品未来很可能还会服务于人工开发、传统软件交付、外包验收和企业审计。

我希望名字表达三个意思：

- Trace：过程可追溯。
- Proof：结果有依据。
- Delivery：交付能够被验证。

我初步筛选了下面几个名字。需要说明，我做的是公开网页和项目检索，不等于已经完成商标、域名和 GitHub 全面的可用性清查。

| 名称            | 含义                 | 品牌感觉    |
| ------------- | ------------------ | ------- |
| **Veristack** | Verifiable + Stack | 技术平台、专业 |
| **Tracewell** | Trace + Well       | 可信、易记   |
| **ShipProof** | 交付 + 证明            | 产品定位直观  |
| **Relogic**   | Record + Logic     | 工程化、工具感 |
| **Evidock**   | Evidence + Dock    | 证据汇聚站   |
| **Delivera**  | Delivery 的品牌化表达    | 商业化、国际化 |

我会优先考虑 **Veristack** 这个方向，但目前不把它当成已确认可注册的名字。

更重要的是，我不建议选 `ProofRail`、`ShipTrace`、`TaskProof` 这些名字：已经能检索到相关产品或项目，没必要从第一天就给自己制造品牌混淆。

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

GitHub+2



还有两个项目值得注意：

- LobeHub Acceptance：能够让 Coding Agent 把任务转化为可观察的检查，运行真实产品，并附上截图、录屏和日志。

  [image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

  GitHub

- ShipProof：直接把需求转成真实网页操作检查，保存每一步的截图，支持测试流程重放。

  [image](https://www.google.com/s2/favicons?domain=https://tryshipproof.com\&sz=32)

  Before you ship, prove it works.



因此，新项目应该明确不做自动化测试框架，也不做 Agent 任务调度器。

我会把它设计为一个**独立于 Git、CI、测试框架和 Agent 平台的交付证据层**。它消费这些工具产生的证据，再把它们整理成客户能够理解、审核和确认的交付记录。

这决定了接下来的技术选择：本地采集端要轻，服务器要能私有化部署，核心数据模型要可版本化，前端要擅长展示时间线、验收状态和证据关联。

# **三、我建议的新项目产品定义**

暂时用中文名称 **「验迹」** 作为工作名称。

# **验迹**

Independent Software Delivery Evidence

**让每一次软件交付，都有可以独立核验的依据。**

一个独立的、开源优先、支持本地优先和私有化部署的软件交付证据平台。

[FDA, EU MDR, & ISO Compliance Software & Application Lifecycle Management (ALM) - Ketryx](https://images.openai.com/static-rsc-4/dRpo3RDEEK5KqjYsirUn-TjhIxuAX6cLe0w05pGuOAn4x5e2Ev33WfJgPAaBfhUD7qR6BO3Oj7ifcYjpCz9QtXji7Xu6XqlT8LRkoTCnSlNx7yzOHJa56iRqC7ImXew6C2jquiJxyIi3f9XVaLab8OH85H2TkCs41vcGrJkA-PI?purpose=inline)

[ketryx.com](https://www.ketryx.com/?utm_source=chatgpt.com)

**需求有依据**

原始需求与验收条件

**变更有依据**

代码版本与开发记录

**验证有依据**

测试与运行结果

**验收有依据**

客户确认与异议记录

**核心用户：** 软件定制开发团队、小型软件外包团队、自由职业开发者及其客户。

**核心产品对象：** 一次软件交付，而不是一个 Agent Session。

一个交付版本对应一个不可随意覆盖的证据快照。

这是整个产品的数据模型和产品设计应该围绕的中心。

# **四、最适合这个项目的技术栈**

我不建议为了追求技术新颖而堆框架。这个项目最重要的技术特点是：

1. Web 端需要有复杂的项目、需求、交付、审批关系。
2. 后端需要处理 Git、CI、第三方系统的 Webhook。
3. 需要可靠的文件上传、版本管理、签名和审计记录。
4. 未来可能有本地采集 CLI，必须支持 Windows、Linux、macOS。
5. 需要支持 Docker 私有化部署。

基于这些要求，我建议采用 **Go + React + PostgreSQL**。

# **Recommended Tech Stack**

推荐方案

**Frontend**

React

TypeScript

Vite

Tailwind CSS

shadcn/ui

TanStack Query + TanStack Router，复杂表格使用 TanStack Table。

**Backend**

Go 1.27.x

Chi

pgx

sqlc

OpenAPI

模块化单体架构，REST API，类型安全的 SQL 访问。

**Data & Storage**

PostgreSQL

SQLite

S3 API

PostgreSQL 保存业务及证据元数据，S3 兼容存储保存附件和交付材料，SQLite 用于未来本地采集缓存。

**Integration**

GitHub

GitLab

Gitee

GitHub Actions

Playwright

以 Webhook、标准测试报告和文件导入为主。

**Security & Deployment**

Ed25519

SHA-256

OIDC

Docker Compose

GitHub Actions

下面说明为什么这样选。

## **1. 后端：Go，而不是 Spring Boot**

首选 Go

我建议你直接用 Go 开发服务端。

主要原因不是 Go 比 Java 更好，而是这个项目未来存在两种程序：

- 运行在服务器上的 API Server。
- 运行在开发者电脑上的本地 Collector。

两者可以使用同一套 Go 核心模型、校验逻辑、签名模块和 Git 适配逻辑。

例如：

```
youju-server
    ├── HTTP API
    ├── Project Service
    ├── Scope Service
    ├── Acceptance Service
    ├── Evidence Service
    └── Integration Service

youju-cli
    ├── Git Collector
    ├── Agent Hooks
    ├── Evidence Generator
    ├── Local Cache
    └── Signature Verifier
```

未来可以从同一个代码仓库构建两个不同的可执行程序。

具体组件：

| 组件         | 选择                      | 职责                 |
| ---------- | ----------------------- | ------------------ |
| HTTP       | Chi                     | 路由与中间件             |
| PostgreSQL | pgx                     | 数据库连接              |
| SQL        | sqlc                    | 从 SQL 生成 Go 类型安全代码 |
| Migration  | Goose                   | 数据库版本管理            |
| Validation | go-playground/validator | 数据校验               |
| Config     | Viper 或 koanf           | 配置管理               |
| Logging    | slog                    | 结构化日志              |
| API Schema | OpenAPI 3.1             | API 契约             |

Go 官方发布记录已经进入 1.27 系列，可以采用当前受支持的 Go 1.27.x 工具链。

[image](https://www.google.com/s2/favicons?domain=https://go.dev\&sz=32)

The Go Programming Language+1



我不建议一开始使用微服务，也不建议引入 Kafka。

你的第一版只需要一个模块化单体：

```
HTTP Request
     |
     v
Router / Middleware
     |
     v
Application Service
     |
     +---- Project
     |
     +---- Scope
     |
     +---- Change Request
     |
     +---- Delivery
     |
     +---- Evidence
     |
     +---- Acceptance
     |
     v
PostgreSQL
```

不同模块有清晰的边界，未来需要拆分时再拆。

## **2. 前端：React + TypeScript + Vite**

我更推荐 React，而不是再选择 Angular。

原因：

- 这个产品主要是 B 端管理后台和客户验收门户。
- React 的表格、复杂交互、文件预览、工作流 UI 生态成熟。
- shadcn/ui 可以快速建立一致的产品设计系统。
- TanStack Query 适合服务端数据缓存和异步状态管理。
- Vite 开发体验好，部署只需要静态文件。

建议结构：

```
apps/web/
├── src/
│   ├── app/
│   ├── layouts/
│   ├── components/
│   ├── features/
│   │   ├── projects/
│   │   ├── requirements/
│   │   ├── changes/
│   │   ├── milestones/
│   │   ├── deliveries/
│   │   ├── evidence/
│   │   └── acceptance/
│   ├── routes/
│   ├── services/
│   └── shared/
```

我不建议 Next.js。

不是 Next.js 不好，而是这个产品的核心不是 SEO 内容网站，主要是登录后的业务系统。没有必要为了 SSR 引入额外的服务端运行环境。

Tauri 官方也推荐使用 Vite 构建 SPA 前端。未来真的需要桌面应用时，可以将同一套前端接入 Tauri 2，而不用重写整个界面。

[image](https://www.google.com/s2/favicons?domain=https://tauri.app\&sz=32)

Tauri+1



不过，我建议第一版不要开发桌面应用。

响应式 Web 就够了。

## **3. 数据库：PostgreSQL**

我会明确选择 PostgreSQL，而不是 MySQL。

原因是业务数据和证据数据有不同的结构。

例如：

- 项目、客户、合同、里程碑：强关系数据。
- Git Commit、Webhook：结构化事件。
- Agent Session：不同厂商的数据结构不统一。
- CI 报告：具有可扩展的嵌套结构。
- Evidence Manifest：需要保存结构化 JSON。

PostgreSQL 的 JSONB 能够容纳不同来源的扩展数据，同时核心业务字段仍然保留关系模型。

建议核心表：

| 表名                  | 作用       |
| ------------------- | -------- |
| organizations       | 团队与组织    |
| users               | 用户       |
| projects            | 项目       |
| scope_baselines     | 项目初始范围基线 |
| requirements        | 需求条目     |
| change_requests     | 需求变更     |
| milestones          | 交付里程碑    |
| deliveries          | 交付版本     |
| acceptance_criteria | 验收标准     |
| evidence_items      | 证据记录     |
| evidence_links      | 证据与需求关联  |
| approvals           | 审核和客户确认  |
| audit_events        | 审计事件     |

有一个设计原则：

**验收标准不能直接被覆盖修改，必须版本化。**

比如：

```
项目 A
  |
  +--- Scope v1
  |      |
  |      +--- Requirement 1
  |      +--- Requirement 2
  |
  +--- Change Request 001
  |      |
  |      +--- 客户确认
  |      +--- Scope v2
  |
  +--- Delivery v1
         |
         +--- Evidence Pack
         +--- Customer Approval
```

这样才能避免需求发生变化之后，旧交付记录也被悄悄修改。

## **4. 证据文件与异步任务**

附件包括截图、录屏、测试报告、合同、交付包等，不要全部放到 PostgreSQL 的 BYTEA 字段中。

建议：

- PostgreSQL：业务元数据、关联关系、审批记录。
- S3 兼容存储：文件、附件、导出包。
- SHA-256：内容摘要。
- Ed25519：交付清单签名。

异步任务第一版直接使用 PostgreSQL 的任务队列或者简单的 Outbox + Worker。

例如：

```
GitHub Webhook
      |
      v
接收事件
      |
      v
保存原始事件
      |
      v
创建异步处理任务
      |
      v
关联项目、需求、交付
      |
      v
更新交付证据
```

暂时不需要 Redis、RabbitMQ、Kafka 三件套一起上。服务器不会因为少装三个中间件就失去存在的意义。

## **5. AI 应该放在哪里？**

我反而建议你不要把 AI Agent 当成整个系统的核心执行引擎。

AI 只承担它擅长的工作：

| 场景     | AI 的职责          |
| ------ | --------------- |
| 导入需求文档 | 提取候选需求和验收条件     |
| 需求变更   | 对比新旧要求，提出影响分析   |
| Git 变更 | 总结代码改动关联的业务需求   |
| 交付检查   | 提示缺少的验收材料       |
| 客户反馈   | 归纳反馈、建议创建缺陷或变更单 |
| 最终报告   | 生成自然语言交付摘要      |

但以下操作不能由 AI 擅自决定：

- 修改正式需求基线。
- 判定客户已经验收。
- 把未运行的测试标记为通过。
- 将推测的开发行为记录为真实事实。

LLM 输出只是建议，需要经过确定性校验和人工确认。

第一版甚至不需要 LangChain、LangGraph、向量数据库。直接建立模型适配接口即可。

# **五、整体项目架构**

把它画成实际工程架构，大概是这样：

## **有据 · 整体系统架构**

**用户入口**

**开发团队 Web**

项目、需求、变更、交付

**客户验收门户**

查看证据、确认、提出异议

**Go API Server**

Project

Scope

Change Request

Milestone

Delivery

Evidence

Acceptance

Audit

**PostgreSQL**

关系数据、版本化记录

**Object Storage**

合同、附件、交付包

**外部数据接入**

GitHub / GitLab / Gitee

CI / Playwright / Allure

Agent Hooks / CLI

文件上传 / 人工审批

# **六、代码仓库应该怎么规划？**

我推荐 Monorepo，但不是把所有代码塞进一个工程目录。

```
youju/
│
├── apps/
│   └── web/                    # React + Vite
│
├── cmd/
│   ├── server/                 # API Server
│   └── youju/                  # 本地 CLI (后期)
│
├── internal/
│   ├── project/
│   ├── scope/
│   ├── change/
│   ├── milestone/
│   ├── delivery/
│   ├── evidence/
│   ├── acceptance/
│   ├── integration/
│   │   ├── github/
│   │   ├── gitlab/
│   │   └── gitee/
│   ├── storage/
│   ├── identity/
│   └── audit/
│
├── contracts/
│   ├── openapi/
│   └── evidence-pack/
│
├── migrations/
│
├── deploy/
│   ├── docker/
│   └── compose/
│
├── docs/
│   ├── architecture/
│   ├── product/
│   └── security/
│
└── Makefile
```

`contracts/evidence-pack/` 我建议独立维护版本。

因为未来你可能会有其他产品需要消费这套交付证据格式，甚至可以让其他开发者自行实现兼容客户端。

# **七、开发阶段怎么安排？**

我会严格控制第一版的范围。

## **产品开发路线**

1. **阶段一：交付业务核心**
   - 项目和客户管理
   - 需求范围基线
   - 变更申请及确认
   - 里程碑管理

   第一个可用版本
2. **阶段二：证据与验收**
   - GitHub/GitLab 集成
   - 上传截图和测试报告
   - 验收条件与证据关联
   - 客户分享链接
   - 确认、拒绝及异议记录
3. **阶段三：自动化**
   - AI 需求提取
   - CI 报告自动导入
   - Git 变更自动关联
   - Agent Hook 采集
   - 自动生成交付报告
4. **阶段四：开放生态**
   - Gitee、自托管 GitLab
   - OpenAPI 与 Webhook 开放
   - Evidence Pack 独立验证 CLI
   - Docker 私有化部署
   - 扩展更多第三方工具

我尤其建议你在第一版就把客户验收门户做好。

因为它是产品和普通 Git 统计工具、Agent 调试工具之间最直观的区别。

# **八、最终方案，我帮你拍板**

# **交付有据**

### Software Delivery Evidence & Acceptance

[Damp and Mould Compliance: What the New Legislation Means for Housing Providers and How to Respond | Esuasive](https://images.openai.com/static-rsc-4/m8cU9zerpp2HyBo_ixMVoHawHtllQO3lwYVPzAlaqUgxZu2Tne1ETV5UP_8FdlUVqt_wcMNI6av7quQb_i8suGrXWF0n-DTlYIKqJ0zX0_BC_ILe-PRd0RhvtDYuq6TvRBWJlfD8Ligw1DWCnaq6ooS90sy6iY4b-SEeRclKobE?purpose=inline)

[esuasive.co.uk](https://www.esuasive.co.uk/news-insights/damp-and-mould-compliance-what-the-new-legislation-means-for-housing-providers-and-how-to-respond?utm_source=chatgpt.com)

**品牌定位**

让软件需求、范围变更、交付成果和客户验收形成一条可核验的记录链。

**技术栈**

- Go：后端与未来本地 CLI。
- React + TypeScript + Vite：Web 管理端和客户门户。
- PostgreSQL：业务与证据数据。
- S3：文件与交付材料。
- Docker Compose：部署。
- Ed25519 + SHA-256：完整性与签名。

  **第一批目标用户**

2～20 人的软件定制开发工作室、外包团队及自由职业开发者。

**核心卖点**

需求有边界、变更有确认、交付有凭据、验收可追溯。

有一点我必须强调：这次检索已经发现了多个相近开源项目和商业产品，甚至 Cycle 已经明确在做面向 AI 编程代理的本地证据门禁和交付回执。

[image](https://www.google.com/s2/favicons?domain=https://runcycle.dev\&sz=32)

面向 AI 编程代理的证据门禁式软件交付+2



所以我不建议你再投入大量时间，开发一个通用的 Agent 证据采集平台。当前这个调整后的定位更值得验证，但也不能直接宣称没有竞品。

**我建议把「交付有据」作为暂定产品名称，把「软件外包项目的范围变更与客户验收」作为第一条产品线。**

名字暂未完成商标和域名核验。技术上先用 Go + React 开始，先建立一个完整的需求变更和客户验收闭环。等真实团队愿意使用，再引入更复杂的 Agent 过程采集。

这样它才是一个有独立业务价值的新项目，而不只是又一个给 AI 写代码做记录的工具。