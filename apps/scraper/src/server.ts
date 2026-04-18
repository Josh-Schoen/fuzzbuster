// gRPC server that exposes ScraperService.RunJob (proto/fuzzbuster/v1/scrape.proto).
// The Go orchestrator is the client; this process owns the headless Chromium
// pool and the Claude-Agent-SDK + Stagehand glue.
//
// v0 skeleton: dispatches to whichever Skill matches `producer`. Today we
// only ship the news-mprnews Skill as a worked example; the rest land in
// week 3-4 (see docs/data-sources.md).

import * as grpc from "@grpc/grpc-js";
import * as protoLoader from "@grpc/proto-loader";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { runMprNews } from "./skills/news-mprnews/run.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const PROTO_PATH = path.resolve(
  __dirname,
  "../../../proto/fuzzbuster/v1/scrape.proto",
);

const packageDef = protoLoader.loadSync(PROTO_PATH, {
  keepCase: false,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
  includeDirs: [path.resolve(__dirname, "../../../proto")],
});

const proto = grpc.loadPackageDefinition(packageDef) as any;
const ScraperService = proto.fuzzbuster.v1.ScraperService;

type ScrapeJob = {
  id: string;
  producer: string;
  params: Record<string, string>;
  scheduledAt?: { seconds: string; nanos: number };
  region: string;
};

async function runJobImpl(
  call: grpc.ServerUnaryCall<ScrapeJob, any>,
  callback: grpc.sendUnaryData<any>,
) {
  const job = call.request;
  console.log(JSON.stringify({ msg: "run_job", id: job.id, producer: job.producer, region: job.region }));

  try {
    let result;
    switch (job.producer) {
      case "news-mprnews":
        result = await runMprNews(job);
        break;
      // TODO(week-3-4): bluesky-search, news-startribune, news-sahan,
      // news-minnpost, news-mnreformer, news-local-tv, ice-field-office-stpaul,
      // mn-sheriff-press, reddit-subreddit-watch, mastodon-tag-search.
      default:
        callback(null, {
          jobId: job.id,
          items: [],
          inputTokens: 0,
          outputTokens: 0,
          modelUsed: "",
          error: `unknown producer: ${job.producer}`,
        });
        return;
    }
    callback(null, { jobId: job.id, ...result });
  } catch (err) {
    callback(null, {
      jobId: job.id,
      items: [],
      inputTokens: 0,
      outputTokens: 0,
      modelUsed: "",
      error: String(err),
    });
  }
}

const server = new grpc.Server();
server.addService(ScraperService.service, { RunJob: runJobImpl });

const addr = process.env.SCRAPER_GRPC_ADDR ?? "0.0.0.0:50051";
server.bindAsync(addr, grpc.ServerCredentials.createInsecure(), (err, port) => {
  if (err) {
    console.error("bind failed", err);
    process.exit(1);
  }
  console.log(JSON.stringify({ msg: "scraper listening", addr, port }));
});
