import json

from fastapi import APIRouter, Depends, HTTPException, Header, Query
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field

from app.core.config import settings
from app.usecase import rag

router = APIRouter()


class ChatMessage(BaseModel):
    role: str = Field(..., pattern="^(user|assistant)$")
    content: str = Field(..., min_length=1, max_length=4000)


class ChatRequest(BaseModel):
    message: str = Field(..., min_length=1, max_length=4000)
    scope: str = "all"
    user_id: str = Field(..., min_length=1)
    allow_mutations: bool = False
    history: list[ChatMessage] = Field(default_factory=list)


def _verify_internal_key(x_internal_key: str = Header("")) -> None:
    if not settings.internal_api_key:
        raise HTTPException(status_code=503, detail="Internal API key is not configured")
    if x_internal_key != settings.internal_api_key:
        raise HTTPException(status_code=401, detail="Unauthorized")


async def _event_stream(question: str, scope: str, user_id: str):
    async for event in rag.query(question=question, scope=scope, user_id=user_id):
        yield f"data: {json.dumps(event, default=str)}\n\n"


async def _chat_event_stream(req: ChatRequest):
    history = [item.model_dump() for item in req.history]
    async for event in rag.chat(
        message=req.message,
        scope=req.scope,
        user_id=req.user_id,
        allow_mutations=req.allow_mutations,
        history=history,
    ):
        yield f"data: {json.dumps(event, default=str)}\n\n"


@router.get("/query", dependencies=[Depends(_verify_internal_key)])
async def query_endpoint(
    q: str = Query(..., min_length=1, max_length=2000),
    scope: str = Query("all"),
    user_id: str = Query(..., min_length=1),
):
    return StreamingResponse(
        _event_stream(question=q, scope=scope, user_id=user_id),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
        },
    )


@router.post("/chat", dependencies=[Depends(_verify_internal_key)])
async def chat_endpoint(req: ChatRequest):
    return StreamingResponse(
        _chat_event_stream(req),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
        },
    )
