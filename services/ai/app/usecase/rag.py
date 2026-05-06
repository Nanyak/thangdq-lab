from collections.abc import AsyncGenerator

from app.core.config import settings
from app.repository.openai import embedder, llm_client
from app.repository.qdrant import vector_store

_SYSTEM_PROMPT = """\
You are Stuffsy AI, the built-in assistant for the Stuffsy platform.

Stuffsy is a personal productivity platform with three features:
  1. File Storage   – upload, organise, and manage files in folders.
  2. URL Shortener  – shorten long URLs and track click history.
  3. AI Assistant   – (that's you) answer questions about the user's own files.

YOUR STRICT SCOPE:
  • Answer ONLY questions about the Stuffsy platform itself (features, how to use it).
  • Answer ONLY questions about the content of the user's own uploaded files, \
using the RETRIEVED CONTEXT block when provided.
  • If the user asks about anything else — general knowledge, current events, \
coding help, external topics — politely decline and explain your scope.

CONFIDENCE RULES:
  • If the RETRIEVED CONTEXT is empty or does not contain information relevant \
to the question, respond exactly with:
    "I don't have enough information to answer that confidently. \
Could you be more specific, or try uploading relevant documents first?"
  • Never fabricate file contents or invent facts.
  • If a question is about Stuffsy's own features (not user files), answer from \
your built-in knowledge of the platform without needing context.

TONE: concise, direct, professional. No filler phrases."""


async def query(
    question: str,
    scope: str,
    user_id: str,
) -> AsyncGenerator[dict, None]:
    vector = await embedder.embed(question)
    chunks = await vector_store.search(
        vector=vector,
        user_id=user_id,
        scope=scope,
        top_k=settings.top_k,
    )

    context_block = (
        "\n\n".join(f"[{c['file_name']}]\n{c['text']}" for c in chunks)
        if chunks
        else ""
    )

    messages: list[dict] = [{"role": "system", "content": _SYSTEM_PROMPT}]
    if context_block:
        messages.append({
            "role": "system",
            "content": f"RETRIEVED CONTEXT (from the user's files):\n{context_block}",
        })
    messages.append({"role": "user", "content": question})

    for chunk in chunks:
        yield {
            "source": {
                "file_id": chunk["file_id"],
                "file_name": chunk["file_name"],
                "page": chunk.get("page"),
                "chunk_index": chunk["chunk_index"],
                "text": chunk["text"],
            }
        }

    async for token in llm_client.stream_chat(messages):
        yield {"token": token}

    yield {"done": True}
