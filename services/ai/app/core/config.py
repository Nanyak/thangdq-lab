from pydantic import AliasChoices, Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    openai_api_key: str
    qdrant_url: str = "http://localhost:6333"
    qdrant_api_key: str = ""
    embedding_model: str = "text-embedding-3-small"
    chat_model: str = "gpt-4o-mini"
    vision_model: str = "gpt-4o"
    whisper_model: str = "whisper-1"
    similarity_threshold: float = 0.30
    top_k: int = 6
    max_chat_history_messages: int = 20
    max_chat_history_chars: int = 12000
    chunk_size: int = 800
    chunk_overlap: int = 100
    aws_access_key_id: str = ""
    aws_secret_access_key: str = ""
    aws_region: str = "us-east-1"
    s3_bucket: str = ""
    s3_endpoint: str = ""
    redis_url: str = "redis://localhost:6379"
    embedding_queue_key: str = "stuffsy:embed"
    embedding_dead_letter_key: str = "stuffsy:embed:failed"
    internal_api_key: str = Field(
        default="",
        validation_alias=AliasChoices("INTERNAL_API_KEY", "AI_INTERNAL_KEY"),
    )


settings = Settings()
