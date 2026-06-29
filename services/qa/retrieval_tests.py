"""
DLQA-188 — 4.3 后端：实现检索体验测试运行记录

目标：创建检索体验测试并查询结果快照，供管理员评估「当前 / 覆盖参数」下的检索质量。
路径（服务级草案 /api/v1；契约以 docs/services/qa/api/openapi.yaml 为准）：
  POST /api/v1/retrieval-test-runs            创建一次检索测试（body: query, knowledgeBaseIds, overrides）
  GET  /api/v1/retrieval-test-runs/{testRunId}  读取某次测试的稳定快照

边界：
  - 只调用 knowledge 检索/重排（POST /api/v1/knowledge-queries 或 MCP search_knowledge），**不调用 LLM**；
  - overrides 仅用于本次检索，**不修改活动 QA 配置**（只写 retrieval_test_* 表）；
  - 结果按 rankNo 稳定保存，快照可重复审计；
  - 仅管理员可访问。

响应：成功 {data, requestId}；错误 {error:{code,message,requestId,fields?}}。

集成进 services/qa/ 时，get_pool / require_admin / req_id / err 与其他 QA 模块复用团队实现。
"""
from __future__ import annotations

import json
import os
import time
import uuid
from typing import Any, Optional

import asyncpg
import httpx
from fastapi import APIRouter, Depends, Header, HTTPException
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field

router = APIRouter(prefix="/api/v1", tags=["RetrievalTests"])

KNOWLEDGE_BASE_URL = os.getenv("KNOWLEDGE_SERVICE_BASE_URL", "http://knowledge-service")
KNOWLEDGE_TIMEOUT = float(os.getenv("KNOWLEDGE_TIMEOUT", "15"))


# ─────────────────────────────────────────────────────────────
# 请求 / 响应模型（对齐 QA openapi 的 RetrievalTest*）
# ─────────────────────────────────────────────────────────────
class RetrievalOptions(BaseModel):
    topK: Optional[int] = None
    useRerank: Optional[bool] = None
    similarityThreshold: Optional[float] = None
    rerankThreshold: Optional[float] = None


class RetrievalTestRequest(BaseModel):
    query: str = Field(min_length=1)
    knowledgeBaseIds: list[str] = Field(default_factory=list)
    overrides: Optional[RetrievalOptions] = None


# ─────────────────────────────────────────────────────────────
# 依赖 / 工具占位 —— 集成时与其他 QA 模块复用
# ─────────────────────────────────────────────────────────────
async def get_pool() -> asyncpg.Pool:  # pragma: no cover
    raise NotImplementedError("用团队的 asyncpg 连接池替换")


async def require_admin(
    x_user_id: Optional[str] = Header(default=None, alias="X-User-Id"),
) -> str:  # pragma: no cover
    """仅管理员可访问；返回管理员 external_user_id。集成时替换为团队 auth。"""
    if not x_user_id:
        raise HTTPException(status_code=401, detail="unauthenticated")
    return x_user_id


def req_id(x_request_id: Optional[str] = Header(default=None, alias="X-Request-Id")) -> str:
    return x_request_id or str(uuid.uuid4())


def ok(data: Any, request_id: str) -> dict:
    return {"data": data, "requestId": request_id}


def err(code: str, message: str, request_id: str, fields: Optional[dict] = None) -> dict:
    e: dict = {"code": code, "message": message, "requestId": request_id}
    if fields:
        e["fields"] = fields
    return {"error": e}


# ─────────────────────────────────────────────────────────────
# 调用 knowledge 检索（只检索/重排，不碰 LLM、不改活动配置）
# 返回：按 rank 升序的命中列表（字段做容错映射，最终以 knowledge 契约为准）
# ─────────────────────────────────────────────────────────────
async def _call_knowledge(query: str, kb_ids: list[str], overrides: Optional[RetrievalOptions]) -> list[dict]:
    payload: dict[str, Any] = {"query": query, "knowledgeBaseIds": kb_ids}
    if overrides:
        payload["options"] = overrides.model_dump(exclude_none=True)
    async with httpx.AsyncClient(base_url=KNOWLEDGE_BASE_URL, timeout=KNOWLEDGE_TIMEOUT) as client:
        resp = await client.post("/api/v1/knowledge-queries", json=payload)
        resp.raise_for_status()
        body = resp.json()
    hits = body.get("results") or body.get("data", {}).get("results") or []
    return hits


def _pick(h: dict, *keys, default=None):
    for k in keys:
        if k in h and h[k] is not None:
            return h[k]
    return default


# ─────────────────────────────────────────────────────────────
# POST：创建检索测试
# ─────────────────────────────────────────────────────────────
@router.post("/retrieval-test-runs")
async def create_retrieval_test(
    body: RetrievalTestRequest,
    request_id: str = Depends(req_id),
    admin: str = Depends(require_admin),
    pool: asyncpg.Pool = Depends(get_pool),
):
    started = time.monotonic()
    try:
        hits = await _call_knowledge(body.query, body.knowledgeBaseIds, body.overrides)
    except httpx.HTTPError:
        # 检索依赖失败 → 落一条 failed 记录（可审计），并归一错误
        latency_ms = int((time.monotonic() - started) * 1000)
        async with pool.acquire() as conn:
            await conn.execute(
                """INSERT INTO retrieval_test_runs
                     (external_user_id, query, overrides, status, latency_ms, error_message, finished_at)
                   VALUES ($1,$2,$3,'failed',$4,$5, NOW())""",
                admin, body.query,
                json.dumps(body.overrides.model_dump(exclude_none=True)) if body.overrides else None,
                latency_ms, "knowledge retrieval failed",
            )
        return JSONResponse(502, err("KNOWLEDGE_UNAVAILABLE", "knowledge retrieval failed", request_id))

    latency_ms = int((time.monotonic() - started) * 1000)

    # 落库：run + results（按 rankNo 稳定保存），同一事务
    async with pool.acquire() as conn:
        async with conn.transaction():
            run_id = await conn.fetchval(
                """INSERT INTO retrieval_test_runs
                     (external_user_id, query, overrides, status, result_count, latency_ms, finished_at)
                   VALUES ($1,$2,$3,'completed',$4,$5, NOW())
                   RETURNING id""",
                admin, body.query,
                json.dumps(body.overrides.model_dump(exclude_none=True)) if body.overrides else None,
                len(hits), latency_ms,
            )
            for rank_no, h in enumerate(hits, start=1):
                await conn.execute(
                    """INSERT INTO retrieval_test_results
                         (test_run_id, rank_no, external_kb_id, external_doc_id, external_chunk_id,
                          doc_name, text_snapshot, vector_score, rerank_score, metadata)
                       VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)""",
                    run_id, rank_no,
                    str(_pick(h, "externalKbId", "external_kb_id", default="")),
                    str(_pick(h, "externalDocId", "external_doc_id", default="")),
                    str(_pick(h, "externalChunkId", "external_chunk_id", default="")),
                    _pick(h, "documentName", "doc_name"),
                    _pick(h, "contentPreview", "text", "text_snapshot"),
                    _pick(h, "vectorScore", "score", "vector_score"),
                    _pick(h, "rerankScore", "rerank_score"),
                    json.dumps(_pick(h, "metadata", default={})),
                )

    return await _load_run(pool, run_id, request_id)


# ─────────────────────────────────────────────────────────────
# GET：读取某次测试快照（可重复审计）
# ─────────────────────────────────────────────────────────────
@router.get("/retrieval-test-runs/{testRunId}")
async def get_retrieval_test(
    testRunId: str,
    request_id: str = Depends(req_id),
    admin: str = Depends(require_admin),
    pool: asyncpg.Pool = Depends(get_pool),
):
    try:
        rid = uuid.UUID(testRunId)
    except ValueError:
        return JSONResponse(404, err("TEST_RUN_NOT_FOUND", "test run not found", request_id))
    run = await pool.fetchrow("SELECT id FROM retrieval_test_runs WHERE id = $1", rid)
    if run is None:
        return JSONResponse(404, err("TEST_RUN_NOT_FOUND", "test run not found", request_id))
    return await _load_run(pool, rid, request_id)


# ─────────────────────────────────────────────────────────────
# 公共：把 run + results 组装为 RetrievalTestResponse 形状
# ─────────────────────────────────────────────────────────────
async def _load_run(pool: asyncpg.Pool, run_id, request_id: str) -> dict:
    async with pool.acquire() as conn:
        run = await conn.fetchrow(
            "SELECT id, status, result_count, latency_ms, created_at FROM retrieval_test_runs WHERE id = $1",
            run_id,
        )
        rows = await conn.fetch(
            """SELECT rank_no, doc_name, text_snapshot, vector_score, rerank_score, metadata
               FROM retrieval_test_results WHERE test_run_id = $1 ORDER BY rank_no""",
            run_id,
        )
    results = []
    for r in rows:
        md = r["metadata"]
        if isinstance(md, str):
            md = json.loads(md)
        results.append({
            "documentName": r["doc_name"],
            "sectionPath": (md or {}).get("sectionPath"),
            "score": float(r["vector_score"]) if r["vector_score"] is not None else None,
            "rerankScore": float(r["rerank_score"]) if r["rerank_score"] is not None else None,
            "contentPreview": r["text_snapshot"],
        })
    data = {
        "id": str(run["id"]),
        "status": run["status"],
        "resultCount": run["result_count"],
        "latencyMs": run["latency_ms"],
        "results": results,                       # 已按 rankNo 升序
        "createdAt": run["created_at"].isoformat(),
    }
    return ok(data, request_id)
