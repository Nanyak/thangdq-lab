import json

from fastapi import APIRouter, Depends, HTTPException, Header, Query
from fastapi.responses import StreamingResponse

from app.core.config import settings
from app.usecase import rag

router = APIRouter()


def _verify_internal_key(x_internal_key: str = Header("")) -> None:
    if settings.internal_api_key and x_internal_key != settings.internal_api_key:
        raise HTTPException(status_code=401, detail="Unauthorized")


async def _event_stream(question: str, scope: str, user_id: str):
    async for event in rag.query(question=question, scope=scope, user_id=user_id):
        yield f"data: {json.dumps(event)}\n\n"


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
