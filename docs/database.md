# 本地数据库迁移与演示数据

本文命令均在本机 Windows PowerShell 7 下执行，数据库为本机 Docker 中的 PostgreSQL。迁移只在本机库执行，不要用云主机替代验证。

## 1. 确认数据库已启动

```powershell
docker ps --filter "name=veristack-postgres"
```

本机默认连接串：

```text
postgres://veristack:veristack@localhost:5432/veristack?sslmode=disable
```

## 2. 执行迁移

```powershell
cd D:\Veristack\api
$env:VERISTACK_DATABASE_DSN = 'postgres://veristack:veristack@localhost:5432/veristack?sslmode=disable'
go run ./cmd/migrate
```

输出应为：

```text
数据库迁移完成，当前版本 10
```

`010_schema_comments.sql` 已包含 25 张业务表、229 个字段的中文注释。迁移只新增注释，不修改表结构，也不覆盖已有数据。

## 3. 验证表和字段都有注释

在数据库容器中执行：

```sql
SELECT count(*) AS tables,
       count(*) FILTER (WHERE obj_description(c.oid, 'pg_class') IS NULL) AS missing
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind = 'r';

SELECT count(*) AS columns,
       count(*) FILTER (
         WHERE col_description(
           format('%I.%I', table_schema, table_name)::regclass::oid,
           ordinal_position
         ) IS NULL
       ) AS missing
FROM information_schema.columns
WHERE table_schema = 'public';
```

预期 `missing` 均为 `0`。

## 4. 写入演示数据

```powershell
cd D:\Veristack\api
$env:VERISTACK_DATABASE_DSN = 'postgres://veristack:veristack@localhost:5432/veristack?sslmode=disable'
go run ./cmd/seed -count 1000
```

演示数据按外键顺序在单个事务内写入，覆盖组织、用户、项目、需求、基线、里程碑、交付、证据、审批、异议、审计、通知、Outbox、Webhook、CI 报告、AI 建议、客户验收等表。重复执行会被 `demo.%@veristack.local` 用户标记拦截，不会重复插入。

也可以在仓库根目录使用 `make seed-demo`，效果相同。

## 5. 常见问题

### goose 版本号冲突

如果此前手工写入过 `goose_db_version`，序列可能没有推进，迁移会报 `duplicate key`。执行一次：

```sql
SELECT setval(
  pg_get_serial_sequence('goose_db_version', 'id'),
  COALESCE((SELECT MAX(id) FROM goose_db_version), 0) + 1,
  false
);
```

然后重新执行迁移。
