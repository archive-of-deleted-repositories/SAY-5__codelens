import { useCallback, useState } from "react";
import {
  findReferences,
  findRelated,
  lookupSymbol,
  type Ref,
  type Related,
  type Symbol,
} from "./api";
import { RelatedPanel } from "./components/RelatedPanel";
import { SymbolPanel } from "./components/SymbolPanel";

export function App() {
  const [name, setName] = useState("Add");
  const [symbol, setSymbol] = useState<Symbol | null>(null);
  const [refs, setRefs] = useState<Ref[]>([]);
  const [related, setRelated] = useState<Related[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const search = useCallback(async (lookup: string) => {
    setError("");
    setLoading(true);
    try {
      const { symbol: found } = await lookupSymbol(lookup);
      if (!found) {
        setSymbol(null);
        setRefs([]);
        setRelated([]);
        setError(`no symbol named "${lookup}"`);
        return;
      }
      setSymbol(found);
      const [{ references }, { related: rel }] = await Promise.all([
        findReferences(found.id),
        findRelated(found.id, 5),
      ]);
      setRefs(references);
      setRelated(rel);
    } catch (e) {
      setError(e instanceof Error ? e.message : "request failed");
    } finally {
      setLoading(false);
    }
  }, []);

  return (
    <main className="app">
      <h1>codelens</h1>
      <p className="lede">
        Browse a parsed source tree: look up a symbol, jump to its definition,
        and find every reference across files.
      </p>
      <form
        className="search"
        onSubmit={(e) => {
          e.preventDefault();
          void search(name);
        }}
      >
        <input
          aria-label="symbol name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="symbol name (e.g. Add)"
        />
        <button type="submit" disabled={loading}>
          {loading ? "searching" : "find"}
        </button>
      </form>
      {error && <p className="error">{error}</p>}
      {symbol && <SymbolPanel symbol={symbol} refs={refs} />}
      {symbol && <RelatedPanel related={related} />}
    </main>
  );
}
