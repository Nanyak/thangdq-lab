import json
from collections.abc import AsyncGenerator

import redis.asyncio as aioredis

from app.core.config import settings

_pool = aioredis.ConnectionPool.from_url(settings.redis_url, decode_responses=True)


async def consume() -> AsyncGenerator[dict, None]:
    client = aioredis.Redis(connection_pool=_pool)
    while True:
        item = await client.blpop(settings.embedding_queue_key, timeout=5)
        if item is None:
            continue
        _, raw = item
        yield json.loads(raw)


async def push(job: dict) -> None:
    client = aioredis.Redis(connection_pool=_pool)
    await client.rpush(settings.embedding_queue_key, json.dumps(job))
