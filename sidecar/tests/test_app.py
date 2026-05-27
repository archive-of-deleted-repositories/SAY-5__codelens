from fastapi.testclient import TestClient

from codelens_sidecar.app import create_app


def client():
    return TestClient(create_app())


def test_healthz():
    resp = client().get("/healthz")
    assert resp.status_code == 200
    assert resp.json()["status"] == "ok"


def test_rank_endpoint():
    body = {
        "target": {"id": 1, "name": "ReadFile", "kind": "func"},
        "candidates": [
            {"id": 2, "name": "WriteFile", "kind": "func"},
            {"id": 3, "name": "Balance", "kind": "func"},
        ],
        "limit": 5,
    }
    resp = client().post("/rank", json=body)
    assert resp.status_code == 200
    ids = [r["id"] for r in resp.json()["results"]]
    assert ids == [2]
