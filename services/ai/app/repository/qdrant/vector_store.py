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

# text-embedding-3-small output dimension
_VECTOR_SIZE = 1536


async def ensure_collection() -> None:
    existing = await _client.get_collections()
    names = {c.name for c in existing.collections}
    if settings.qdrant_collection not in names:
        await _client.create_collection(
            collection_name=settings.qdrant_collection,
            vectors_config=VectorParams(size=_VECTOR_SIZE, distance=Distance.COSINE),
        )

    # Ensure payload indexes required for filtering
    for field in ("file_id", "user_id", "folder"):
        await _client.create_payload_index(
            collection_name=settings.qdrant_collection,
            field_name=field,
            field_schema=PayloadSchemaType.KEYWORD,
        )


async def search(
    vector: list[float],
    user_id: str,
    scope: str,
    top_k: int,
) -> list[dict]:
    must: list = [FieldCondition(key="user_id", match=MatchValue(value=user_id))]
    if scope not in ("all", ""):
        must.append(FieldCondition(key="folder", match=MatchValue(value=scope)))

    try:
        results = await _client.search(
            collection_name=settings.qdrant_collection,
            query_vector=vector,
            query_filter=Filter(must=must),
            limit=top_k,
            with_payload=True,
        )
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
    # Remove existing points for this file before re-indexing
    await _client.delete(
        collection_name=settings.qdrant_collection,
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
                "user_id": user_id,
                "folder": folder,
                "text": chunk["text"],
                "page": chunk.get("page"),
                "chunk_index": chunk["chunk_index"],
            },
        )
        for chunk in chunks
    ]

    if points:
        await _client.upsert(collection_name=settings.qdrant_collection, points=points)
