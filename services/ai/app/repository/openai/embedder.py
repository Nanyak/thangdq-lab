from openai import AsyncOpenAI
from app.core.config import settings

_client = AsyncOpenAI(api_key=settings.openai_api_key)


async def embed(text: str) -> list[float]:
    resp = await _client.embeddings.create(
        model=settings.embedding_model,
        input=text,
    )
    return resp.data[0].embedding
