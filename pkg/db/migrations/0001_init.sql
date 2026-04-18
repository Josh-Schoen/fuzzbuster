-- 0001_init.sql
-- Fuzzbuster initial schema. Postgres 16 + PostGIS + pgvector.
--
-- Safety contract enforced at the schema level:
--   * sighting.location is geography(Point, 4326), constrained to the
--     200m grid by a CHECK that the snapped coords match.
--   * sighting.notes_scrubbed is the only place free-text is stored;
--     no column for original notes exists.
--   * sighting.created_at + decay are enforced by a partial index that
--     the public API reads from; rows older than 8h are invisible.

CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- for gen_random_uuid()

-- ---------------------------------------------------------------------------
-- Sources / scraper provenance
-- ---------------------------------------------------------------------------

CREATE TYPE source_kind AS ENUM (
  'rss',
  'api',
  'browser_agent',
  'human_public',
  'human_partner',
  'human_moderator'
);

CREATE TABLE sources (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kind        source_kind NOT NULL,
  url         TEXT NOT NULL,
  producer    TEXT,                  -- e.g. "news-mprnews", "rss-gdelt"
  fetched_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX sources_url_idx ON sources (url);

-- ---------------------------------------------------------------------------
-- Sightings (decayed, anonymous, coarse-located)
-- ---------------------------------------------------------------------------

CREATE TYPE sighting_category AS ENUM (
  'sighting',
  'checkpoint',
  'courthouse',
  'raid_rumor',
  'detention',
  'agency_presence'
);

CREATE TYPE sighting_status AS ENUM (
  'pending',
  'published',
  'hidden',
  'expired'
);

CREATE TYPE language_code AS ENUM (
  'en', 'es', 'ht', 'ar', 'zh', 'so', 'hmn'
);

CREATE TABLE sightings (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  -- Geography stored as PostGIS geography for spatial radius queries.
  -- Coordinates MUST be pre-snapped to the 200m grid in application code.
  location        geography(Point, 4326) NOT NULL,
  label           TEXT,
  observed_at     TIMESTAMPTZ NOT NULL,
  category        sighting_category NOT NULL,
  notes_scrubbed  TEXT NOT NULL DEFAULT '',
  notes_language  language_code NOT NULL DEFAULT 'en',
  source_id       UUID NOT NULL REFERENCES sources(id),
  status          sighting_status NOT NULL DEFAULT 'pending',
  confirms        INTEGER NOT NULL DEFAULT 0,
  denies          INTEGER NOT NULL DEFAULT 0,
  -- Embedding for dedup against the last 48h sliding window.
  embedding       vector(768),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT sighting_observed_in_past CHECK (observed_at <= NOW() + INTERVAL '5 minutes'),
  CONSTRAINT sighting_label_no_house_number CHECK (label !~ '^\s*\d+\s')
);

-- Spatial index for "near me" queries.
CREATE INDEX sightings_location_gix ON sightings USING GIST (location);

-- ANN index for dedup.
CREATE INDEX sightings_embedding_ivfflat
  ON sightings USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 50);

-- The PUBLIC API reads from this view, never from `sightings` directly.
-- It enforces the decay horizon at the data layer.
CREATE OR REPLACE VIEW sightings_public AS
SELECT id, location, label, observed_at, category, notes_scrubbed, notes_language,
       source_id, confirms, denies, created_at, updated_at,
       -- Decay weight, exposed so the client can sort/dim accordingly.
       EXP(-EXTRACT(EPOCH FROM (NOW() - observed_at)) / 3600.0 / 6.0) AS decay_weight
FROM sightings
WHERE status = 'published'
  AND observed_at >= NOW() - INTERVAL '8 hours';

-- ---------------------------------------------------------------------------
-- Resources (durable directory entries)
-- ---------------------------------------------------------------------------

CREATE TYPE resource_kind AS ENUM (
  'legal_aid',
  'hotline',
  'kyr_material',
  'mutual_aid_fund',
  'rapid_response',
  'detention_visitation',
  'bond_fund',
  'training_or_event'
);

CREATE TABLE resources (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kind                  resource_kind NOT NULL,
  name                  TEXT NOT NULL,
  description           TEXT NOT NULL DEFAULT '',
  languages             language_code[] NOT NULL DEFAULT ARRAY['en']::language_code[],
  -- Precise location is allowed for resources (public addresses).
  location              geography(Point, 4326),
  street                TEXT,
  city                  TEXT,
  region                TEXT,                 -- two-letter US state code
  postal_code           TEXT,
  country               TEXT,
  url                   TEXT,
  phone                 TEXT,
  email                 TEXT,
  hours                 TEXT,
  consent_on_file       BOOLEAN NOT NULL DEFAULT FALSE,
  consent_evidence_url  TEXT,                 -- internal only
  last_reviewed_at      TIMESTAMPTZ,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX resources_location_gix ON resources USING GIST (location);
CREATE INDEX resources_region_kind_idx ON resources (region, kind);

-- Public view never exposes consent_evidence_url and never returns resources
-- without consent_on_file.
CREATE OR REPLACE VIEW resources_public AS
SELECT id, kind, name, description, languages, location, street, city,
       region, postal_code, country, url, phone, email, hours,
       last_reviewed_at, created_at, updated_at
FROM resources
WHERE consent_on_file = TRUE;

-- ---------------------------------------------------------------------------
-- Events
-- ---------------------------------------------------------------------------

CREATE TYPE event_status AS ENUM (
  'pending',
  'published',
  'cancelled',
  'past'
);

CREATE TABLE events (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title             TEXT NOT NULL,
  description       TEXT NOT NULL DEFAULT '',
  languages         language_code[] NOT NULL DEFAULT ARRAY['en']::language_code[],
  location          geography(Point, 4326),
  street            TEXT,
  city              TEXT,
  region            TEXT,
  postal_code       TEXT,
  country           TEXT,
  starts_at         TIMESTAMPTZ NOT NULL,
  ends_at           TIMESTAMPTZ,
  host_resource_id  UUID NOT NULL REFERENCES resources(id),
  url               TEXT,
  source_id         UUID REFERENCES sources(id),
  status            event_status NOT NULL DEFAULT 'pending',
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX events_location_gix ON events USING GIST (location);
CREATE INDEX events_region_starts_at_idx ON events (region, starts_at);

-- ---------------------------------------------------------------------------
-- Moderation decisions (audit trail)
-- ---------------------------------------------------------------------------

CREATE TYPE moderation_action AS ENUM (
  'publish', 'hide', 'delete', 'escalate'
);

CREATE TYPE moderation_source AS ENUM (
  'auto', 'crowd', 'moderator', 'decay'
);

CREATE TABLE moderation_decisions (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subject_id    UUID NOT NULL,
  subject_kind  TEXT NOT NULL,        -- 'sighting' | 'resource' | 'event'
  action        moderation_action NOT NULL,
  source        moderation_source NOT NULL,
  reason        TEXT NOT NULL DEFAULT '',
  moderator_id  TEXT,                 -- opaque; only the moderation svc resolves
  at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX moderation_decisions_subject_idx
  ON moderation_decisions (subject_kind, subject_id, at DESC);

-- ---------------------------------------------------------------------------
-- Raw items (scraper output, before classification)
-- ---------------------------------------------------------------------------

CREATE TABLE raw_items (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source_id           UUID NOT NULL REFERENCES sources(id),
  text                TEXT NOT NULL,
  metadata            JSONB NOT NULL DEFAULT '{}'::jsonb,
  candidate_location  geography(Point, 4326),
  language            language_code NOT NULL DEFAULT 'en',
  embedding           vector(768),
  classified_bucket   TEXT,           -- mirrors ClassifyResponse.Bucket
  classifier_confidence REAL,
  classifier_model    TEXT,
  captured_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  processed_at        TIMESTAMPTZ
);
CREATE INDEX raw_items_unprocessed_idx
  ON raw_items (captured_at) WHERE processed_at IS NULL;
CREATE INDEX raw_items_embedding_ivfflat
  ON raw_items USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);

-- ---------------------------------------------------------------------------
-- Asynq queue tables are managed by the asynq library; not declared here.
-- ---------------------------------------------------------------------------
