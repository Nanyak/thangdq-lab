import json

from fastapi import APIRouter, Query
from fastapi.responses import StreamingResponse

from app.usecase import rag

router = APIRouter()


async def _event_stream(question: str, scope: str, user_id: str):
    async for event in rag.query(question=question, scope=scope, user_id=user_id):
        yield f"data: {json.dumps(event)}\n\n"


@router.get("/query")
async def query_endpoint(
    q: str = Query(..., min_length=1, max_length=2000),
    scope: str = Query("all"),
    user_id: str = Query(""),
):
    return StreamingResponse(
        _event_stream(question=q, scope=scope, user_id=user_id),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
        },
    )
