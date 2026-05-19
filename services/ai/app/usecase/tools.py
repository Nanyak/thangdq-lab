import asyncio
import posixpath
from dataclasses import dataclass
from mimetypes import guess_type

import boto3
from botocore.exceptions import ClientError

from app.core.config import settings
from app.repository.openai import embedder
from app.repository.qdrant import vector_store

_s3 = boto3.client(
    "s3",
    region_name=settings.aws_region,
    aws_access_key_id=settings.aws_access_key_id,
    aws_secret_access_key=settings.aws_secret_access_key,
    endpoint_url=settings.s3_endpoint or None,
)


@dataclass
class ToolResult:
    content: dict
    sources: list[dict] | None = None


def _clean_relative_key(key: str) -> str:
    key = key.strip().strip("/")
    parts = [p for p in key.split("/") if p and p != "."]
    if any(p == ".." for p in parts):
        raise ValueError("file keys cannot contain path traversal")
    return "/".join(parts)


def _clean_folder(folder: str) -> str:
    return _clean_relative_key(folder)


def _user_prefix(user_id: str) -> str:
    return f"{user_id}/"


def _relative_key(user_id: str, full_key: str) -> str:
    return full_key.removeprefix(_user_prefix(user_id))


def _folder_for(key: str) -> str:
    folder = posixpath.dirname(key)
    return "" if folder == "." else folder


async def search_files(query: str, scope: str, user_id: str) -> ToolResult:
    vector = await embedder.embed(query)
    chunks = await vector_store.search(
        vector=vector,
        query_text=query,
        user_id=user_id,
        scope=scope,
        top_k=settings.top_k,
    )
    sources = [
        {
            "file_id": chunk["file_id"],
            "file_name": chunk["file_name"],
            "page": chunk.get("page"),
            "chunk_index": chunk["chunk_index"],
            "text": chunk["text"],
        }
        for chunk in chunks
    ]
    return ToolResult(
        content={
            "chunks": [
                {
                    "file_name": chunk["file_name"],
                    "page": chunk.get("page"),
                    "chunk_index": chunk["chunk_index"],
                    "text": chunk["text"],
                }
                for chunk in chunks
            ]
        },
        sources=sources,
    )


async def list_files(user_id: str, folder: str = "") -> ToolResult:
    folder = _clean_folder(folder) if folder else ""
    prefix = _user_prefix(user_id) + (f"{folder}/" if folder else "")

    def _list() -> list[dict]:
        paginator = _s3.get_paginator("list_objects_v2")
        files: list[dict] = []
        for page in paginator.paginate(Bucket=settings.s3_bucket, Prefix=prefix):
            for obj in page.get("Contents", []):
                full_key = obj["Key"]
                if full_key.endswith("/"):
                    continue
                key = _relative_key(user_id, full_key)
                content_type = guess_type(key)[0] or "application/octet-stream"
                files.append(
                    {
                        "key": key,
                        "name": posixpath.basename(key),
                        "folder": _folder_for(key),
                        "size": obj.get("Size", 0),
                        "content_type": content_type,
                        "last_modified": obj["LastModified"].isoformat()
                        if obj.get("LastModified")
                        else "",
                    }
                )
        return files

    return ToolResult(content={"files": await asyncio.to_thread(_list)})


def _object_exists(full_key: str) -> bool:
    try:
        _s3.head_object(Bucket=settings.s3_bucket, Key=full_key)
        return True
    except ClientError as exc:
        status = exc.response.get("ResponseMetadata", {}).get("HTTPStatusCode")
        if status == 404:
            return False
        raise


def _unique_destination(user_id: str, folder: str, name: str) -> str:
    base, ext = posixpath.splitext(name)
    for idx in range(0, 1000):
        candidate_name = name if idx == 0 else f"{base} ({idx}){ext}"
        rel = f"{folder}/{candidate_name}" if folder else candidate_name
        full = _user_prefix(user_id) + rel
        if not _object_exists(full):
            return rel
    raise ValueError("could not allocate a unique destination key")


async def organize_files(user_id: str, moves: list[dict]) -> ToolResult:
    planned: list[dict] = []
    for move in moves:
        source_key = _clean_relative_key(str(move.get("key", "")))
        destination_folder = _clean_folder(str(move.get("destination_folder", "")))
        if not source_key:
            raise ValueError("each move requires a file key")
        source_name = posixpath.basename(source_key)
        destination_key = _unique_destination(user_id, destination_folder, source_name)
        if source_key == destination_key:
            planned.append(
                {"key": source_key, "destination_key": destination_key, "status": "unchanged"}
            )
            continue
        planned.append(
            {
                "key": source_key,
                "destination_key": destination_key,
                "destination_folder": destination_folder,
                "status": "planned",
            }
        )

    def _move() -> list[dict]:
        completed: list[dict] = []
        for item in planned:
            if item["status"] == "unchanged":
                completed.append(item)
                continue

            source_full = _user_prefix(user_id) + item["key"]
            destination_full = _user_prefix(user_id) + item["destination_key"]
            _s3.copy_object(
                Bucket=settings.s3_bucket,
                CopySource={"Bucket": settings.s3_bucket, "Key": source_full},
                Key=destination_full,
            )
            _s3.delete_object(Bucket=settings.s3_bucket, Key=source_full)
            completed.append({**item, "status": "moved"})
        return completed

    completed = await asyncio.to_thread(_move)

    for item in completed:
        if item["status"] != "moved":
            continue
        await vector_store.move_file(
            old_file_id=_user_prefix(user_id) + item["key"],
            new_file_id=_user_prefix(user_id) + item["destination_key"],
            new_file_name=posixpath.basename(item["destination_key"]),
            new_folder=item["destination_folder"],
            user_id=user_id,
        )

    return ToolResult(content={"moves": completed})
