from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.api.routes import router
from app.repository.qdrant import vector_store


@asynccontextmanager
async def lifespan(app: FastAPI):
    await vector_store.ensure_collection()
    yield


app = FastAPI(title="Stuffsy AI Service", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["GET"],
    allow_headers=["*"],
)

app.include_router(router, prefix="/v1/ai")
