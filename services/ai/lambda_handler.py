import asyncio
import json
import logging

from app.worker.embedding import _process

logger = logging.getLogger()
logger.setLevel(logging.INFO)


def handler(event, context):
    for record in event.get("Records", []):
        body = json.loads(record["body"])
        logger.info("processing sqs message %s", record["messageId"])
        try:
            asyncio.run(_process(body))
        except Exception:
            logger.exception("failed job %s", record["messageId"])
            raise  # triggers SQS retry via visibility timeout
