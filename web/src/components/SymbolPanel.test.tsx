import { render, screen } from "@testing-library/react";
import { SymbolPanel } from "./SymbolPanel";
import type { Ref, Symbol } from "../api";

const file = { id: "1", path: "calc/calc.go", package: "calc", lines: 40 };

const symbol: Symbol = {
  id: "7",
  name: "Add",
  kind: "method",
  receiver: "Op",
  defLine: 10,
  defCol: 9,
  file,
};

test("shows the definition location and reference list", () => {
  const refs: Ref[] = [
    { id: "1", refLine: 24, refCol: 12, file },
    { id: "2", refLine: 31, refCol: 8, file },
  ];
  render(<SymbolPanel symbol={symbol} refs={refs} />);
  expect(screen.getByText(/Add/)).toBeDefined();
  expect(screen.getByText("calc/calc.go:10:9")).toBeDefined();
  expect(screen.getByText("2")).toBeDefined();
  expect(screen.getByText("calc/calc.go:24:12")).toBeDefined();
});

test("shows an empty state with no references", () => {
  render(<SymbolPanel symbol={symbol} refs={[]} />);
  expect(screen.getByText("no references")).toBeDefined();
});
