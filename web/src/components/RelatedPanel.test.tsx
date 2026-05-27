import { render, screen } from "@testing-library/react";
import { RelatedPanel } from "./RelatedPanel";
import type { Related } from "../api";

const file = { id: "1", path: "ledger/ledger.go", package: "ledger", lines: 50 };

function related(name: string, score: number, id: string): Related {
  return {
    score,
    symbol: { id, name, kind: "func", receiver: "", defLine: 1, defCol: 1, file },
  };
}

test("lists related symbols with scores in order", () => {
  const items = [related("Balance", 0.5, "2"), related("Report", 0.333, "3")];
  render(<RelatedPanel related={items} />);
  expect(screen.getByText("Balance")).toBeDefined();
  expect(screen.getByText("0.500")).toBeDefined();
  expect(screen.getByText("Report")).toBeDefined();
});

test("shows an empty state when there are no related symbols", () => {
  render(<RelatedPanel related={[]} />);
  expect(screen.getByText("no related symbols")).toBeDefined();
});
