from collections.abc import AsyncGenerator

from openai import AsyncOpenAI
from app.core.config import settings

_client = AsyncOpenAI(api_key=settings.openai_api_key)


async def stream_chat(messages: list[dict]) -> AsyncGenerator[str, None]:
    stream = await _client.chat.completions.create(
        model=settings.chat_model,
        messages=messages,
        stream=True,
        temperature=0.1,
        max_completion_tokens=1024,
    )
    async for chunk in stream:
        delta = chunk.choices[0].delta
        if delta.content:
            yield delta.content
