import logging

from app.repository.openai import embedder
from app.repository.qdrant import vector_store
from app.repository.redis import queue
from app.repository.s3 import storage
from app.worker.processor import extract_chunks

logger = logging.getLogger(__name__)


async def run() -> None:
    logger.info("embedding worker started")
    async for job in queue.consume():
        try:
            await _process(job)
        except Exception as exc:
            logger.exception("failed to process job %s", job)
            await queue.push_dead_letter(job, str(exc))


async def _process(job: dict) -> None:
    file_id: str = job["file_id"]
    file_name: str = job["file_name"]
    s3_key: str = job["s3_key"]
    user_id: str = job["user_id"]
    folder: str = job.get("folder", "")
    mime_type: str = job.get("mime_type", "text/plain")

    logger.info("processing file_id=%s name=%s mime=%s", file_id, file_name, mime_type)

    data = await storage.download(s3_key)
    chunks = await extract_chunks(data, mime_type)

    if not chunks:
        logger.warning("no chunks extracted for file_id=%s", file_id)
        return

    # Prefix each chunk with file context so short chunks embed with richer semantics
    embed_texts = [f"[{file_name}] {c.text}" for c in chunks]
    vectors = await embedder.embed_batch(embed_texts)

    embedded = [
        {"text": c.text, "page": c.page, "chunk_index": c.chunk_index, "vector": v}
        for c, v in zip(chunks, vectors)
    ]

    await vector_store.upsert(
        file_id=file_id,
        file_name=file_name,
        user_id=user_id,
        folder=folder,
        chunks=embedded,
    )

    logger.info("indexed %d chunks for file_id=%s", len(embedded), file_id)
