import logging
import uuid

from qdrant_client import AsyncQdrantClient
from qdrant_client.models import Distance, FieldCondition, Filter, MatchValue, PayloadSchemaType, PointStruct, VectorParams

from app.core.config import settings

logger = logging.getLogger(__name__)

_client = AsyncQdrantClient(
    url=settings.qdrant_url,
    api_key=settings.qdrant_api_key or None,
)

_VECTOR_SIZE = 1536
_initialized: set[str] = set()


def _collection(user_id: str) -> str:
    return user_id


async def ensure_collection(user_id: str) -> None:
    col = _collection(user_id)
    if col in _initialized:
        return

    existing = await _client.get_collections()
    names = {c.name for c in existing.collections}
    if col not in names:
        await _client.create_collection(
            collection_name=col,
            vectors_config=VectorParams(size=_VECTOR_SIZE, distance=Distance.COSINE),
        )

    for field in ("file_id", "folder"):
        await _client.create_payload_index(
            collection_name=col,
            field_name=field,
            field_schema=PayloadSchemaType.KEYWORD,
        )

    _initialized.add(col)


async def search(
    vector: list[float],
    user_id: str,
    scope: str,
    top_k: int,
) -> list[dict]:
    col = _collection(user_id)
    must: list = []
    if scope not in ("all", ""):
        must.append(FieldCondition(key="folder", match=MatchValue(value=scope)))

    try:
        response = await _client.query_points(
            collection_name=col,
            query=vector,
            query_filter=Filter(must=must) if must else None,
            limit=top_k,
            with_payload=True,
            score_threshold=settings.similarity_threshold,
        )
        results = response.points
    except Exception:
        logger.exception("qdrant search failed user_id=%s scope=%s", user_id, scope)
        return []

    return [
        {
            "score": r.score,
            "file_id": r.payload.get("file_id", ""),
            "file_name": r.payload.get("file_name", ""),
            "page": r.payload.get("page"),
            "chunk_index": r.payload.get("chunk_index", 0),
            "text": r.payload.get("text", ""),
        }
        for r in results
    ]


async def upsert(
    file_id: str,
    file_name: str,
    user_id: str,
    folder: str,
    chunks: list[dict],
) -> None:
    col = _collection(user_id)
    await ensure_collection(user_id)

    await _client.delete(
        collection_name=col,
        points_selector=Filter(
            must=[FieldCondition(key="file_id", match=MatchValue(value=file_id))]
        ),
    )

    points = [
        PointStruct(
            id=str(uuid.uuid4()),
            vector=chunk["vector"],
            payload={
                "file_id": file_id,
                "file_name": file_name,
                "folder": folder,
                "text": chunk["text"],
                "page": chunk.get("page"),
                "chunk_index": chunk["chunk_index"],
            },
        )
        for chunk in chunks
    ]

    if points:
        await _client.upsert(collection_name=col, points=points)
