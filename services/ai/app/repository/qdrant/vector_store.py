import logging
import uuid

from fastembed import SparseTextEmbedding
from qdrant_client import AsyncQdrantClient
from qdrant_client.models import (
    Distance,
    FieldCondition,
    Filter,
    Fusion,
    FusionQuery,
    MatchValue,
    PayloadSchemaType,
    PointStruct,
    Prefetch,
    SparseVector,
    SparseVectorParams,
    VectorParams,
)

from app.core.config import settings

logger = logging.getLogger(__name__)

_client = AsyncQdrantClient(
    url=settings.qdrant_url,
    api_key=settings.qdrant_api_key or None,
)

_DENSE_SIZE = 1536
_SPARSE_MODEL = "Qdrant/bm25"
_sparse_encoder = SparseTextEmbedding(model_name=_SPARSE_MODEL)

_initialized: set[str] = set()


def _collection(user_id: str) -> str:
    return user_id


def _encode_sparse(texts: list[str]) -> list[SparseVector]:
    embeddings = list(_sparse_encoder.embed(texts))
    return [
        SparseVector(indices=e.indices.tolist(), values=e.values.tolist())
        for e in embeddings
    ]


async def ensure_collection(user_id: str) -> None:
    col = _collection(user_id)
    if col in _initialized:
        return

    existing = await _client.get_collections()
    names = {c.name for c in existing.collections}

    if col in names:
        info = await _client.get_collection(col)
        has_sparse = bool(info.config.params.sparse_vectors_config)
        if not has_sparse:
            # Existing dense-only collection: recreate with hybrid config.
            # All documents must be re-uploaded to re-index.
            logger.warning(
                "collection %s missing sparse vectors — recreating for hybrid search. "
                "All files must be re-uploaded.",
                col,
            )
            await _client.delete_collection(col)
            names.discard(col)

    if col not in names:
        await _client.create_collection(
            collection_name=col,
            vectors_config={"dense": VectorParams(size=_DENSE_SIZE, distance=Distance.COSINE)},
            sparse_vectors_config={"sparse": SparseVectorParams()},
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
    query_text: str,
    user_id: str,
    scope: str,
    top_k: int,
) -> list[dict]:
    col = _collection(user_id)

    query_filter: Filter | None = None
    if scope not in ("all", ""):
        query_filter = Filter(
            must=[FieldCondition(key="folder", match=MatchValue(value=scope))]
        )

    sparse_vec = _encode_sparse([query_text])[0]

    prefetch_limit = top_k * 2
    prefetches = [
        Prefetch(query=vector, using="dense", limit=prefetch_limit, filter=query_filter),
        Prefetch(query=sparse_vec, using="sparse", limit=prefetch_limit, filter=query_filter),
    ]

    try:
        response = await _client.query_points(
            collection_name=col,
            prefetch=prefetches,
            query=FusionQuery(fusion=Fusion.RRF),
            query_filter=query_filter,
            limit=top_k,
            with_payload=True,
        )
        results = response.points
    except Exception:
        logger.exception("qdrant hybrid search failed user_id=%s scope=%s", user_id, scope)
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


async def delete_file(file_id: str, user_id: str) -> None:
    col = _collection(user_id)
    try:
        await _client.delete(
            collection_name=col,
            points_selector=Filter(
                must=[FieldCondition(key="file_id", match=MatchValue(value=file_id))]
            ),
        )
        logger.info("deleted vectors file_id=%s user_id=%s", file_id, user_id)
    except Exception:
        logger.exception("qdrant delete_file failed file_id=%s user_id=%s", file_id, user_id)


async def move_file(
    old_file_id: str,
    new_file_id: str,
    new_file_name: str,
    new_folder: str,
    user_id: str,
) -> None:
    col = _collection(user_id)
    try:
        await _client.set_payload(
            collection_name=col,
            payload={
                "file_id": new_file_id,
                "file_name": new_file_name,
                "folder": new_folder,
            },
            points=Filter(
                must=[FieldCondition(key="file_id", match=MatchValue(value=old_file_id))]
            ),
        )
        logger.info(
            "moved vector payload old_file_id=%s new_file_id=%s user_id=%s",
            old_file_id,
            new_file_id,
            user_id,
        )
    except Exception:
        logger.exception(
            "qdrant move_file failed old_file_id=%s new_file_id=%s user_id=%s",
            old_file_id,
            new_file_id,
            user_id,
        )


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

    texts = [chunk["text"] for chunk in chunks]
    sparse_vecs = _encode_sparse(texts)

    points = [
        PointStruct(
            id=str(uuid.uuid4()),
            vector={"dense": chunk["vector"], "sparse": sparse_vecs[i]},
            payload={
                "file_id": file_id,
                "file_name": file_name,
                "folder": folder,
                "text": chunk["text"],
                "page": chunk.get("page"),
                "chunk_index": chunk["chunk_index"],
            },
        )
        for i, chunk in enumerate(chunks)
    ]

    if points:
        await _client.upsert(collection_name=col, points=points)
