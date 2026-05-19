from collections.abc import AsyncGenerator
from dataclasses import dataclass, field
import re
from uuid import uuid4

from langchain.agents import create_agent
from langchain_core.messages import AIMessage, BaseMessage, HumanMessage
from langchain_core.tools import StructuredTool
from langgraph.checkpoint.memory import InMemorySaver
from pydantic import BaseModel, Field

from app.core.config import settings
from app.repository.openai import embedder, llm_client
from app.repository.qdrant import vector_store
from app.usecase import tools as file_tools

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

_AGENT_SYSTEM_PROMPT = """\
You are Stuffsy AI, the built-in assistant for the Stuffsy platform.

You answer questions about Stuffsy and the user's own files. Use tools as needed,
observe the result, then either call another tool or answer. Do not reveal hidden
reasoning.

Available capabilities:
- search_files: retrieve relevant chunks from uploaded files for RAG answers.
- list_files: list the user's uploaded files and folders.
- organize_files: move files into folders when the user asks you to organize,
  tidy, categorize, or move files.

Rules:
- Use tools before answering questions about uploaded file contents or file lists.
- Before organizing files, list files unless the user provided exact file keys.
- Only organize files when the user clearly asks for file movement or organization.
- Never invent file contents. If search_files returns no useful chunks, say you do
  not have enough information.
- Stay within Stuffsy, user files, and file organization. Politely decline other topics.
- Keep responses concise and include what changed after organize_files succeeds."""


class SearchFilesInput(BaseModel):
    query: str = Field(..., description="Question or search phrase for RAG retrieval.")
    scope: str = Field("all", description="Folder scope. Use all for all files.")


class ListFilesInput(BaseModel):
    folder: str = Field("", description="Folder path relative to the user's root.")


class FileMoveInput(BaseModel):
    key: str = Field(..., description="Current file key relative to the user's root.")
    destination_folder: str = Field(
        ..., description="Destination folder relative to the user's root."
    )


class OrganizeFilesInput(BaseModel):
    moves: list[FileMoveInput] = Field(..., description="File moves to apply.")


@dataclass
class AgentRunState:
    sources: list[dict] = field(default_factory=list)


_CONFIRMATION_MESSAGES = {
    "confirm",
    "yes",
    "yes apply",
    "yes, apply",
    "apply",
    "apply changes",
    "confirm apply",
    "confirm changes",
    "confirm apply changes",
    "proceed",
    "do it",
}

_CANCEL_MESSAGES = {"cancel", "stop", "never mind", "nevermind"}
_MOVE_LINE_RE = re.compile(r"^\s*(?:[-*]\s*)?(?P<key>.+?)\s*(?:->|\u2192)\s*(?P<folder>.+?)\s*$")


def _history_messages(history: list[dict] | None) -> list[BaseMessage]:
    messages: list[BaseMessage] = []
    total_chars = 0
    bounded_history = (history or [])[-settings.max_chat_history_messages:]
    for item in reversed(bounded_history):
        role = item.get("role")
        content = item.get("content")
        if not isinstance(content, str) or not content:
            continue
        total_chars += len(content)
        if total_chars > settings.max_chat_history_chars:
            break
        if role == "user":
            messages.append(HumanMessage(content=content))
        elif role == "assistant":
            messages.append(AIMessage(content=content))
    return list(reversed(messages))


def _normalized_command(message: str) -> str:
    return " ".join(message.strip().lower().split())


def _is_confirmation(message: str) -> bool:
    return _normalized_command(message) in _CONFIRMATION_MESSAGES


def _is_cancel(message: str) -> bool:
    return _normalized_command(message) in _CANCEL_MESSAGES


def _strip_markup(value: str) -> str:
    value = value.strip()
    value = value.strip("`")
    value = re.sub(r"^\d+[.)]\s*", "", value)
    return value.strip()


def _extract_pending_moves(history: list[dict] | None) -> list[dict]:
    for item in reversed(history or []):
        if item.get("role") != "assistant":
            continue
        content = item.get("content", "")
        if not isinstance(content, str):
            continue

        moves: list[dict] = []
        for line in content.splitlines():
            match = _MOVE_LINE_RE.match(line)
            if not match:
                continue
            key = _strip_markup(match.group("key"))
            folder = _strip_markup(match.group("folder"))
            folder = folder.split(" (", 1)[0].strip()
            folder = folder.rstrip("/")
            if key and folder:
                moves.append({"key": key, "destination_folder": folder})
        if moves:
            return moves
    return []


async def _apply_confirmed_plan(
    message: str,
    user_id: str,
    history: list[dict] | None,
) -> AsyncGenerator[dict, None]:
    if _is_cancel(message):
        yield {"token": "Cancelled. No files were moved."}
        yield {"done": True}
        return

    if not _is_confirmation(message):
        return

    moves = _extract_pending_moves(history)
    if not moves:
        yield {
            "token": (
                "I do not see a pending organization plan to apply. "
                "Ask me to organize the files again and I will propose a new plan."
            )
        }
        yield {"done": True}
        return

    yield {
        "tool": {
            "name": "organize_files",
            "status": "started",
            "input": {"moves": moves},
        }
    }
    try:
        result = await file_tools.organize_files(user_id=user_id, moves=moves)
    except Exception as exc:
        yield {
            "tool": {
                "name": "organize_files",
                "status": "failed",
                "error": str(exc),
            }
        }
        yield {"token": f"I could not apply the organization changes: {exc}"}
        yield {"done": True}
        return

    yield {
        "tool": {
            "name": "organize_files",
            "status": "completed",
            "result": result.content,
        }
    }
    completed = result.content.get("moves", [])
    moved = [
        f"{item['key']} -> {item['destination_key']}"
        for item in completed
        if item.get("status") == "moved"
    ]
    unchanged = [
        item["key"]
        for item in completed
        if item.get("status") == "unchanged"
    ]
    lines = ["Applied the file organization changes."]
    if moved:
        lines.append("Moved:")
        lines.extend(f"- {item}" for item in moved)
    if unchanged:
        lines.append("Already in place:")
        lines.extend(f"- {item}" for item in unchanged)
    yield {"token": "\n".join(lines)}
    yield {"done": True}


def _build_langchain_tools(
    user_id: str,
    default_scope: str,
    allow_mutations: bool,
    state: AgentRunState,
):
    async def search_files(query: str, scope: str = "all") -> dict:
        result = await file_tools.search_files(
            query=query,
            scope=scope or default_scope or "all",
            user_id=user_id,
        )
        state.sources.extend(result.sources or [])
        return result.content

    async def list_files(folder: str = "") -> dict:
        result = await file_tools.list_files(user_id=user_id, folder=folder)
        return result.content

    async def organize_files(moves: list[FileMoveInput]) -> dict:
        normalized = [
            {"key": move.key, "destination_folder": move.destination_folder}
            if isinstance(move, FileMoveInput)
            else move
            for move in moves
        ]
        if not allow_mutations:
            return {
                "requires_confirmation": True,
                "moves": normalized,
                "message": "File organization requires confirmation before changes are applied.",
            }
        result = await file_tools.organize_files(user_id=user_id, moves=normalized)
        return result.content

    return [
        StructuredTool.from_function(
            coroutine=search_files,
            name="search_files",
            description="Retrieve relevant chunks from the user's indexed uploaded files.",
            args_schema=SearchFilesInput,
        ),
        StructuredTool.from_function(
            coroutine=list_files,
            name="list_files",
            description="List the user's uploaded files, optionally within a folder.",
            args_schema=ListFilesInput,
        ),
        StructuredTool.from_function(
            coroutine=organize_files,
            name="organize_files",
            description="Move uploaded files into destination folders.",
            args_schema=OrganizeFilesInput,
        ),
    ]


def _build_agent(
    user_id: str,
    scope: str,
    allow_mutations: bool,
    history: list[dict] | None,
):
    state = AgentRunState()
    agent_tools = _build_langchain_tools(
        user_id=user_id,
        default_scope=scope,
        allow_mutations=allow_mutations,
        state=state,
    )
    agent = create_agent(
        model=llm_client.langchain_chat(),
        tools=agent_tools,
        system_prompt=_AGENT_SYSTEM_PROMPT,
        checkpointer=InMemorySaver(),
    )

    session_id = str(uuid4())
    return agent, state, session_id, _history_messages(history)

async def query(
    question: str,
    scope: str,
    user_id: str,
) -> AsyncGenerator[dict, None]:
    vector = await embedder.embed(question)
    chunks = await vector_store.search(
        vector=vector,
        query_text=question,
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


async def chat(
    message: str,
    scope: str,
    user_id: str,
    allow_mutations: bool = False,
    history: list[dict] | None = None,
) -> AsyncGenerator[dict, None]:
    if not allow_mutations and (_is_confirmation(message) or _is_cancel(message)):
        handled = False
        async for event in _apply_confirmed_plan(
            message=message,
            user_id=user_id,
            history=history,
        ):
            handled = True
            yield event
        if handled:
            return

    agent, state, session_id, memory = _build_agent(
        user_id=user_id,
        scope=scope,
        allow_mutations=allow_mutations,
        history=history,
    )
    emitted_source_count = 0

    async for event in agent.astream_events(
        {"messages": [*memory, HumanMessage(content=message)]},
        config={
            "configurable": {"thread_id": session_id},
            "recursion_limit": 12,
        },
        version="v2",
    ):
        event_name = event.get("event")
        name = event.get("name", "")
        data = event.get("data", {})

        if event_name == "on_tool_start":
            yield {
                "tool": {
                    "name": name,
                    "status": "started",
                    "input": data.get("input"),
                }
            }
        elif event_name == "on_tool_end":
            yield {
                "tool": {
                    "name": name,
                    "status": "completed",
                    "result": data.get("output"),
                }
            }
            for source in state.sources[emitted_source_count:]:
                yield {"source": source}
            emitted_source_count = len(state.sources)
        elif event_name == "on_tool_error":
            yield {
                "tool": {
                    "name": name,
                    "status": "failed",
                    "error": str(data.get("error", "")),
                }
            }
        elif event_name == "on_chat_model_stream":
            chunk = data.get("chunk")
            content = getattr(chunk, "content", "")
            if isinstance(content, str) and content:
                yield {"token": content}

    yield {"done": True}
