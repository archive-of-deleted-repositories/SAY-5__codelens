import type { Ref, Symbol } from "../api";

export function SymbolPanel({ symbol, refs }: { symbol: Symbol; refs: Ref[] }) {
  return (
    <section className="panel">
      <header className="def">
        <span className="kind">{symbol.kind}</span>
        <span className="sym-name">
          {symbol.receiver && `(${symbol.receiver}) `}
          {symbol.name}
        </span>
        <span className="loc">
          {symbol.file.path}:{symbol.defLine}:{symbol.defCol}
        </span>
      </header>
      <h2>
        references <span className="count">{refs.length}</span>
      </h2>
      <ul className="refs">
        {refs.map((ref) => (
          <li key={ref.id}>
            <code>
              {ref.file.path}:{ref.refLine}:{ref.refCol}
            </code>
          </li>
        ))}
        {refs.length === 0 && <li className="empty">no references</li>}
      </ul>
    </section>
  );
}
