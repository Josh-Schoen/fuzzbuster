import { TopNav } from "@/components/TopNav";

type ResourceDTO = {
  id: string;
  kind: string;
  name: string;
  description: string;
  languages: string[];
  url?: string;
  phone?: string;
  city?: string;
  region?: string;
};

async function fetchResources(): Promise<ResourceDTO[]> {
  const base = process.env.NEXT_PUBLIC_API_BASE ?? "http://localhost:8080";
  const res = await fetch(`${base}/v1/resources?region=MN`, { cache: "no-store" });
  if (!res.ok) return [];
  return res.json();
}

export default async function ResourcesPage() {
  const items = await fetchResources();
  return (
    <>
      <TopNav />
      <main>
        <h1>Resources</h1>
        {items.length === 0 && (
          <p>No resources are listed yet. Outreach to ILCM, The Advocates for Human Rights, Mid-Minnesota Legal Aid, Unidos MN, COPAL, Navigate MN, CLUES, and Twin Cities Rapid Response Network is in progress.</p>
        )}
        <ul>
          {items.map((r) => (
            <li key={r.id}>
              <strong>{r.name}</strong> — {r.kind}
              {r.city && <> · {r.city}, {r.region}</>}
              {r.phone && <> · <a href={`tel:${r.phone}`}>{r.phone}</a></>}
              {r.url && <> · <a href={r.url} rel="noopener noreferrer" target="_blank">website</a></>}
              <div style={{ color: "var(--muted)" }}>{r.description}</div>
              {r.languages?.length > 0 && (
                <div style={{ color: "var(--muted)" }}>Languages: {r.languages.join(", ")}</div>
              )}
            </li>
          ))}
        </ul>
      </main>
    </>
  );
}
