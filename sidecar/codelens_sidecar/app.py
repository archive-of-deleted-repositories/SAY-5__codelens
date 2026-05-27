"""HTTP surface for the semantic sidecar."""

from __future__ import annotations

import os

from fastapi import FastAPI
from pydantic import BaseModel

from .provider import Symbol, build_provider


class SymbolIn(BaseModel):
    id: int
    name: str
    kind: str


class RankRequest(BaseModel):
    target: SymbolIn
    candidates: list[SymbolIn]
    limit: int = 5


class ScoredOut(BaseModel):
    id: int
    score: float


class RankResponse(BaseModel):
    results: list[ScoredOut]


def create_app() -> FastAPI:
    provider = build_provider(os.getenv("SEMANTIC_PROVIDER", "fake"))
    app = FastAPI(title="codelens-sidecar")

    @app.get("/healthz")
    def healthz() -> dict[str, str]:
        return {"status": "ok"}

    @app.post("/rank", response_model=RankResponse)
    def rank(req: RankRequest) -> RankResponse:
        target = Symbol(req.target.id, req.target.name, req.target.kind)
        cands = [Symbol(c.id, c.name, c.kind) for c in req.candidates]
        results = provider.rank(target, cands, req.limit)
        return RankResponse(results=[ScoredOut(id=r.id, score=r.score) for r in results])

    return app


app = create_app()
