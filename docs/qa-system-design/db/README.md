# 智能问答系统 — PostgreSQL 本地部署

## 一键启动

```bash
cd docs/qa-system-design/db
docker compose up -d
```

首次启动会自动执行 `init/` 下所有 `.sql`（按文件名排序）。

## 连接信息

| 项 | 值 |
|----|-----|
| Host | `localhost` |
| Port | `5432` |
| Database | `qa_system` |
| User | `qa_app` |
| Password | `qa_app_dev` |

```bash
docker exec -it qa-system-postgres psql -U qa_app -d qa_system
```

## 验证表是否建好

```sql
\dt
SELECT table_name FROM information_schema.tables
WHERE table_schema = 'public' ORDER BY table_name;
```

## 重置数据库（清空后重新建表）

`init/` 脚本**只在数据目录为空时**执行一次。要重来：

```bash
docker compose down -v   # 删除 volume
docker compose up -d
```

## 文件说明

| 文件 | 作用 |
|------|------|
| `docker-compose.yml` | Postgres 16 容器 + 端口 + 健康检查 |
| `init/01_extensions.sql` | 扩展（`pgcrypto` / UUID）与时区 |
| `init/02_schema.sql` | 全部建表 + 索引 |
| `init/03_seed_dev.sql` | 开发种子数据（可删） |

## 后续变更 schema

不要改已执行的 `init/*.sql`（同学已有 volume 不会重跑）。新增迁移请放到单独目录，例如 `migrations/20250628_add_xxx.sql`，手动或脚本执行：

```bash
docker exec -i qa-system-postgres psql -U qa_app -d qa_system < migrations/xxx.sql
```
