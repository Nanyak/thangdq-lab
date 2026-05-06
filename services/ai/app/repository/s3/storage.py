import asyncio
import boto3
from app.core.config import settings

_s3 = boto3.client(
    "s3",
    region_name=settings.aws_region,
    aws_access_key_id=settings.aws_access_key_id,
    aws_secret_access_key=settings.aws_secret_access_key,
)


async def download(s3_key: str) -> bytes:
    def _get():
        resp = _s3.get_object(Bucket=settings.s3_bucket, Key=s3_key)
        return resp["Body"].read()

    return await asyncio.to_thread(_get)
