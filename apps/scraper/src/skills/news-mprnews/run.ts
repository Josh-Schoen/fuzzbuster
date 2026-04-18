// Worked-example Skill: drive a headless browser to MPR News, find recent
// MN immigration coverage, and return RawItems for the orchestrator.
//
// The browser is driven by Stagehand (Playwright + LLM); the agent loop is
// owned by the Claude Agent SDK, which loads SKILL.md as the system context.
// In v0 we wire just enough to demonstrate the shape; the actual prompt
// engineering and fixture-based test land in week 3-4.

import { Stagehand } from "@browserbasehq/stagehand";
import { z } from "zod";

const ArticleSchema = z.object({
  url: z.string().url(),
  headline: z.string().min(4),
  published_at: z.string().optional(),
  summary: z.string().min(20).max(500),
  candidate_city: z.string().optional(),
});
type Article = z.infer<typeof ArticleSchema>;

type ScrapeJob = {
  id: string;
  producer: string;
  params: Record<string, string>;
  region: string;
};

export async function runMprNews(job: ScrapeJob) {
  if (job.region !== "US-MN") {
    return {
      items: [],
      inputTokens: 0,
      outputTokens: 0,
      modelUsed: "",
      error: `news-mprnews supports US-MN only, got ${job.region}`,
    };
  }

  const lookbackHours = Number(job.params.lookback_hours ?? "24");
  const stagehand = new Stagehand({
    env: "LOCAL",
    headless: true,
    modelName: "claude-haiku-4-5",
  });
  await stagehand.init();

  let items: Article[] = [];
  try {
    const page = stagehand.page;
    // Step 1: respect robots.txt before doing anything else.
    const robotsRes = await page.request.get("https://www.mprnews.org/robots.txt");
    const robotsTxt = await robotsRes.text();
    if (/Disallow:\s*\/search/i.test(robotsTxt)) {
      return {
        items: [],
        inputTokens: 0,
        outputTokens: 0,
        modelUsed: "claude-haiku-4-5",
        error: "robots_disallow",
      };
    }

    // Step 2: search.
    await page.goto("https://www.mprnews.org/search?q=ICE+immigration", {
      waitUntil: "domcontentloaded",
    });

    // Step 3: ask the agent to extract candidate articles in one structured call.
    const extracted = await stagehand.extract({
      instruction:
        "Find news articles published in the last " +
        lookbackHours +
        " hours about ICE, immigration enforcement, detention, or deportation in Minnesota. " +
        "For each article return its canonical URL, headline, published_at (ISO if visible), a 60-word summary " +
        "describing the situation (NOT naming or quoting any private individual), and the Minnesota city or county " +
        "if mentioned. Do NOT include any name, photo, badge number, or plate number of ICE personnel.",
      schema: z.object({ articles: z.array(ArticleSchema) }),
    });

    items = extracted.articles ?? [];
  } catch (err) {
    return {
      items: [],
      inputTokens: 0,
      outputTokens: 0,
      modelUsed: "claude-haiku-4-5",
      error: String(err),
    };
  } finally {
    await stagehand.close();
  }

  return {
    items: items.map((a) => ({
      id: cryptoRandomId(),
      source: {
        kind: "KIND_BROWSER_AGENT",
        url: a.url,
        producer: "news-mprnews",
        fetchedAt: nowProto(),
      },
      text: `${a.headline}\n\n${a.summary}`,
      metadata: {
        headline: a.headline,
        published_at: a.published_at ?? "",
        candidate_city: a.candidate_city ?? "",
      },
      capturedAt: nowProto(),
      language: "LANGUAGE_EN",
    })),
    inputTokens: 0,   // Stagehand exposes these on its result; wire in week 3.
    outputTokens: 0,
    modelUsed: "claude-haiku-4-5",
    error: "",
  };
}

function cryptoRandomId(): string {
  return crypto.randomUUID();
}

function nowProto(): { seconds: string; nanos: number } {
  const ms = Date.now();
  return { seconds: String(Math.floor(ms / 1000)), nanos: (ms % 1000) * 1_000_000 };
}
