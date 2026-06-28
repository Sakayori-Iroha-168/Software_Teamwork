"""
DLQA-149 — 3.2 后端：对接文件服务原文读取边界

目标：从 Citation 的 external_doc_id 安全地读取原文，QA 只做「适配 + 权限上下文 + 错误归一」，
     不复制文件能力、不保存原始文件 / object key / 文件服务内部 URL。

路径（服务级草案 /api/v1，尚未进入 gateway active paths）：
  GET /api/v1/citations/{citationId}/content   经 QA 适配，转读 file-owned 原文

依赖（QA 只消费，不实现 —— file 服务负责）：
  file-owned: GET /api/v1/documents/{documentId}/content

边界：
  - QA 只认 citation.external_doc_id（documentId），不接触文件主表/对象存储；
  - 文件服务内部地址来自配置，绝不返回给前端；
  - file 返回无权/缺失/依赖错误时，错误标准化，且引用快照仍可通过
    GET /api/v1/citations/{citationId} 查看（互不影响）。

集成进 services/qa/ 时，get_pool / current_user_id / req_id / err / _to_uuid
建议与 citations_api.py 复用同一份实现（这里为便于单文件审阅而内联）。
"""
from __future__ import annotations

import os
import uuid
from typing import Optional

import asyncpg
import httpx
from fastapi import APIRouter, Depends, Header, HTTPException
from fastapi.responses import JSONResponse, StreamingResponse

router = APIRouter(prefix="/api/v1", tags=["citations"])

# 文件服务内部地址：仅服务端使用，绝不外泄
FILE_SERVICE_BASE_URL = os.getenv("FILE_SERVICE_BASE_URL", "http://file-service")
FILE_READ_TIMEOUT = float(os.getenv("FILE_READ_TIMEOUT", "30"))


# ─────────────────────────────────────────────────────────────
# 依赖 / 工具占位 —— 集成时与 citations_api.py 复用
# ─────────────────────────────────────────────────────────────
async def get_pool() -> asyncpg.Pool:  # pragma: no cover
    raise NotImplementedError("用团队的 asyncpg 连接池替换")


async def current_user_id(
    x_user_id: Optional[str] = Header(default=None, alias="X-User-Id"),
) -> str:  # pragma: no cover
    if not x_user_id:
        raise HTTPException(status_code=401, detail="unauthenticated")
    return x_user_id


def req_id(x_request_id: Optional[str] = Header(default=None, alias="X-Request-Id")) -> str:
    return x_request_id or str(uuid.uuid4())


def err(code: str, message: str, request_id: str, fields: Optional[dict] = None) -> dict:
    e: dict = {"code": code, "message": message, "requestId": request_id}
    if fields:
        e["fields"] = fields
    return {"error": e}


def _to_uuid(s: str) -> Optional[uuid.UUID]:
    try:
        return uuid.UUID(str(s))
    except (ValueError, AttributeError, TypeError):
        return None


# ─────────────────────────────────────────────────────────────
# 错误归一：file 服务状态码 → QA 标准错误信封
# ─────────────────────────────────────────────────────────────
def _normalize_file_error(status: int, request_id: str) -> tuple[int, dict]:
    if status in (401, 403):
        return 403, err("DOCUMENT_FORBIDDEN", "no permission to read document", request_id)
    if status == 404:
        return 404, err("DOCUMENT_NOT_FOUND", "document not found", request_id)
    # 其它（含依赖错误 / 5xx）统一归一为上游不可用
    return 502, err("FILE_SERVICE_UNAVAILABLE", "upstream file service error", request_id)


# ─────────────────────────────────────────────────────────────
# 接口：从引用进入原文
# ─────────────────────────────────────────────────────────────
@router.get("/citations/{citationId}/content")
async def read_citation_source(
    citationId: str,
    request_id: str = Depends(req_id),
    user: str = Depends(current_user_id),
    pool: asyncpg.Pool = Depends(get_pool),
):
    # 1) 校验引用归属，仅取 external_doc_id（QA 不碰文件内部）
    cid = _to_uuid(citationId)
    if cid is None:
        return JSONResponse(404, err("CITATION_NOT_FOUND", "citation not found", request_id))
    sql = """
        SELECT c.external_doc_id
        FROM citations c
        JOIN messages m       ON m.id = c.message_id
        JOIN conversations cv ON cv.id = m.conversation_id
        WHERE c.id = $1 AND cv.external_user_id = $2
    """
    async with pool.acquire() as conn:
        row = await conn.fetchrow(sql, cid, user)
    if row is None:
        # 不存在 / 越权 → 一律 404（不泄露），引用快照接口不受影响
        return JSONResponse(404, err("CITATION_NOT_FOUND", "citation not found", request_id))
    document_id = row["external_doc_id"]

    # 2) 适配 file-owned 接口，透传权限上下文（X-User-Id / X-Request-Id）
    client = httpx.AsyncClient(base_url=FILE_SERVICE_BASE_URL, timeout=FILE_READ_TIMEOUT)
    try:
        upstream = await client.send(
            client.build_request(
                "GET",
                f"/api/v1/documents/{document_id}/content",
                headers={"X-User-Id": user, "X-Request-Id": request_id},
            ),
            stream=True,
        )
    except httpx.RequestError:
        await client.aclose()
        return JSONResponse(502, err("FILE_SERVICE_UNAVAILABLE", "file service unreachable", request_id))

    # 3) 错误归一（拿到状态码后再决定是否透传流）
    if upstream.status_code >= 400:
        status = upstream.status_code
        await upstream.aclose()
        await client.aclose()
        code, body = _normalize_file_error(status, request_id)
        return JSONResponse(code, body)

    # 4) 成功：流式透传内容，保留 content-type，但不暴露内部地址 / object key
    content_type = upstream.headers.get("content-type", "application/octet-stream")

    async def body_iter():
        try:
            async for chunk in upstream.aiter_bytes():
                yield chunk
        finally:
            await upstream.aclose()
            await client.aclose()

    return StreamingResponse(
        body_iter(),
        media_type=content_type,
        headers={"X-Request-Id": request_id},
    )
