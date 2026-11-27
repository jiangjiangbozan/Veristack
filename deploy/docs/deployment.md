# 私有化部署指南

Veristack 支持在自有服务器或内网环境完整部署，不依赖任何外部 SaaS。

## 一、环境要求

| 组件 | 版本 | 说明 |
| --- | --- | --- |
| Docker | 24+ | 提供容器编排 |
| Docker Compose | v2+ | 编排 API、Web 与 PostgreSQL |
| PostgreSQL | 17 | 业务与证据数据存储 |
| 反向代理 | Nginx / Traefik | 可选，用于 HTTPS 与多域名 |

最低硬件：2 核 4GB 内存、20GB 磁盘。证据附件较多时建议单独规划存储卷。

## 二、快速启动

```bash
# 1. 准备环境变量
cp .env.example .env
# 编辑 .env，至少修改 VERISTACK_AUTH_SECRET 与数据库密码

# 2. 启动全部服务
docker compose -f deploy/compose/docker-compose.yml up -d --build

# 3. 查看状态
docker compose -f deploy/compose/docker-compose.yml ps
```

启动后：

- Web 控制台：<http://localhost:5173>
- API 服务：<http://localhost:8080>
- 健康检查：<http://localhost:8080/healthz>

首次启动时 API 会自动执行数据库迁移（Goose），无需手动初始化。

## 三、配置项

所有配置通过环境变量注入，前缀为 `VERISTACK_`，完整清单见仓库根目录 `.env.example`。

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `VERISTACK_DATABASE_DSN` | 无（必填） | PostgreSQL 连接串 |
| `VERISTACK_AUTH_SECRET` | 无（必填） | 会话令牌签名密钥，生产环境必须修改 |
| `VERISTACK_AUTH_EXPIRESIN` | `24h` | 令牌有效期 |
| `VERISTACK_SERVER_ADDR` | `:8080` | 监听地址 |
| `VERISTACK_SERVER_PUBLICBASEURL` | `http://localhost:5173` | 前端公网地址，用于密码重置与通知邮件里的链接 |
| `VERISTACK_SMTP_HOST` | 空 | SMTP 主机；留空时账户与通知邮件只输出到日志 |
| `VERISTACK_SMTP_PORT` | `587` | SMTP 端口 |
| `VERISTACK_SMTP_USERNAME` | 空 | SMTP 用户名 |
| `VERISTACK_SMTP_PASSWORD` | 空 | SMTP 密码 |
| `VERISTACK_SMTP_FROM` | 空 | 发件地址 |
| `VERISTACK_SMTP_FROMNAME` | `Veristack` | 发件人显示名称 |
| `VERISTACK_LOGLEVEL` | `info` | 日志级别 |
| `VERISTACK_STORAGE_DRIVER` | `local` | 对象存储驱动，一期支持 `local` |
| `VERISTACK_STORAGE_LOCALROOT` | `./data` | 证据附件与 Pack 存放目录 |
| `VERISTACK_SIGNING_PRIVATEKEYPATH` | 空 | Ed25519 私钥路径，配置后导出的 Pack 带签名 |
| `VERISTACK_WEBHOOK_GITHUBSECRET` | 空 | GitHub 事件签名密钥 |
| `VERISTACK_WEBHOOK_GITLABSECRET` | 空 | GitLab 事件令牌 |
| `VERISTACK_WEBHOOK_GITEESECRET` | 空 | Gitee 事件令牌 |
| `VERISTACK_AI_APIKEY` | 空 | 模型接口密钥，留空则使用本地确定性规则 |
| `VERISTACK_AI_BASEURL` | `https://api.openai.com/v1` | 模型接口地址 |
| `VERISTACK_AI_MODEL` | 见代码默认值 | 模型标识 |

### 关于令牌密钥

`VERISTACK_AUTH_SECRET` 未配置时服务拒绝启动。生产环境请使用足够随机的值：

```bash
openssl rand -base64 48
```

### 关于交付清单签名

配置签名私钥后，导出的 Evidence Pack 会附带 Ed25519 签名，客户可离线核验交付内容：

```bash
# 生成密钥对（私钥用于服务端，公钥分发给客户）
docker compose -f deploy/compose/docker-compose.yml exec api \
  /app/veristack keygen /data/signing.key

# 在 .env 中配置
# VERISTACK_SIGNING_PRIVATEKEYPATH=/data/signing.key
```

## 四、启用 HTTPS

生产环境应在服务前放置反向代理终止 TLS。Nginx 示例：

```nginx
server {
    listen 443 ssl http2;
    server_name veristack.example.com;

    ssl_certificate     /etc/letsencrypt/live/veristack.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/veristack.example.com/privkey.pem;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        proxy_pass http://127.0.0.1:5173;
        proxy_set_header Host $host;
    }
}
```

## 五、第三方平台接入

在 GitHub、GitLab 或 Gitee 中为每个项目配置回调地址：

```
POST https://veristack.example.com/api/v1/webhooks/{provider}/projects/{projectId}
```

其中 `provider` 取值 `github`、`gitlab`、`gitee`，`projectId` 为项目 ID（在项目概览页可见）。

回调地址的鉴权方式：

| 平台 | 校验方式 | 配置项 |
| --- | --- | --- |
| GitHub | `X-Hub-Signature-256` HMAC-SHA256 签名 | `VERISTACK_WEBHOOK_GITHUBSECRET` |
| GitLab | `X-Gitlab-Token` 令牌 | `VERISTACK_WEBHOOK_GITLABSECRET` |
| Gitee | `X-Gitee-Token` 令牌 | `VERISTACK_WEBHOOK_GITEESECRET` |

密钥未配置时签名校验放行，便于内网联调；正式环境务必配置。

## 六、备份与恢复

需要备份的内容：

| 数据 | 位置 | 说明 |
| --- | --- | --- |
| 业务与证据数据 | PostgreSQL 数据卷 | 项目、基线、交付、验收与审计 |
| 证据附件 | `storage_data` 卷 | 截图、测试报告等原始文件 |
| Evidence Pack | `storage_data` 卷的 `packs` 子目录 | 导出后可独立核验的交付快照 |
| 签名私钥 | 自行保管 | 丢失后无法再生成可验证的新签名 |

备份示例：

```bash
# 数据库
docker exec veristack-postgres pg_dump -U veristack veristack > veristack-$(date +%F).sql

# 证据附件
docker run --rm -v veristack_storage_data:/data -v $(pwd):/backup alpine \
  tar czf /backup/storage-$(date +%F).tar.gz -C /data .
```

## 七、升级

```bash
git pull
docker compose -f deploy/compose/docker-compose.yml up -d --build
```

数据库迁移在 API 启动时自动执行，迁移脚本只增不改，可在升级前先备份数据库。

## 八、常见问题

**API 启动即退出，日志显示数据库连接失败**
确认 PostgreSQL 容器已健康启动，且 `VERISTACK_DATABASE_DSN` 中的主机名与密码正确。容器网络内应使用服务名 `postgres` 而非 `localhost`。

**前端能打开但接口全部 404**
前端通过 `/api` 相对路径访问后端。本地开发由 Vite 代理转发（见 `web/vite.config.ts`），容器部署由 Nginx 反向代理转发。检查代理配置是否指向了正确的后端地址。

**Webhook 事件已收到但证据没有生成**
事件会先落库再由后台 Worker 异步处理，默认每 5 秒轮询一次。若持续没有生成，检查 API 日志中是否有 `处理异步任务失败` 的记录，并确认提交信息里包含 `REQ-xxx` 形式的需求编号。

**导出的 Pack 提示未附带签名**
未配置 `VERISTACK_SIGNING_PRIVATEKEYPATH`。参照上文第三节生成密钥并重启服务。
