# **Veristack 商业化审查报告**

我检查了你目前 GitHub `main` 分支的 README、Go 后端、React 前端、OpenAPI 契约、Evidence Pack 实现、部署配置和部分核心业务代码。

先给结论：

**当前产品成熟度**

Alpha 阶段

# **已经有真实业务实现，但还不能正式商用。**

不是只有架构图和页面原型。核心业务模块已经有相当一部分代码，甚至已经有独立的 Evidence Pack 核验 CLI。

但目前最重要的短板不是继续增加功能，而是：

- 证据完整性与身份可信度。
- 数据安全与多租户隔离。
- 业务状态和审计的一致性。
- 自动化测试及生产环境可靠性。
- 真实客户使用验证。

我还发现一个需要纠正的地方：`docs/implementation-plan.md` 第 2 节仍然写着 API 没有源码、Web 还是默认模板，但实际仓库已经存在完整的 Go 服务入口、业务模块、React 路由和多个业务页面。它后面的实施路线又把阶段 2～4 标成已完成。这说明项目文档没有跟上代码，需要及时整理。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub+3



下面按实际实现、风险和商用交付要求展开。

# **一、目前已经实现到什么程度？**

## **1. 后端：已经有比较完整的业务骨架**

已有实际代码

你当前使用 Go 1.27、Chi、PostgreSQL、pgx、sqlc、Goose。

从服务入口 `api/cmd/server/main.go` 可以看到，已经注册了：

| 模块          | 当前实现内容               |
| ----------- | -------------------- |
| Identity    | 用户注册、登录、JWT          |
| Project     | 项目管理、成员管理、项目概览       |
| Scope       | 需求范围基线               |
| Change      | 需求变更申请               |
| Milestone   | 里程碑管理                |
| Delivery    | 交付版本、交付清单            |
| Evidence    | 证据上传、关联、查询           |
| Acceptance  | 客户逐条验收、异议和回复         |
| Integration | Webhook、CI 报告、AI 建议  |
| Platform    | 数据库、认证、存储、日志、签名、错误处理 |

这不是简单的 CRUD 拼接，至少已经建立了 Handler、Service、数据访问层的业务分层。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub+3



## **2. 前端：已经不再是默认模板**

已有实际业务页面

我检查了几个主要页面：

**项目列表**

[Logger Dashboard ✨ by Mahdi ✤ for Piqo Studio on Dribbble](https://images.openai.com/static-rsc-4/ln7xqV4LhhWEXZ5MoJ0Pl-7jkcv3wsGfQR-tyiWgG6kCidf_QrNtX3WOqxvx0k6nWMmYHUtfTVnvA_pn2ObzRFTNZsKJrr9V0PMN4DMaeeZbIPswhu8L3SaBDTUeSTKdGGHm9SYQ-X6DD-MuUUgCtQlc0CYwNjQnFL4ol315KWE?purpose=inline)

[dribbble.com](https://dribbble.com/shots/20571535-Logger-Dashboard?utm_source=chatgpt.com)

对应 `project-list-page.tsx`，已经接入 TanStack Query，包含项目查询和新建操作。

**项目详情**

[Meridi — Guided Project Operating System for Delivery Teams](https://images.openai.com/static-rsc-4/fPTa2CLaWSOo6xChrVhCS1QzNS8Sp7Zf2cpAKGU1RxJyE32Wu8MTfdTghWsf9MC-ss312RjdO-6XMAkIZdoy5Wm5IiKbqu1ehRotRO6FCcroifgHcvNiEegsyLvsAvJT1enXi8fZ_9IG7LAGSwPTf0owaA6mUgMmWte0RH4Qmb0?purpose=inline)

[meridi.ai](https://meridi.ai/?utm_source=chatgpt.com)

已经有概览、基线、变更、里程碑、交付、自动化集成等业务区域。

**交付详情**

[How to Run UAT in Jira: Templates & Sign-Off | BesTest](https://images.openai.com/static-rsc-4/-_zNT6yZapPUwg-yplhPEDUJ_szvGvtGSDFT9EZgj1uEQCUH3yOGCBvFuV1Nyx7wI90e84AOe42sxWgnMI3oZD27sGZ7KgUz30i5YwqjyvGt5hAPCpnywuoK5jHZtRdICSgLHBOaeQWVxQnWuBh1NHG-OoYaT7doP7xnW0BwhCA?purpose=inline)

[getbestest.com](https://getbestest.com/blog/user-acceptance-testing-in-jira/?utm_source=chatgpt.com)

已经组织清单、证据时间线、客户验收三个主要视图。

**客户验收门户**

[Free Ecommerce Website Statement of Work - Better Proposals](https://images.openai.com/static-rsc-4/pc1wW0GEreWeiuL8DRUQ2nqWDDyIoXkdliA3sCk2vnskj3K5zUehU6-USuk6R7whJ4dSiJsgnlcu-obaGGI5zQXkUiztwa1HA6h9_9K9ZCgZfTHpJ-UQCPSTsxgtA4k9R1eABB010bTSIy2d3WgfEkbx1UT-_gWXQAsKXIGrR9w?purpose=inline)

[betterproposals.io](https://betterproposals.io/statement-of-work/e-commerce-website?utm_source=chatgpt.com)

客户可以通过分享链接访问交付、逐条提交验收结果及异议。

这里的图片只是界面类型示意，不是我实际打开的仓库截图。实际源码对应 `project-list-page.tsx`、`project-detail-page.tsx`、`delivery-detail-page.tsx` 和 `acceptance-portal-page.tsx`。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub+3



我认为目前前端已经具备业务操作路径，但还不能仅凭组件和页面存在，就认定所有交互、异常状态和移动端表现都已经通过验收。

## **3. 自动化集成：已经有初步实现**

这一块比我预期的完整。

| 集成能力            | 实现情况               |
| --------------- | ------------------ |
| GitHub Webhook  | 已有接收和校验代码          |
| GitLab Webhook  | 已有接收和校验代码          |
| Gitee Webhook   | 已有接收和校验代码          |
| Git Commit 自动归档 | 已有 Worker 处理逻辑     |
| 需求关联            | 支持从 `REQ-xxx` 匹配需求 |
| JUnit XML       | 已有解析和报告保存逻辑        |
| AI 需求提取         | 已有模型适配与建议记录        |
| AI 变更分析         | 已有建议生成             |
| AI 交付摘要         | 已有建议生成             |

例如，Worker 会处理 push 事件，把 Commit 信息转化为证据记录；集成服务会解析 JUnit XML，并保存测试报告统计。AI 结果也设计成草稿，不会自动改变正式需求基线。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub+2



不过现在的 Git 自动关联比较简单，主要依靠 Commit Message 中的需求编号，尚不能将它描述成完整的代码变更追踪系统。

## **4. Evidence Pack：已经实现了一个很重要的独立能力**

有实际实现

你已经定义了独立的 Evidence Pack v1 格式，并实现：

- `manifest.json`：交付需求和验收清单。
- `pack.json`：交付元数据、附件摘要和签名信息。
- `files/`：交付证据附件。
- Ed25519：可选签名。
- SHA-256：内容完整性校验。
- `veristack verify`：独立核验。
- `veristack keygen`：生成签名密钥。

这一点很有价值，因为它使证据包不必完全依赖 Veristack 在线服务器才能阅读和检查。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub+2



但我在检查实现时发现了几个比较严重的安全和可信度问题，后面专门讲。

## **5. 部署基础**

目前有：

- Docker Compose。
- API 多阶段 Dockerfile。
- Web Dockerfile。
- PostgreSQL。
- 本地文件存储。
- 环境变量配置。
- HTTPS 反向代理示例。
- 私有化部署文档。

这些构成了部署原型，但还没有达到企业生产交付的标准。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub+2



# **二、距离商用最近的五个关键问题**

这里是这次审查中最重要的部分。

我没有只检查功能列表，而是直接看了几个关键代码路径，发现有些问题必须在发布之前修复。

## **P0：Evidence Pack 的签名目前不能证明整份证据包完整**

必须修复

这是我认为最重要的一个问题。

当前 `pack.json` 里面包含：

```
Pack
├── SchemaVersion
├── GeneratedAt
├── Delivery
├── ManifestSHA
├── Files[]
└── Signature
```

其中 `Files[]` 存放每份证据的路径、SHA-256、大小和标题。

但是查看 `verifySignature()` 可以发现：

```
ed25519.Verify(
    ed25519.PublicKey(publicKey),
    []byte(meta.ManifestSHA),
    signature,
)
```

目前签名只覆盖 `manifestSha256`，并没有覆盖整个 `pack.json` 中的 `Files` 和 `Delivery` 等信息。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub+1



这会产生一个重要问题。

假设原始证据包：

```
pack.json
├── manifestSha256: abc123
├── files:
│   ├── screenshot-1.png
│   ├── test-report.xml
│   └── commit-info.json
└── signature
```

有人修改 `pack.json`：

```
pack.json
├── manifestSha256: abc123
├── files:
│   └── screenshot-1.png
└── signature
```

由于签名没有覆盖附件清单，核验程序仍可能通过。

也就是说，证据附件可以从清单中被删除，而清单签名不一定失败。

### **应该怎么改？**

升级成 Evidence Pack v2。

我建议对一个完整的、规范化的签名载荷签名：

```
{
  "schemaVersion": "2",
  "delivery": {},
  "manifest": {
    "path": "manifest.json",
    "sha256": "..."
  },
  "evidence": [
    {
      "id": "...",
      "path": "files/...",
      "sha256": "...",
      "size": 20480,
      "title": "..."
    }
  ],
  "generatedAt": "...",
  "issuer": {
    "organizationId": "...",
    "keyId": "..."
  }
}
```

然后：

```
签名载荷
    |
    v
规范化序列化
    |
    v
SHA-256
    |
    v
Ed25519 Sign
    |
    v
signature
```

另外，不能只依靠 Pack 自己携带的公钥验证身份。

公钥也应该通过独立可信渠道进行绑定，例如客户预先固定的组织公钥、组织密钥注册信息或可信证书。

这样才真正形成：

- 内容没有被修改。
- 附件没有被偷偷删除。
- 交付信息没有被偷偷更换。
- 签名主体能够被识别。

## **P0：客户分享链接存在明文泄露**

必须立即修复

我在 `api/internal/acceptance/handler.go` 里面看到了：

```
h.recordAudit(
    r,
    principal,
    "acceptance.share",
    deliveryID,
    nil,
    map[string]string{"token": token},
)
```

而 `audit.Writer.Record()` 会将 `after` 对象 JSON 序列化后写入数据库。

这意味着：

**虽然&#x20;**`share_tokens`**&#x20;表只保存了 Token 哈希，但是&#x20;**`audit_events.after`**&#x20;可能保存了完整的明文分享 Token。**

这样就破坏了只存 Token 哈希的安全设计。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub+1



修复建议：

```
h.recordAudit(
    r,
    principal,
    "acceptance.share",
    deliveryID,
    nil,
    map[string]any{
        "tokenIssued": true,
        "expiresAt":   expiresAt,
    },
)
```

审计日志只记录 Token 已签发，不记录 Token 本身。

对于当前已经签发的分享链接，我建议：

1. 先修复代码。
2. 撤销已有分享链接。
3. 清理或安全处理已泄露的审计记录。
4. 重新签发新的客户链接。

这项不要留到正式发布之后。

## **P0：客户填写邮箱，不等于验证了客户身份**

当前客户验收门户要求填写邮箱。

但后端的 `EnsureCustomer()` 是：

```
user, err := queries.GetUserByEmail(ctx, email)

if err == nil {
    return user, nil
}
```

如果邮箱不存在，就直接创建一个客户账号，并使用随机密码保存。

也就是说，客户只需持有分享链接，再输入某个邮箱，就可能以这个邮箱所对应的用户身份提交验收结果。

当前没有验证邮箱归属。甚至已有邮箱属于其他组织时，代码也没有在这个分支验证组织归属。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub+1



这对普通项目协作也许能凑合，但你的产品定位是交付证据，身份归因不能这么随意。

建议改为：

**客户验收身份验证流程**

1. 开发人员创建验收邀请，绑定客户邮箱。
2. 系统通过邮件发送一次性验证链接或验证码。
3. 客户验证邮箱后获得对应交付的验收权限。
4. 提交验收时，记录已验证身份、交付版本、验收清单摘要和操作时间。
5. 验收结论形成不可覆盖的事件记录。

第一版可以使用邮箱 OTP，不需要急着接入复杂的电子签章平台。

但产品文案要区分：

- 已验证邮箱的客户确认。
- 通过一次性分享链接提交的匿名确认。
- 法律意义上的电子签名。

不能把这三件事混为一谈。

## **P0：审计日志的写入失败被忽略**

例如刚才看到的 `recordAudit()`：

```
_ = writer.Record(
    r.Context(),
    audit.FromRequest(r, principal.ID),
    action,
    "delivery",
    deliveryID,
    before,
    after,
)
```

审计记录发生错误，业务操作仍然返回成功。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub+1



对于普通管理后台，这可能只是日志缺失。

但对 Veristack 来说，审计记录本身就是产品价值的一部分。

建议：

- 关键业务操作和审计事件在同一个数据库事务里写入。
- 业务事件使用 Outbox 保证后续可靠处理。
- 审计事件只允许追加。
- 审计写失败时，关键状态变更必须回滚或明确进入待补偿状态。
- 增加审计事件连续性检查。

不然你会遇到一个很荒唐的情况：

客户确认过什么，系统说不清楚；开发者修改过什么，系统也说不清楚，但界面上的状态已经更新。

这就失去了产品最核心的可信性。

## **P0：Evidence Pack 存在文件路径安全问题**

在 `verify.go` 中：

```
content, err := os.ReadFile(
    filepath.Join(
        root,
        filepath.FromSlash(file.Path),
    ),
)
```

`file.Path` 来自外部导入的 Pack 元数据。当前路径没有显式约束在 Pack 根目录之内。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub



恶意构造的路径，例如：

```
../../some-file
```

可能让核验程序读取 Pack 目录以外的本地文件。

修复时至少做到：

- 仅接受规范化后的相对路径。
- 禁止 `..` 路径段和绝对路径。
- 检查解析后的最终路径确实位于 Pack 根目录之内。
- 防止符号链接逃逸。
- 限制单个附件与整个 Pack 的大小。
- 改用流式哈希，避免超大文件直接占满内存。
- 检查重复附件路径和无效 Schema Version。

这是一个独立 CLI 工具，客户可能会拿到来自不可信渠道的 Pack，必须按处理不可信输入的标准设计。

# **三、其他必须修复的工程问题**

上面五项属于核心信任问题。下面这些也应进入正式发布前的缺陷清单。

| 优先级 | 问题                             | 建议                   |
| --- | ------------------------------ | -------------------- |
| P0  | 分享 Token 泄露到审计数据               | 移除明文，撤销旧链接           |
| P0  | Pack 签名范围不完整                   | 升级 v2 签名载荷           |
| P0  | Pack 文件路径校验不足                  | 路径约束和资源限制            |
| P0  | 客户邮箱身份未经验证                     | 邮箱验证或邀请认证            |
| P0  | 审计写失败被忽略                       | 事务化审计                |
| P1  | 导出 Pack 会删除旧目录重建               | 改为不可覆盖的版本化导出         |
| P1  | Webhook 重试可能重复记录 Commit        | 增加事件幂等键              |
| P1  | 签名密钥读取失败时静默降级                  | 配置了签名就必须启动失败         |
| P1  | 默认认证密钥过弱                       | 拒绝已知默认值，强制高熵密钥       |
| P1  | S3 配置被接受，但主程序实际始终使用 LocalStore | 实现 S3 工厂，或暂时拒绝 S3 配置 |
| P1  | 审计事件和业务变更非原子操作                 | 事务化                  |
| P1  | 健康检查只有存活检查                     | 增加数据库就绪检查            |
| P2  | 缺少邮件通知                         | 增加变更、验收、异议通知         |
| P2  | 缺少完整升级与恢复流程                    | 增加备份恢复演练             |

其中有几个具体代码依据值得特别说明。

**第一，Pack 导出存在覆盖历史的问题。**

`ExportPack()` 每次导出都会执行：

```
os.RemoveAll(root)
```

然后使用当前数据库里的证据重新生成同一个交付目录。这与不可覆盖的历史证据包设计存在冲突。应该为每次导出分配独立的 Export ID，先写入临时目录，完整校验后原子发布，绝不覆盖历史导出。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub



**第二，Webhook 需要幂等。**

当前 Worker 会将 push 中的 Commit 转成新的证据。代码路径没有体现基于第三方事件 ID 或 Commit SHA 的幂等保证；需要确认数据库约束，并补充重复投递和失败重试测试。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub



**第三，密钥配置需要 Fail Closed。**

目前 `Config.Validate()` 只要求 JWT 密钥非空，而 Compose 提供了 `change-me-in-production` 作为默认值。如果用户没有修改环境变量，系统仍可能使用已知密钥启动。另外，签名私钥读取失败会记录错误并继续以未签名模式运行。生产模式应拒绝这两种情况。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub+2



这些不是未来的高级功能，而是正式交付之前必须处理的基本工程问题。

# **四、我认为正式商用之前，必须交付的产品形态**

不要把商用定义成“功能全部开发完”。

对于 Veristack，应该定义为一条真实客户能够完整使用的业务链路。

我建议把产品的第一个商业版本限定为：

**小型软件定制开发团队的项目范围管理、交付证明和客户验收系统。**

暂时不需要让 Agent 成为产品主角。

## **1. 一个完整的商用工作流**

## **Veristack 商业版核心业务流程**

1. **创建项目**

   开发团队创建项目，添加客户和项目成员，设置开始时间、交付周期和项目范围。
2. **确认初始需求**

   将需求拆成可执行条目，每项都有明确的验收条件。

   冻结 Scope Baseline v1，生成版本摘要。
3. **处理范围变更**

   客户提出新增功能或修改需求。

   开发人员创建 Change Request，填写原因、工作量和影响范围。

   客户确认后，产生 Baseline v2，历史版本保留。
4. **生成交付版本**

   例如 Delivery v1.0。

   关联 Git Commit、CI 测试报告、功能截图、操作录屏和交付文档。
5. **客户验收**

   客户通过验证后的邀请链接访问，不需要注册完整组织账号。

   逐项确认、提出异议，开发团队处理并回复。
6. **正式交付归档**

   冻结交付证据快照，生成 Evidence Pack v2。

   客户获取交付报告和离线验证工具，可以独立验证证据完整性。

最终交付结果

**一份可以脱离 Veristack 服务器阅读、能够核验完整性、清楚记录客户验收过程的交付档案。**

这条链路应该成为产品的核心验收标准。

## **2. 商用版本的功能边界**

我建议这样区分：

| 功能                   | 商用首版要求          |
| -------------------- | --------------- |
| 组织、用户、权限             | 必须              |
| 项目、成员管理              | 必须              |
| 需求基线版本管理             | 必须              |
| 需求变更审批               | 必须              |
| 交付版本管理               | 必须              |
| 证据上传与关联              | 必须              |
| 客户验收门户               | 必须              |
| 异议与回复闭环              | 必须              |
| 完整审计                 | 必须              |
| Evidence Pack 独立核验   | 必须              |
| Email 通知             | 必须              |
| GitHub / GitLab      | 必须至少做好一个，再扩展另一个 |
| Gitee                | 中国市场版本重点        |
| JUnit CI 接入          | 必须              |
| AI 需求提取              | 可选              |
| AI 变更分析              | 可选              |
| Coding Agent Session | 暂缓              |
| 代码 AI 生成率检测          | 不做              |
| 区块链存证                | 不做              |

注意：这不是要求你删掉已经开发的 AI 和集成功能，而是不让这些功能抢占商业首版的资源。

# **五、下一步开发路线：从 Alpha 到商业版**

我给你制定一个真正面向交付的路线。

这里不是简单的功能清单，而是每个阶段都有验收条件。

# **Veristack Release Roadmap**

**P0 · 信任与数据安全**

最高优先级

- Evidence Pack v2 完整签名。
- CLI 路径安全及恶意文件防护。
- 分享 Token 泄露修复及旧 Token 撤销。
- 客户邮箱真实性验证。
- 审计写入事务化。
- 签名密钥加载失败时拒绝启动。
- JWT 和数据库默认密钥安全校验。
- 交付包版本化，禁止覆盖历史。\
  **验收标准：** 修改、删除、替换任意签名范围内的数据，独立核验必须失败。

**P1 · 完整业务闭环**

核心产品

- 需求基线冻结和版本对比。
- 变更审批状态机。
- 交付版本和证据关联。
- 客户邮箱邀请与身份验证。
- 验收、异议、回复、复核。
- 通知和邮件发送。
- 项目和组织权限隔离。\
  **验收标准：** 一个真实软件项目可以完整完成从需求确定到最终验收的过程。

**P2 · 质量保障**

正式发布前

- Go 单元测试。
- PostgreSQL 集成测试。
- Playwright 端到端测试。
- Webhook 重复投递测试。
- 多租户越权测试。
- 文件上传和恶意 Pack 安全测试。
- 完整备份恢复测试。\
  **验收标准：** CI 自动执行测试，关键业务链路没有未解决的 P0/P1 缺陷。

**P3 · 商业部署**

产品交付

- 正式 Docker 镜像。
- AMD64 / ARM64 多架构镜像。
- HTTPS 生产部署。
- S3 兼容对象存储。
- 数据库备份、恢复、升级。
- 系统运行状态和错误告警。
- 安装和使用文档。
- Release、Changelog、版本升级说明。\
  **验收标准：** 没有开发人员现场协助，客户也能独立安装、升级、备份和恢复。

**P4 · 商业验证**

正式商业化

- 找到 3 家真实试用团队。
- 至少完成 3 个真实项目交付。
- 客户完整使用验收门户。
- 验证客户是否愿意为产品付费。
- 建立定价、售后和问题反馈机制。\
  **验收标准：** 有真实客户持续使用，并至少形成初步付费验证。

## **六、测试体系需要单独建设**

我特别提醒你：不能只依赖人工点击页面。

这个项目的核心价值是数据不能被随意修改，所以必须有针对业务不变量的自动化测试。

建议测试体系如下：

**Go Backend**

```
Unit Tests
    |
    +-- Baseline Freeze
    +-- Change Approval
    +-- Delivery Snapshot
    +-- Acceptance State Machine
    +-- Evidence Hash
    +-- Signature Verification

Integration Tests
    |
    +-- PostgreSQL Transaction
    +-- Tenant Isolation
    +-- Webhook Idempotency
    +-- Token Expiration
    +-- Audit Persistence
```

**React Frontend**

```
Type Check
    |
    ESLint
    |
    Build
    |
    Playwright E2E
        |
        +-- Project Creation
        +-- Scope Management
        +-- Change Approval
        +-- Delivery Submission
        +-- Customer Acceptance
        +-- Objection Resolution
```

特别建议增加以下测试场景：

| 测试场景             | 预期结果            |
| ---------------- | --------------- |
| 冻结后的基线直接修改       | 拒绝              |
| 同一个变更申请重复审批      | 不产生重复基线         |
| 同一个 Webhook 重复发送 | 不产生重复 Commit 证据 |
| 客户使用过期 Token     | 拒绝              |
| 客户使用被撤销 Token    | 拒绝              |
| A 组织查询 B 组织项目    | 拒绝              |
| 交付证据被修改          | Hash 校验失败       |
| 证据附件被删除          | Pack 校验失败       |
| 签名元数据被替换         | 签名校验失败          |
| 审计写入失败           | 关键业务记录不能静默成功    |

目前 `web/package.json` 暴露了 `build`、`lint` 等脚本，但没有前端 `test` 脚本；Makefile 有 `api-test`，但这只是调用 `go test ./...`，不能证明已经存在充分的测试覆盖。

[image](https://www.google.com/s2/favicons?domain=https://raw.githubusercontent.com\&sz=32)

GitHub+1



我还没有在当前环境实际执行项目构建和端到端测试，因此上面的实现判断属于静态代码审查，不是通过运行结果出具的质量认证。

# **七、正式商用还需要哪些产品能力？**

还有几个容易被程序员忽略的问题。

功能开发完成，不代表产品可以卖给别人。特别是你准备做开源项目，还要考虑产品长期维护。

我建议补齐以下东西。

### **1. 安装和部署**

[How to Deploy Dokploy – Self-Hosted PaaS for Docker Applications | Vultr Docs](https://images.openai.com/static-rsc-4/NMHZnKtFQbbEt9mVDnygX5PE-hW0ecWETmhfWGRhKsGGbkqmgwT2JLhJj2GUcXkv9OWpD9bgHIV8DW-ICd928d1Glo94C4X5IIzv-MTkXvzsQXzYr-mQ-Iwi1q0w4Roo-t7GmlCWpgEJ7UvBDLW9Itdo48VFS84X_N4wXNfHSUY?purpose=inline)

[docs.vultr.com](https://docs.vultr.com/how-to-deploy-dokploy-self-hosted-paas-for-docker-applications?utm_source=chatgpt.com)

至少提供：

- 一键 Docker Compose 部署。
- `.env.example` 和启动前配置检查。
- 生产环境 HTTPS。
- 数据库迁移升级。
- 数据备份与恢复。
- 存储空间管理。
- 管理员初始化。
- 运行状态检查。

本地存储可以保留，但生产部署需要正式完成 S3 驱动，或者暂时明确不支持 S3。不能让配置接受某个驱动，程序却仍然使用 LocalStore。

### **2. 用户和组织管理**

目前基础注册登录已经有实现，但商业版还要考虑：

- 邮箱验证。
- 忘记密码。
- 修改密码。
- 成员邀请。
- 成员禁用和移除。
- 项目级角色管理。
- 组织级权限。
- 登录限流。
- 会话失效和 Token 撤销。

特别是你服务的是软件团队，不能只保证单个用户能使用。

### **3. 客户通知**

客户不可能每天打开 Veristack 看有没有新消息。

必须有：

- 需求变更待确认。
- 新交付待验收。
- 客户提出异议。
- 开发者回复异议。
- 验收完成。

首版 SMTP 邮件就足够，不需要先做复杂的消息中心。

### **4. 产品运维与支持**

正式发布时，应该有：

- 健康检查和 Ready 检查。
- 数据库连接和磁盘空间监控。
- 日志记录及日志脱敏。
- 备份恢复说明。
- 隐私政策。
- 数据删除和保留策略。
- 安全问题反馈渠道。
- 升级说明及兼容性说明。

目前仓库已经有部署文档，不过 GitHub 页面仍显示只有一次提交、没有正式 Release。一次提交可能是 Squash 合并，不能据此评价代码质量，但正式发布体系确实还需要建立。

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

GitHub+1



# **八、我建议采用怎样的商业模式？**

这一点我想提前帮你确定，因为它会影响后续技术设计。

我不建议你一开始就投入精力做复杂的多租户云 SaaS 计费系统。

你的项目是开源优先的软件交付证据产品，我更倾向于：

## **Open Core 商业模式**

Community Edition

**免费开源、自托管**

- 项目和需求管理。
- 变更审批。
- 交付证据。
- 客户验收。
- Evidence Pack。
- 基础 Git 和 CI 集成。
- 本地 CLI 验证。

Cloud Edition

**托管云服务**

- 官方托管。
- 自动备份。
- 多项目团队协作。
- 邮件通知。
- 更完善的权限管理。
- 更丰富的集成能力。

Enterprise Edition

**企业私有化部署与支持**

- 私有化部署支持。
- OIDC / SSO。
- 外部身份与密钥管理。
- 自定义保留策略。
- 审计和安全管理。
- 企业级技术支持。

Apache-2.0 许可证适合这个方向，可以允许商业使用。但要特别注意，不能只把源码放到 GitHub 就认为商业模式成立。

建议先让 Community Edition 成为真正可运行的工具，先找几家小团队使用，再根据真实需求决定收费边界。

计费可以考虑组织订阅、活跃项目数量或企业支持服务，不建议按提交数或证据条目数收费。

# **九、接下来我建议你实际怎么开发？**

我把上面的所有工作再压缩成一个执行计划。

# **接下来的开发顺序**

#### 不要同时开十条分支，按这个顺序完成

**第一批：解决可信性缺陷**

1. 修复分享 Token 明文进入审计的问题。
2. 修复 Pack v1 签名范围问题，设计 v2。
3. 修复 Pack 路径校验。
4. 修复审计事务一致性。
5. 修复客户邮箱身份验证。
6. 修复签名密钥和默认配置问题。

**第二批：把真实业务闭环跑通**

1. 创建组织和项目。
2. 确定需求基线。
3. 客户确认需求变更。
4. 创建交付版本。
5. 上传证据。
6. 客户验收并提出异议。
7. 导出和验证 Evidence Pack。

**第三批：自动化测试**

先让完整业务链路通过 Playwright E2E，再补数据库、权限、签名和 Webhook 的专项测试。

**第四批：商业部署**

制作正式 Release，完成部署、备份、恢复、密钥配置、邮件配置等工作。

**第五批：真实客户试用**

不要先追求一千个用户。先找 3 家真实的软件定制团队，用他们实际正在进行的项目检验产品。

## **十、最后一个判断：不要让产品失去自己的特色**

我看下来，Veristack 最值得坚持的是这个核心设计：

**需求基线和变更记录不是项目管理附属功能，而是软件交付证据的起点。**

市面上很多系统擅长管理任务、记录代码、运行测试，但 Veristack 应该始终围绕一个问题：

> 当开发团队把一份软件交给客户时，双方能否准确还原当初约定了什么、发生了什么变更、实际交付了什么、客户确认了什么？

这就是你的产品价值。

我的最终判断如下：

| 维度    | 当前判断       |
| ----- | ---------- |
| 产品定位  | 已比较明确      |
| 技术架构  | 基本成形       |
| 后端业务  | 已有较多真实实现   |
| 前端业务  | 已有主要操作页面   |
| 第三方集成 | 已有初步实现     |
| 证据可信性 | 存在必须修复的问题  |
| 自动化测试 | 缺乏足够的可验证证据 |
| 生产运维  | 尚需完善       |
| 商业验证  | 尚未建立       |

**我建议你现在不要继续扩展功能，优先完成 P0 的可信性修复，再用一个真实项目跑通完整交付流程。**

当这条链路能够稳定运行、客户身份能被验证、证据包能够独立核验、系统能够可靠部署和恢复时，Veristack 才真正具备成为商业产品的基础。

你目前已经不需要再花很多时间重新讨论技术架构了。接下来的重点，是把现有代码从「看起来已经完整」推进到「经得起真实用户和真实数据检验」。