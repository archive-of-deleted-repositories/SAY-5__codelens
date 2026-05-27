from codelens_sidecar.provider import FakeProvider, Symbol, tokens


def test_tokens_splits_camel_and_snake():
    assert tokens("ReadFile") == {"read", "file"}
    assert tokens("symbol_id") == {"symbol", "id"}


def test_rank_prefers_shared_tokens():
    target = Symbol(1, "ReadFile", "func")
    candidates = [
        Symbol(2, "WriteFile", "func"),
        Symbol(3, "ReadConfig", "func"),
        Symbol(4, "Balance", "func"),
    ]
    ranked = FakeProvider().rank(target, candidates, limit=5)
    ids = [r.id for r in ranked]
    assert 4 not in ids  # shares no token
    assert set(ids) == {2, 3}


def test_rank_is_deterministic():
    target = Symbol(10, "BalanceTotal", "func")
    candidates = [
        Symbol(11, "Total", "func"),
        Symbol(12, "Balance", "func"),
        Symbol(13, "BalanceReport", "func"),
    ]
    first = FakeProvider().rank(target, candidates, limit=5)
    second = FakeProvider().rank(target, candidates, limit=5)
    assert [(r.id, r.score) for r in first] == [(r.id, r.score) for r in second]
