# 单元测试编写规范（项目官方标准）

> 本文档是 Veristack 项目的**权威测试规范**，在 Google 软件工程实践与 Microsoft Learn 官方单元测试准则的基础上，补充本项目的落地细则。
>
> 规范演进方式：Google 与 Microsoft 的原始出处见文末引用，本文档第三、四章为针对 Veristack 的补充与扩展。

## 一、一句话总纲

> **只要你新增 / 修改业务逻辑代码，原则上就附带单元测试；纯 UI、胶水代码、一次性脚本可以不写。**

配套三条硬性要求：

1. 测试与业务代码**在同一个提交**里，不允许拆开提交
2. 单元测试**必须隔离**：不依赖数据库、Redis、网络、真实文件与外部服务
3. 一个测试只验证**一个行为**，遵循 AAA（Arrange-Act-Assert）

## 二、什么时候必须写单元测试

### 2.1 必须写（评审可直接打回）

1. **新增包含业务规则、计算、判断逻辑的函数 / 方法**
   校验、金额计算、状态流转、权限判断、数据转换、摘要计算、签名校验
2. **修改已有逻辑**
   修复 bug、调整分支条件、变更业务规则
   **修复 bug 时优先先写一个能复现该 bug 的单元测试，再改代码**
3. **重构核心逻辑**
   即使对外行为不变，重构前没有测试的，要先补上测试，防止重构改错
4. **边界复杂、容易踩坑的逻辑**
   枚举、状态机、分页、排序、特殊输入、异常分支
5. **底层公共工具包、通用 util、核心包**
   会被多处复用，改动影响面大

### 2.2 可以不写（明确豁免）

1. **纯胶水代码 / 简单转发**
   只做参数绑定后直接调用 Service，本身没有任何判断或计算
2. **纯 UI 渲染层**
   页面展示、样式绑定、交互动效 —— 交给集成测试与端到端测试
3. **一次性临时脚本、原型验证代码、很快废弃的 Demo**
4. **直接调用稳定第三方框架 / 标准库**
   不要去测框架本身，那是框架作者的职责
5. **HTTP 路由注册、数据库迁移脚本**
   迁移由集成测试覆盖，不写单元测试

### 2.3 豁免不是默认，必须显式说明

**允许豁免的代码必须满足两条**：

- 代码注释或 PR 描述里**写明豁免理由**（属于 2.2 哪一类）
- 评审人认可该理由

> 没有说明的「忘了写」一律视为不合格。

## 三、Veristack 项目落地细则

本章是针对本项目的补充与扩展，优先于通用惯例。

### 3.1 本项目「业务逻辑」的判定清单

以下位置**必须**有单元测试：

| 位置 | 典型逻辑 |
| --- | --- |
| `internal/platform/pack` | 签名载荷构造、规范化序列化、路径安全校验、摘要计算、核验结论判定 |
| `internal/platform/auth` | 令牌签发与解析、过期判定、密码哈希与校验 |
| `internal/platform/ai` | 需求提取、变更影响分析、摘要生成的产出结构 |
| `internal/platform/testreport` | JUnit XML 解析、用例状态判定、统计汇总 |
| `internal/platform/datetime` | 时区转换、边界日期计算 |
| `internal/platform/config` | 配置校验、Fail Closed 判定、默认值拒绝 |
| `internal/platform/dbutil` | 可空类型转换、JSON 序列化 |
| `internal/platform/storage` | 对象键路径校验、目录穿越防护 |
| 各业务模块 `service.go` | 状态流转判断、权限判定、事务边界、幂等判定 |
| 各业务模块 `handler.go` | **仅当**包含业务判断时（如权限校验、参数组合判断） |

以下位置**默认豁免**：

| 位置 | 理由 |
| --- | --- |
| 各模块 `handler.go` 的参数绑定与响应封装 | 纯胶水，交给接口集成测试 |
| `internal/platform/views` 的字段转换 | 纯映射，字段增减由编译期保证 |
| `internal/platform/db` 的 sqlc 生成代码 | 生成物，不手写测试 |
| `web/src/features/**` 的页面组件 | 纯 UI，交给 Playwright E2E |
| `web/src/layouts`、`web/src/app` | 装配层 |
| `web/src/components/ui` | 样式与结构，交给 E2E 与视觉走查 |
| `web/src/services/**` 的请求封装 | 薄封装，交给前端集成测试 |

### 3.2 前端例外：必须写单元测试的位置

前端虽然以 UI 为主，但以下属于**业务逻辑**，必须写单元测试：

| 位置 | 典型逻辑 |
| --- | --- |
| `shared/lib/**` | 类名合并、日期格式化、状态映射 |
| `shared/types/**` 相关的转换函数 | 枚举映射、状态到色调的映射 |
| `features/**/service.ts` 中的纯函数 | 参数构造、响应映射、状态判定 |
| 任何抽离出的自定义 Hook 中的**判定逻辑** | 权限判断、表单校验、状态派生 |
| zod schema | 校验规则本身的边界行为 |

> 判定标准与后端一致：**有没有自己写的判断、计算、转换**。有，就写。

### 3.3 隔离要求（Go）

单元测试**禁止**依赖以下任何一项：

- 数据库连接（真实 PostgreSQL、容器、嵌入式实例）
- 网络访问
- 真实文件系统写入（需要写文件时用 `t.TempDir()`）
- 时间.Now 的实际取值（需要固定时间时注入时间源）
- 随机数（需要确定性时注入种子）

> 依赖数据库、网络、真实文件系统的测试属于**集成测试**，应放在 `integration` 标签或独立包中，不得混入单元测试。

### 3.4 隔离要求（前端）

- 组件渲染测试必须用 Testing Library + jsdom，不依赖真实浏览器
- 网络请求必须 mock，不打真实接口
- 定时器必须用 fake timers，避免不稳定
- 测试之间不得共享可变状态

### 3.5 命名规范

**Go**（表格驱动为主）：

```go
func TestVerifySignatureRejectsTamperedManifest(t *testing.T) {
    // Arrange
    // Act
    // Assert
}
```

- 测试函数名：`Test<被测函数>_<预期行为>`
- 表格用例字段：`name`（用例描述）、输入、期望结果

**前端**（Vitest）：

```ts
describe('toneForStatus', () => {
  it('应当把 approved 映射为成功色调', () => {
    // Arrange
    // Act
    // Assert
  })
})
```

- 中文描述**期望行为**，不描述实现

### 3.6 测试与实现的同提交要求

- 新增业务逻辑 → 同一个提交里带上测试
- 修复 bug → 同一个提交里带上复现测试
- 重构 → 同一个提交里带上保持行为不变的测试

> CI 会检查「业务文件改动但对应测试未改动」的提交。

## 四、覆盖率要求

不设死板的百分比数字，改为**按风险分级**：

| 等级 | 范围 | 要求 |
| --- | --- | --- |
| **关键路径** | 3.1 / 3.2 表中标为「必须」的位置 | 必须有测试，且覆盖正常值、边界值、非法输入、异常分支四类 |
| 一般逻辑 | 其他含判断的业务函数 | 至少覆盖正常值与主要异常分支 |
| 豁免位置 | 3.1 / 3.2 表中标为「默认豁免」的位置 | 不要求 |

**关键路径的四类用例缺一不可**：

```go
cases := []struct {
    name    string
    input   ...
    want    ...
    wantErr bool
}{
    {name: "正常值", ...},
    {name: "边界值", ...},
    {name: "非法输入", ...},
    {name: "异常场景", ...},
}
```

## 五、测试层次与分工

| 层次 | 范围 | 运行方式 | 归属 |
| --- | --- | --- | --- |
| **单元测试** | 纯函数、业务判断、工具 | `go test ./...` / `npm run test` | 与业务代码同目录 |
| **集成测试** | 数据库事务、租户隔离、幂等、过期、审计持久化 | 需要真实 PostgreSQL | `api/internal/integration_test/` 或带 `integration` 标签 |
| **端到端测试** | 完整业务链路 | Playwright | `web/e2e/` |

> 三个层次互补，单元测试不能替代集成测试与端到端测试。

## 六、CI 门禁

CI 必须执行：

1. `go vet ./...` 与 `gofmt -l` 检查
2. `go test ./...`（单元测试）
3. `npm run lint`
4. `npm run test`（前端单元测试）
5. `npm run build`（含 TypeScript 类型检查）
6. `sqlc generate` 后校验无差异（防止手改生成代码）

任一失败即阻断合并。

## 七、例外与豁免流程

确实无法写单元测试时：

1. 在 PR 描述中写明**属于 2.2 的哪一类**以及**为什么本项目不适用**
2. 标注对应的集成测试或端到端测试由哪一层覆盖
3. 由评审人确认

**不接受「时间不够」作为豁免理由。**

## 参考出处

- Google Engineering Practices / Building Secure and Reliable Systems — [When to write unit tests](https://google.github.io/building-secure-and-reliable-systems/ch13.html)
- Microsoft Learn — [单元测试基础](https://learn.microsoft.com/zh-cn/visualstudio/test/unit-test-basics)
- Microsoft Learn — [ASP.NET Core 应用的单元测试](https://learn.microsoft.com/zh-cn/dotnet/architecture/modern-web-apps-azure/test-asp-net-core-mvc-apps)
