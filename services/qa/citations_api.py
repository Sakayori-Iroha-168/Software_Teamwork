"""
DLQA-148 — 3.1 后端：引用快照生成与查询接口

路径（服务级草案 /api/v1，尚未进入 gateway active paths）：
  GET  /api/v1/messages/{messageId}/citations   某条消息的引用列表（按 citationNo 升序）
  GET  /api/v1/citations/{citationId}           单条引用详情
  POST /api/v1/citation-lookups                 批量查询（保序、可重复、部分缺失）

框架：FastAPI + asyncpg（PostgreSQL）
数据源：citations 表（external_kb_id/external_doc_id/external_chunk_id + 快照字段）
响应规范：成功 {data, requestId}；错误 {error:{code,message,requestId,fields?}}；204 除外
访问控制：按会话归属（citations → messages → conversations.external_user_id）

集成进 services/qa/ 时，把 get_pool / current_user_id 换成团队的连接池与鉴权实现。
"""
from __future__ import annotations

import json
import uuid
from typing import Any, Optional

import asyncpg
from fastapi import APIRouter, Depends, Header, HTTPException
from fastapi.responses import JSONResponse
from pydantic import BaseModel

router = APIRouter(prefix="/api/v1", tags=["citations"])

MAX_LOOKUP_IDS = 200


# ─────────────────────────────────────────────────────────────
# DTO（对外字段用 camelCase）
# ─────────────────────────────────────────────────────────────
class CitationDTO(BaseModel):
    id: str
    citationNo: int
    charStart: Optional[int] = None
    charEnd: Optional[int] = None
    externalKbId: str
    externalDocId: str
    externalChunkId: str
    docName: Optional[str] = None
    quoteText: Optional[str] = None
    context: Optional[str] = None
    pageNumber: Optional[int] = None
    score: Optional[float] = None
    metadata: Optional[dict] = None


def _row_to_dto(r: asyncpg.Record) -> dict:
    md = r["metadata"]
    if isinstance(md, str):            # asyncpg 默认把 jsonb 当字符串返回
        md = json.loads(md)
    return CitationDTO(
        id=str(r["id"]),
        citationNo=r["citation_no"],
        charStart=r["char_start"],
        charEnd=r["char_end"],
        externalKbId=r["external_kb_id"],
        externalDocId=r["external_doc_id"],
        externalChunkId=r["external_chunk_id"],
        docName=r["doc_name"],
        quoteText=r["quote_text"],
        context=r["context"],
        pageNumber=r["page_number"],
        score=float(r["score"]) if r["score"] is not None else None,
        metadata=md,
    ).model_dump()


# ─────────────────────────────────────────────────────────────
# 依赖占位 —— 集成时替换为团队实现
# ─────────────────────────────────────────────────────────────
async def get_pool() -> asyncpg.Pool:  # pragma: no cover
    raise NotImplementedError("用团队的 asyncpg 连接池替换")


async def current_user_id(
    x_user_id: Optional[str] = Header(default=None, alias="X-User-Id"),
) -> str:  # pragma: no cover
    """返回当前请求者的 external_user_id。集成时替换为团队鉴权。"""
    if not x_user_id:
        raise HTTPException(status_code=401, detail="unauthenticated")
    return x_user_id


def req_id(x_request_id: Optional[str] = Header(default=None, alias="X-Request-Id")) -> str:
    return x_request_id or str(uuid.uuid4())


# ─────────────────────────────────────────────────────────────
# 响应信封
# ─────────────────────────────────────────────────────────────
def ok(data: Any, request_id: str) -> dict:
    return {"data": data, "requestId": request_id}


def err(code: str, message: str, request_id: str, fields: Optional[dict] = None) -> dict:
    e: dict = {"code": code, "message": message, "requestId": request_id}
    if fields:
        e["fields"] = fields
    return {"error": e}


def _to_uuid(s: str) -> Optional[uuid.UUID]:
    """字符串转 UUID；非法返回 None（不抛错，便于按草案给可预测结果）。"""
    try:
        return uuid.UUID(str(s))
    except (ValueError, AttributeError, TypeError):
        return None


# ─────────────────────────────────────────────────────────────
# SQL —— 统一带「会话归属」过滤，避免越权读取
# ─────────────────────────────────────────────────────────────
_COLS = """
  c.id, c.citation_no, c.char_start, c.char_end,
  c.external_kb_id, c.external_doc_id, c.external_chunk_id,
  c.doc_name, c.quote_text, c.context, c.page_number, c.score, c.metadata
"""
_OWNED_JOIN = """
  FROM citations c
  JOIN messages m       ON m.id = c.message_id
  JOIN conversations cv ON cv.id = m.conversation_id
"""


# ── 1) 列表：某条消息的引用 ──────────────────────────────────
@router.get("/messages/{messageId}/citations")
async def list_message_citations(
    messageId: str,
    request_id: str = Depends(req_id),
    user: str = Depends(current_user_id),
    pool: asyncpg.Pool = Depends(get_pool),
):
    mid = _to_uuid(messageId)
    if mid is None:
        # 非法 id：返回空列表（可预测、不泄露是否存在）
        return ok([], request_id)
    sql = f"SELECT {_COLS} {_OWNED_JOIN} WHERE c.message_id = $1 AND cv.external_user_id = $2 ORDER BY c.citation_no"
    async with pool.acquire() as conn:
        rows = await conn.fetch(sql, mid, user)
    # 消息不存在或属他人 → 同样返回空列表
    return ok([_row_to_dto(r) for r in rows], request_id)


# ── 2) 单条详情 ──────────────────────────────────────────────
@router.get("/citations/{citationId}")
async def get_citation(
    citationId: str,
    request_id: str = Depends(req_id),
    user: str = Depends(current_user_id),
    pool: asyncpg.Pool = Depends(get_pool),
):
    cid = _to_uuid(citationId)
    if cid is None:
        return JSONResponse(404, err("CITATION_NOT_FOUND", "citation not found", request_id))
    sql = f"SELECT {_COLS} {_OWNED_JOIN} WHERE c.id = $1 AND cv.external_user_id = $2"
    async with pool.acquire() as conn:
        row = await conn.fetchrow(sql, cid, user)
    if row is None:
        # 不存在或属他人 → 一律 404（不区分，避免泄露）
        return JSONResponse(404, err("CITATION_NOT_FOUND", "citation not found", request_id))
    return ok(_row_to_dto(row), request_id)


# ── 3) 批量查询：保序、可重复、部分缺失 ─────────────────────
class LookupReq(BaseModel):
    citationIds: list[str]


@router.post("/citation-lookups")
async def citation_lookups(
    body: LookupReq,
    request_id: str = Depends(req_id),
    user: str = Depends(current_user_id),
    pool: asyncpg.Pool = Depends(get_pool),
):
    ids = body.citationIds
    if len(ids) > MAX_LOOKUP_IDS:
        return JSONResponse(
            400,
            err("TOO_MANY_IDS", f"max {MAX_LOOKUP_IDS} ids", request_id, {"citationIds": f"<= {MAX_LOOKUP_IDS}"}),
        )

    # 去重 + 过滤非法 id，仅用合法 UUID 查询一次
    valid = {s: u for s in ids if (u := _to_uuid(s)) is not None}
    found: dict[str, dict] = {}
    if valid:
        sql = f"SELECT {_COLS} {_OWNED_JOIN} WHERE c.id = ANY($1::uuid[]) AND cv.external_user_id = $2"
        async with pool.acquire() as conn:
            rows = await conn.fetch(sql, list(set(valid.values())), user)
        found = {str(r["id"]): _row_to_dto(r) for r in rows}

    # 按请求顺序映射，保留重复；缺失（不存在/越权/非法）标 found=false
    items = [{"id": s, "found": s in found, "citation": found.get(s)} for s in ids]
    missing = sorted({s for s in ids if s not in found})
    return ok({"items": items, "missing": missing}, request_id)
