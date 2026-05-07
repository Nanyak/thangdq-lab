from openai import AsyncOpenAI
from app.core.config import settings

_client = AsyncOpenAI(api_key=settings.openai_api_key)


async def embed(text: str) -> list[float]:
    resp = await _client.embeddings.create(
        model=settings.embedding_model,
        input=text,
    )
    return resp.data[0].embedding


async def embed_batch(texts: list[str]) -> list[list[float]]:
    resp = await _client.embeddings.create(
        model=settings.embedding_model,
        input=texts,
    )
    return [d.embedding for d in sorted(resp.data, key=lambda d: d.index)]
