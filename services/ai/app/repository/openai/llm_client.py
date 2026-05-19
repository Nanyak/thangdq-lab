from collections.abc import AsyncGenerator

from langchain_openai import ChatOpenAI
from openai import AsyncOpenAI
from app.core.config import settings

_client = AsyncOpenAI(api_key=settings.openai_api_key)
_langchain_chat = ChatOpenAI(
    api_key=settings.openai_api_key,
    model=settings.chat_model,
    temperature=0.1,
    max_tokens=1024,
)


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


async def chat_with_tools(messages: list[dict], tools: list[dict]):
    return await _client.chat.completions.create(
        model=settings.chat_model,
        messages=messages,
        tools=tools,
        tool_choice="auto",
        temperature=0.1,
        max_completion_tokens=1024,
    )


def langchain_chat() -> ChatOpenAI:
    return _langchain_chat
