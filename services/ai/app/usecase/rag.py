from collections.abc import AsyncGenerator
from dataclasses import dataclass, field
from uuid import uuid4

from langchain.agents import AgentExecutor, create_openai_tools_agent
from langchain_core.chat_history import InMemoryChatMessageHistory
from langchain_core.messages import AIMessage, BaseMessage, HumanMessage
from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder
from langchain_core.runnables.history import RunnableWithMessageHistory
from langchain_core.tools import StructuredTool
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


def _history_messages(history: list[dict] | None) -> list[BaseMessage]:
    messages: list[BaseMessage] = []
    for item in history or []:
        role = item.get("role")
        content = item.get("content")
        if not isinstance(content, str) or not content:
            continue
        if role == "user":
            messages.append(HumanMessage(content=content))
        elif role == "assistant":
            messages.append(AIMessage(content=content))
    return messages


def _build_langchain_tools(user_id: str, default_scope: str, state: AgentRunState):
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


def _build_agent(user_id: str, scope: str, history: list[dict] | None):
    state = AgentRunState()
    agent_tools = _build_langchain_tools(user_id=user_id, default_scope=scope, state=state)
    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", _AGENT_SYSTEM_PROMPT),
            MessagesPlaceholder("chat_history", optional=True),
            ("human", "{input}"),
            MessagesPlaceholder("agent_scratchpad"),
        ]
    )
    agent = create_openai_tools_agent(
        llm_client.langchain_chat(),
        agent_tools,
        prompt,
    )
    executor = AgentExecutor(
        agent=agent,
        tools=agent_tools,
        max_iterations=6,
        return_intermediate_steps=True,
        handle_parsing_errors=True,
    )

    session_id = str(uuid4())
    histories = {
        session_id: InMemoryChatMessageHistory(messages=_history_messages(history))
    }

    def get_session_history(session: str) -> InMemoryChatMessageHistory:
        return histories.setdefault(session, InMemoryChatMessageHistory())

    runnable = RunnableWithMessageHistory(
        executor,
        get_session_history,
        input_messages_key="input",
        history_messages_key="chat_history",
        output_messages_key="output",
    )
    return runnable, state, session_id

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
    history: list[dict] | None = None,
) -> AsyncGenerator[dict, None]:
    agent, state, session_id = _build_agent(user_id=user_id, scope=scope, history=history)
    result = await agent.ainvoke(
        {"input": message},
        config={"configurable": {"session_id": session_id}},
    )

    for action, observation in result.get("intermediate_steps", []):
        yield {
            "tool": {
                "name": action.tool,
                "status": "completed",
                "result": observation,
            }
        }
    for source in state.sources:
        yield {"source": source}

    yield {"token": result.get("output", "")}
    yield {"done": True}
