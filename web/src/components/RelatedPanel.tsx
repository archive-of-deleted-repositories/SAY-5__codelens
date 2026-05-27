import type { Related } from "../api";

export function RelatedPanel({ related }: { related: Related[] }) {
  return (
    <section className="panel related">
      <h2>related symbols</h2>
      <ul className="related-list">
        {related.map((r) => (
          <li key={r.symbol.id}>
            <span className="sym-name">{r.symbol.name}</span>
            <span className="loc">{r.symbol.file.path}</span>
            <span className="score">{r.score.toFixed(3)}</span>
          </li>
        ))}
        {related.length === 0 && <li className="empty">no related symbols</li>}
      </ul>
    </section>
  );
}
