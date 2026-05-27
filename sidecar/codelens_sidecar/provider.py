"""Provider seam for related-symbol ranking.

The sidecar ranks candidate symbols against a target. The actual scoring lives
behind a ``SemanticProvider`` protocol so an embedding backend can be swapped in
later without touching the HTTP layer. The default ``FakeProvider`` is fully
deterministic and offline, which keeps CI hermetic.
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Protocol


@dataclass(frozen=True)
class Symbol:
    id: int
    name: str
    kind: str


@dataclass(frozen=True)
class Scored:
    id: int
    score: float


class SemanticProvider(Protocol):
    def rank(self, target: Symbol, candidates: list[Symbol], limit: int) -> list[Scored]:
        ...


_CAMEL = re.compile(r"[A-Z]?[a-z0-9]+|[A-Z]+(?![a-z])")


def tokens(name: str) -> set[str]:
    parts: list[str] = []
    for chunk in name.replace("-", "_").split("_"):
        parts.extend(m.group(0) for m in _CAMEL.finditer(chunk))
    return {p.lower() for p in parts if p}


def _round(value: float) -> float:
    return round(value, 3)


class FakeProvider:
    """Deterministic token-overlap ranking used in tests and offline CI.

    Score is the Jaccard overlap of identifier tokens plus a small bonus when
    the kinds match. The same input always produces the same ordering, with
    ties broken by symbol id.
    """

    def rank(self, target: Symbol, candidates: list[Symbol], limit: int) -> list[Scored]:
        if limit <= 0:
            limit = 5
        target_tokens = tokens(target.name)
        scored: list[Scored] = []
        for cand in candidates:
            if cand.id == target.id:
                continue
            cand_tokens = tokens(cand.name)
            if not target_tokens or not cand_tokens:
                continue
            inter = len(target_tokens & cand_tokens)
            if inter == 0:
                continue
            union = len(target_tokens | cand_tokens)
            score = inter / union
            if cand.kind == target.kind:
                score += 0.1
            scored.append(Scored(id=cand.id, score=_round(score)))
        scored.sort(key=lambda s: (-s.score, s.id))
        return scored[:limit]


def build_provider(name: str) -> SemanticProvider:
    """Return a provider by name. Only the offline fake is wired in."""
    if name in ("", "fake"):
        return FakeProvider()
    raise ValueError(f"unknown provider: {name}")
