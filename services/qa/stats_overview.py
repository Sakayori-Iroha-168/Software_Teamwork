"""
DLQA-131 — 9.1.2 后端：聚合问答核心指标
GET /api/admin/stats/overview

框架：FastAPI + asyncpg（PostgreSQL）
数据源：response_runs（统计事实表，不新建统计主表）+ conversations
说明：知识库数/文档数来自外部接口，失败标记 "unavailable"，绝不伪造 0。

这是参考实现，集成进团队 services/qa/ 时，请把 get_pool / require_admin
替换成团队 auth、数据库连接的真实实现。
"""
from __future__ import annotations

from datetime import datetime, timezone
from typing import Union

import asyncpg
import httpx
from fastapi import APIRouter, Depends
from pydantic import BaseModel

router = APIRouter(prefix="/api/admin/stats", tags=["admin-stats"])

UNAVAILABLE = "unavailable"


# ─────────────────────────────────────────────────────────────
# 响应模型（0 与 unavailable 必须可区分：0 是 int，unavailable 是 str）
# ─────────────────────────────────────────────────────────────
class StatsOverview(BaseModel):
    total_qa: int                          # 知识问答总次数
    today_qa: int                          # 今日次数
    avg_latency_ms: int | None             # 平均延迟；无已完成运行时为 None
    active_external_users: int             # 活跃外部用户数
    knowledge_base_count: Union[int, str]  # int 或 "unavailable"
    document_count: Union[int, str]        # int 或 "unavailable"
    generated_at: datetime                 # 统计生成时间


# ─────────────────────────────────────────────────────────────
# 核心聚合 SQL —— 只统计「有效运行」status='completed'
#   · 平均延迟排除未完成运行（finished_at IS NOT NULL）
#   · 活跃外部用户 = response_runs JOIN conversations 的 distinct external_user_id
# ─────────────────────────────────────────────────────────────
AGG_SQL = """
SELECT
  COUNT(*) FILTER (WHERE r.status = 'completed')                                                AS total_qa,
  COUNT(*) FILTER (WHERE r.status = 'completed'
                     AND r.started_at >= date_trunc('day', NOW()))                              AS today_qa,
  AVG(r.latency_ms) FILTER (WHERE r.status = 'completed' AND r.finished_at IS NOT NULL)          AS avg_latency_ms,
  COUNT(DISTINCT c.external_user_id) FILTER (WHERE r.status = 'completed')                       AS active_external_users
FROM response_runs r
JOIN conversations c ON c.id = r.conversation_id;
"""


async def fetch_kb_stats(base_url: str) -> tuple[Union[int, str], Union[int, str]]:
    """调用外部知识库统计接口。任何失败都返回 unavailable，绝不伪造 0。"""
    try:
        async with httpx.AsyncClient(base_url=base_url, timeout=5.0) as client:
            resp = await client.get("/internal/kb/stats")
            resp.raise_for_status()
            data = resp.json()
            return int(data["knowledge_base_count"]), int(data["document_count"])
    except Exception:
        return UNAVAILABLE, UNAVAILABLE


# ─────────────────────────────────────────────────────────────
# 依赖占位 —— 集成时替换为团队真实实现
# ─────────────────────────────────────────────────────────────
async def get_pool() -> asyncpg.Pool:  # pragma: no cover
    """团队应在应用启动时创建连接池并复用，这里仅占位。"""
    raise NotImplementedError("用团队的 asyncpg 连接池替换")


async def require_admin() -> None:  # pragma: no cover
    """权限校验：仅 admin / 超级管理员可访问。集成时替换为团队 auth。"""
    return None


# ─────────────────────────────────────────────────────────────
# 接口
# ─────────────────────────────────────────────────────────────
KB_STATS_BASE_URL = "http://knowledge-service"  # 外部知识库服务地址（待团队确认，依赖 8.1.2）


@router.get("/overview", response_model=StatsOverview)
async def get_overview(
    pool: asyncpg.Pool = Depends(get_pool),
    _: None = Depends(require_admin),
) -> StatsOverview:
    # 1) 从 response_runs 聚合 4 个指标（一次查询）
    async with pool.acquire() as conn:
        row = await conn.fetchrow(AGG_SQL)

    # 2) 外部接口取 知识库数 / 文档数（失败 → unavailable）
    kb_count, doc_count = await fetch_kb_stats(KB_STATS_BASE_URL)

    # 3) 平均延迟取整；无完成运行时 AVG 为 NULL → None
    avg_latency = row["avg_latency_ms"]
    avg_latency_ms = int(round(avg_latency)) if avg_latency is not None else None

    return StatsOverview(
        total_qa=row["total_qa"] or 0,
        today_qa=row["today_qa"] or 0,
        avg_latency_ms=avg_latency_ms,
        active_external_users=row["active_external_users"] or 0,
        knowledge_base_count=kb_count,
        document_count=doc_count,
        generated_at=datetime.now(timezone.utc),
    )
