export interface FileInfo {
  id: string;
  path: string;
  package: string;
  lines: number;
}

export interface Symbol {
  id: string;
  name: string;
  kind: string;
  receiver: string;
  defLine: number;
  defCol: number;
  file: FileInfo;
}

export interface Ref {
  id: string;
  refLine: number;
  refCol: number;
  file: FileInfo;
}

export interface Related {
  score: number;
  symbol: Symbol;
}

const ENDPOINT = import.meta.env.VITE_API_URL ?? "http://localhost:8080/graphql";

async function query<T>(q: string, variables: Record<string, unknown>): Promise<T> {
  const resp = await fetch(ENDPOINT, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query: q, variables }),
  });
  const body = await resp.json();
  if (body.errors) {
    throw new Error(body.errors[0]?.message ?? "graphql error");
  }
  return body.data as T;
}

const SYMBOL_FIELDS = `
  id name kind receiver defLine defCol
  file { id path package lines }
`;

export function lookupSymbol(name: string): Promise<{ symbol: Symbol | null }> {
  return query(`query($name: String!) { symbol(name: $name) { ${SYMBOL_FIELDS} } }`, { name });
}

export function findReferences(symbolId: string): Promise<{ references: Ref[] }> {
  return query(
    `query($id: ID!) { references(symbolId: $id) { id refLine refCol file { id path package lines } } }`,
    { id: symbolId },
  );
}

export function findCallers(symbolId: string, depth: number): Promise<{ callers: Symbol[] }> {
  return query(
    `query($id: ID!, $d: Int) { callers(symbolId: $id, depth: $d) { ${SYMBOL_FIELDS} } }`,
    { id: symbolId, d: depth },
  );
}

export function findRelated(symbolId: string, limit: number): Promise<{ related: Related[] }> {
  return query(
    `query($id: ID!, $l: Int) { related(symbolId: $id, limit: $l) { score symbol { ${SYMBOL_FIELDS} } } }`,
    { id: symbolId, l: limit },
  );
}
