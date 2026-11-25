# API 开发约定（Go + Chi + PostgreSQL）

本文件只约束 `api/` 目录。当前 `go.mod` 已引入 `github.com/go-chi/chi/v5 v5.5.0`；无任何源码，按模块化单体从零搭建。待引入 pgx、sqlc、Goose、validator 后补齐数据访问层。产品范围与数据模型见 `docs/implementation-plan.md`。

## 分层与目录

```text
api/
├── cmd/server/              # 程序入口
├── internal/
│   ├── project/             # 业务模块，参见下方模块清单
│   │   ├── handler.go       # 数据绑定、校验、权限与响应
│   │   ├── service.go       # 业务规则与事务编排
│   │   ├── repository.go    # 数据访问
│   │   └── dto.go           # 本模块请求与响应类型
│   ├── platform/            # 数据库、配置、日志、响应、存储、任务队列
│   └── middleware/          # 鉴权、请求 ID、错误恢复
├── contracts/openapi/       # API 契约
├── migrations/              # Goose 迁移脚本
└── docs/                    # 部署与运维说明
```

业务模块清单：`identity`、`project`、`scope`、`change`、`milestone`、`delivery`、`evidence`、`acceptance`、`audit`、`integration`。

每个业务模块内部分层职责明确：

1. Handler 负责数据绑定、边界校验、权限校验和响应构造；不涉及业务规则。
2. Service 负责业务规则、事务编排和领域数据返回，不感知 HTTP。
3. Repository 负责数据访问，查询条件尽量在 SQL 中完成，禁止把全表捞到 Go 内存后用 `for` 或 `slices` 筛选。
4. 跨模块获取数据时，简单查询可直接调用对方 Repository；涉及业务规则必须调用对方 Service。

## Handler 约定

1. URL 表示资源，HTTP 方法表示动作：查询主体是谁，接口就放在该主体对应的 Handler 中，关联条件不改变归属。例如「查询项目可关联的证据」放在证据 Handler，以项目为查询条件。
2. 路由用 Chi 的 `chi.Router` 按资源挂载子路由，中间件链统一处理请求 ID、鉴权、错误恢复与结构化日志；资源归属类中间件复用 `platform` 包中的通用实现。
3. 每个 Handler 对应一个独立的 DTO 文件；每个接口方法定义自己的请求和响应类型，不与其他方法复用绑定类。
4. 类型名以方法名为基础，请求以 `Request` 结尾、响应以 `Response` 结尾，例如 `CreateDeliveryRequest`、`CreateDeliveryResponse`。
5. 接口的角色、岗位和资源范围权限统一在 Handler 层校验：通用角色校验通过 Chi 中间件封装，涉及资源归属时先通过 Service 获取资源，在 Handler 完成判断后再调用业务方法。
6. 所有外部输入在 Handler 边界用 validator 校验，错误以 400 返回；禁止把数据库错误直接暴露为 500。
7. 统一响应结构与分页参数：分页使用 `page`、`pageSize`，响应包含 `items`、`total`、`page`、`pageSize`。

## Service 约定

1. 一个方法解决一个业务用例，行数以 30 行为宜，最长不超过 40 行。
2. 方法内部避免大量互相调用自身方法，难以单元测试通常意味着职责过多，应拆分。
3. 业务规则集中在 Service，Handler 不重复实现；Service 返回领域数据、实体或分页信息，不返回 HTTP 专用结构。
4. 每个写操作必须写入审计事件：操作人、动作、对象、变更前后摘要、时间与来源 IP。审计事件只追加，不提供更新或删除接口。

## 数据持久化约定

1. 统一使用 PostgreSQL，不使用 MongoDB。通过 `pgx` 连接，SQL 与类型生成使用 `sqlc`，表结构变更使用 Goose 迁移脚本管理，禁止手工改库。
2. 表名与列名使用小写下划线，例如 `scope_baselines`、`evidence_items`、`deleted_at`；Go 结构体字段使用大驼峰，由 sqlc 完成映射。
3. 软删除统一使用 `deleted_at timestamptz NULL`，并创建 `WHERE deleted_at IS NULL` 部分索引；查询有效数据时必须配套该条件，禁止使用布尔 `deleted` 字段，保证查询条件与索引一致。
4. 时间字段统一使用 `timestamptz`，写入与比较时显式指定 `Asia/Shanghai`，禁止依赖数据库或服务器默认时区。
5. 异构扩展数据（验收条件、CI 报告、Webhook 原始事件）使用 `JSONB` 字段；核心业务字段保持关系类型与外键约束，筛选条件必须在 SQL 中完成。
6. 需求基线冻结后禁止直接修改；变更确认生成新基线版本必须在同一事务中完成，旧版本保持只读。版本化表通过 `(project_id, version)` 唯一约束防重。
7. 交付清单生成后不可覆盖，后续证据以关联方式追加，相关写入放在同一事务内保证一致性。
8. 获取数据时调用对应实体的 Service，由其 Repository 实现查询；简单跨业务查询可直接调用 Repository，涉及逻辑的调用 Service。

## 命名与注释

1. 遵循 Go 官方命名习惯：导出类型与函数使用大驼峰，包内私有使用小驼峰，包名使用小写单词。
2. 变量、方法和字段命名要让第一次读代码的人无需询问即可理解，除非名称特别长，否则不使用缩写。
3. 结构体字段与同层方法按名称字母升序排列，便于查找与 review。
4. 新建文件或方法必须添加注释：文件注释说明职责，方法注释说明解决了什么需求，内部用 `-` 逐条描述关键决策；不要逐行翻译实现。
5. 新增库依赖时注明用途；代码中的数字提取为具名常量，禁止散落魔法数字。

## 工程约束

1. 时区统一使用 `Asia/Shanghai`：当前时间、日期边界计算、格式化与定时任务都显式指定上海时区，禁止依赖服务器默认时区。
2. 配置通过 Viper 或 koanf 加载，环境变量优先；日志使用 `slog` 结构化输出，包含请求 ID、用户 ID、动作与耗时。
3. 文件存储通过 `ObjectStore` 接口抽象，一期默认本地文件系统实现，二期接入 S3；写入即计算 SHA-256，交付清单支持 Ed25519 签名。
4. 异步任务一期使用 Outbox + Worker，不引入 Redis、Kafka 等中间件；Webhook 先保存原始事件再创建任务。
5. 模型调用通过 `platform/ai` 的 `Model` 接口接入，业务层不得直接依赖具体厂商；未配置密钥时使用本地确定性实现，保证功能开箱可用且结果可复现。
6. 模型输出必须先序列化为建议草稿落库，经人工采纳或忽略后才生效；模型不得直接写入基线、需求或验收结论。
7. 外部回调接口必须在 Handler 层校验签名或令牌，密钥缺失时的放行行为要在代码注释中写明原因。
8. Worker 领取任务使用 `FOR UPDATE SKIP LOCKED` 保证多实例不重复消费，失败任务按最大重试次数退避后置为失败。
9. 除被明确要求外，不创建单元测试文件。
10. 提交前运行 `go vet ./...` 与 `go build ./...`，确保静态检查与构建通过。
