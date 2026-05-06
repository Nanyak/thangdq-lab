import asyncio
import logging

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")

from app.worker.embedding import run

if __name__ == "__main__":
    asyncio.run(run())
