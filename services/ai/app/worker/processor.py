import asyncio
import base64
import io
import subprocess
import tempfile
from dataclasses import dataclass

import cv2
import pypdf
from PIL import Image
from openai import AsyncOpenAI

from app.core.config import settings

_client = AsyncOpenAI(api_key=settings.openai_api_key)

_FRAME_INTERVAL_SEC = 30


@dataclass
class Chunk:
    text: str
    page: int | None
    chunk_index: int


def _split(text: str) -> list[str]:
    pieces, start = [], 0
    text = text.strip()
    while start < len(text):
        end = min(start + settings.chunk_size, len(text))
        piece = text[start:end].strip()
        if piece:
            pieces.append(piece)
        if end == len(text):
            break
        start += settings.chunk_size - settings.chunk_overlap
    return pieces


async def _describe_image_bytes(data: bytes) -> str:
    img = Image.open(io.BytesIO(data))
    fmt = (img.format or "PNG").lower()
    mime = "image/jpeg" if fmt in ("jpg", "jpeg") else f"image/{fmt}"
    b64 = base64.b64encode(data).decode()

    resp = await _client.chat.completions.create(
        model=settings.vision_model,
        messages=[{
            "role": "user",
            "content": [
                {"type": "image_url", "image_url": {"url": f"data:{mime};base64,{b64}"}},
                {"type": "text", "text": "Describe the content of this image in detail."},
            ],
        }],
        max_tokens=1024,
    )
    return resp.choices[0].message.content or ""


async def _extract_pdf(data: bytes) -> list[Chunk]:
    chunks: list[Chunk] = []
    reader = pypdf.PdfReader(io.BytesIO(data))
    for page_num, page in enumerate(reader.pages, start=1):
        text = page.extract_text() or ""
        for piece in _split(text):
            chunks.append(Chunk(text=piece, page=page_num, chunk_index=len(chunks)))
    return chunks


async def _extract_image(data: bytes) -> list[Chunk]:
    description = await _describe_image_bytes(data)
    return [Chunk(text=description, page=None, chunk_index=0)]


async def _extract_text(data: bytes) -> list[Chunk]:
    text = data.decode("utf-8", errors="replace")
    return [
        Chunk(text=piece, page=None, chunk_index=i)
        for i, piece in enumerate(_split(text))
    ]


async def _extract_video(data: bytes) -> list[Chunk]:
    chunks: list[Chunk] = []

    with tempfile.TemporaryDirectory() as tmp:
        video_path = f"{tmp}/input.mp4"
        audio_path = f"{tmp}/audio.mp3"

        with open(video_path, "wb") as f:
            f.write(data)

        # Transcribe audio via Whisper
        result = subprocess.run(
            ["ffmpeg", "-i", video_path, "-vn", "-ar", "16000", "-ac", "1", "-q:a", "0", audio_path],
            capture_output=True,
        )
        if result.returncode == 0:
            with open(audio_path, "rb") as f:
                transcript = await _client.audio.transcriptions.create(
                    model=settings.whisper_model,
                    file=f,
                    response_format="text",
                )
            for piece in _split(str(transcript)):
                chunks.append(Chunk(text=piece, page=None, chunk_index=len(chunks)))

        # Describe keyframes via vision
        cap = cv2.VideoCapture(video_path)
        fps = cap.get(cv2.CAP_PROP_FPS) or 25
        interval = int(fps * _FRAME_INTERVAL_SEC)
        frame_bufs: list[bytes] = []
        idx = 0
        while True:
            ret, frame = cap.read()
            if not ret:
                break
            if idx % interval == 0:
                _, buf = cv2.imencode(".jpg", frame)
                frame_bufs.append(buf.tobytes())
            idx += 1
        cap.release()

        if frame_bufs:
            descriptions = await asyncio.gather(*[_describe_image_bytes(b) for b in frame_bufs])
            for desc in descriptions:
                for piece in _split(desc):
                    chunks.append(Chunk(text=piece, page=None, chunk_index=len(chunks)))

    return chunks


_HANDLERS = {
    "application/pdf": _extract_pdf,
    "image/jpeg": _extract_image,
    "image/png": _extract_image,
    "image/gif": _extract_image,
    "image/webp": _extract_image,
    "video/mp4": _extract_video,
    "video/quicktime": _extract_video,
    "video/x-msvideo": _extract_video,
    "video/webm": _extract_video,
    "text/plain": _extract_text,
    "text/markdown": _extract_text,
    "text/csv": _extract_text,
}


async def extract_chunks(data: bytes, mime_type: str) -> list[Chunk]:
    handler = _HANDLERS.get(mime_type, _extract_text)
    return await handler(data)
