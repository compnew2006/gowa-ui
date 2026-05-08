"""CocoIndex CodeEmbedding flow for indexing Go and TypeScript code."""

from __future__ import annotations

import hashlib
import pathlib
from dataclasses import dataclass
from typing import Iterator

import numpy as np
from numpy.typing import NDArray

import cocoindex as coco
from cocoindex.connectors import localfs
from cocoindex.connectors.sqlite import (
    managed_connection,
    mount_table_target,
)
from cocoindex.ops.sentence_transformers import SentenceTransformerEmbedder
from cocoindex.ops.text import RecursiveSplitter, detect_code_language
from cocoindex.resources.file import PatternFilePathMatcher

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

EMBEDDING_MODEL = "sentence-transformers/all-MiniLM-L6-v2"
CHUNK_SIZE = 1200
CHUNK_OVERLAP = 200

SQLITE_CONN = coco.ContextKey["ManagedConnection"]("sqlite_conn")
EMBEDDER_KEY = coco.ContextKey[SentenceTransformerEmbedder]("embedder")
SPLITTER_KEY = coco.ContextKey[RecursiveSplitter]("splitter")

GO_PATTERNS = ["**/*.go"]
TS_PATTERNS = ["**/*.ts", "**/*.tsx", "**/*.vue"]

EXCLUDE_PATTERNS = [
    "**/node_modules/**",
    "**/vendor/**",
    "**/dist/**",
    "**/__pycache__/**",
    "**/.*",
]

SOURCE_DIRS = [
    ("internal", GO_PATTERNS),
    ("cmd", GO_PATTERNS),
    ("frontend/src", TS_PATTERNS),
]


# ---------------------------------------------------------------------------
# Row schema — regular table with BLOB embedding column
# ---------------------------------------------------------------------------


@dataclass
class CodeChunk:
    id: int
    file_path: str
    language: str
    chunk_index: int
    content: str
    embedding: NDArray[np.float32]


def _stable_id(file_path: str, chunk_index: int) -> int:
    """Generate a deterministic integer ID from file path and chunk index."""
    h = hashlib.sha256(f"{file_path}:{chunk_index}".encode()).digest()
    return int.from_bytes(h[:8], "big") & 0x7FFFFFFFFFFFFFFF


# ---------------------------------------------------------------------------
# Lifespan
# ---------------------------------------------------------------------------


@coco.lifespan
def coco_lifespan(builder: coco.EnvironmentBuilder) -> Iterator[None]:
    """Configure the CocoIndex environment."""
    builder.settings.db_path = pathlib.Path("./cocoindex.db")

    db_path = pathlib.Path("./code_embeddings.db")
    builder.provide_with(
        SQLITE_CONN,
        managed_connection(db_path, load_vec="auto"),
    )
    builder.provide(EMBEDDER_KEY, SentenceTransformerEmbedder(EMBEDDING_MODEL))
    builder.provide(SPLITTER_KEY, RecursiveSplitter())
    yield


# ---------------------------------------------------------------------------
# Processing function
# ---------------------------------------------------------------------------


@coco.fn(memo=True, version=1)
async def embed_chunk(
    embedder: SentenceTransformerEmbedder,
    text: str,
) -> NDArray[np.float32]:
    """Embed a single text chunk."""
    return await embedder.embed(text)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


async def _index_directory(
    table: coco.connectors.sqlite.TableTarget,
    dir_path: pathlib.Path,
    include_patterns: list[str],
) -> None:
    """Walk a single directory, chunk code, embed, and store."""
    embedder = coco.use_context(EMBEDDER_KEY)
    splitter = coco.use_context(SPLITTER_KEY)

    files = localfs.walk_dir(
        dir_path,
        recursive=True,
        path_matcher=PatternFilePathMatcher(
            included_patterns=include_patterns,
            excluded_patterns=EXCLUDE_PATTERNS,
        ),
    )

    async for relative_path, file in files.items():
        content_bytes = await file.read()
        try:
            text = content_bytes.decode("utf-8")
        except UnicodeDecodeError:
            continue

        if not text.strip():
            continue

        language = detect_code_language(filename=str(file.file_path.path.name))
        lang_name = language if language else "unknown"

        chunks = splitter.split(
            text,
            chunk_size=CHUNK_SIZE,
            chunk_overlap=CHUNK_OVERLAP,
            language=lang_name,
        )

        for idx, chunk in enumerate(chunks):
            full_path = f"{dir_path.name}/{relative_path}"
            embedding = await embed_chunk(embedder, chunk.text)

            table.declare_row(
                row=CodeChunk(
                    id=_stable_id(full_path, idx),
                    file_path=full_path,
                    language=lang_name,
                    chunk_index=idx,
                    content=chunk.text,
                    embedding=embedding,
                )
            )


# ---------------------------------------------------------------------------
# Main pipeline
# ---------------------------------------------------------------------------


@coco.fn
async def app_main() -> None:
    """Walk project directories, chunk code, embed, and store in vector index."""

    embedder = coco.use_context(EMBEDDER_KEY)

    schema = await coco.connectors.sqlite.TableSchema.from_class(
        CodeChunk,
        primary_key=["id"],
        column_overrides={"embedding": embedder},
    )

    table = await mount_table_target(
        SQLITE_CONN,
        "code_embeddings",
        schema,
    )

    for dir_name, patterns in SOURCE_DIRS:
        dir_path = pathlib.Path(dir_name)
        if dir_path.is_dir():
            await _index_directory(table, dir_path, patterns)


app = coco.App(
    coco.AppConfig(name="whatomate"),
    app_main,
)
