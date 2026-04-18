-- 0002_seed_mn_resources.sql
--
-- Seed the resource directory with Minnesota legal-aid and mutual-aid orgs.
--
-- IMPORTANT: consent_on_file = FALSE for every row here. That means the
-- public API (which reads only from resources_public, filtered by
-- consent_on_file = TRUE) will NOT display any of them until written
-- consent is obtained. The cold-outreach workstream in the plan tracks
-- that conversion.
--
-- Until consent lands:
--   * These rows exist so we can dev against a realistic dataset.
--   * The moderator dashboard will show them in an "awaiting consent"
--     queue so outreach progress is visible.
--
-- Source: docs/data-sources.md. Edit there first, then mirror here.

INSERT INTO resources (kind, name, description, languages, url, phone, hours, region, country, consent_on_file)
VALUES
  ('legal_aid', 'Immigrant Law Center of Minnesota (ILCM)',
   'Statewide low-fee and pro-bono immigration legal services; offices in Saint Paul, Austin, Worthington, Moorhead, and Rochester.',
   ARRAY['en','es','so']::language_code[],
   'https://www.ilcm.org/', '651-641-1011', 'Mon-Fri 9am-5pm CT', 'MN', 'US', FALSE),

  ('legal_aid', 'The Advocates for Human Rights',
   'Pro-bono asylum, refugee, and removal-defense representation; know-your-rights trainings.',
   ARRAY['en','es']::language_code[],
   'https://www.theadvocatesforhumanrights.org/', '612-341-3302', 'Mon-Fri 9am-5pm CT', 'MN', 'US', FALSE),

  ('legal_aid', 'Mid-Minnesota Legal Aid',
   'Civil legal services including immigration-adjacent matters (public benefits, housing, family law) for low-income Minnesotans.',
   ARRAY['en','es']::language_code[],
   'https://mylegalaid.org/', '612-334-5970', 'Mon-Fri 9am-5pm CT', 'MN', 'US', FALSE),

  ('legal_aid', 'Southern Minnesota Regional Legal Services',
   'Civil legal aid across southern Minnesota.',
   ARRAY['en','es']::language_code[],
   'https://smrls.org/', '1-888-575-2954', 'Mon-Fri 9am-5pm CT', 'MN', 'US', FALSE),

  ('rapid_response', 'Unidos MN',
   'Grassroots Latine-led org organizing around immigration and economic justice in Minnesota.',
   ARRAY['en','es']::language_code[],
   'https://unidosmn.org/', NULL, NULL, 'MN', 'US', FALSE),

  ('rapid_response', 'COPAL (Comunidades Organizando el Poder y la Acción Latina)',
   'Latine-led organizing and mutual-aid in Minnesota; know-your-rights trainings in Spanish.',
   ARRAY['en','es']::language_code[],
   'https://copalmn.org/', '612-206-3122', NULL, 'MN', 'US', FALSE),

  ('rapid_response', 'Navigate MN',
   'Undocumented-led organization supporting undocumented immigrants in Minnesota.',
   ARRAY['en','es']::language_code[],
   'https://navigatemn.org/', NULL, NULL, 'MN', 'US', FALSE),

  ('rapid_response', 'CLUES (Comunidades Latinas Unidas en Servicio)',
   'Statewide Latino-serving nonprofit offering social services, workforce, and family support.',
   ARRAY['en','es']::language_code[],
   'https://www.clues.org/', '651-379-4200', 'Mon-Fri 8:30am-5pm CT', 'MN', 'US', FALSE),

  ('rapid_response', 'CTUL (Centro de Trabajadores Unidos en Lucha)',
   'Worker center organizing low-wage workers in the Twin Cities, with immigration-related solidarity work.',
   ARRAY['en','es']::language_code[],
   'https://ctul.net/', '612-232-5480', NULL, 'MN', 'US', FALSE),

  ('rapid_response', 'Asamblea de Derechos Civiles',
   'Grassroots civil-rights org centering Latino immigrant families in Minnesota.',
   ARRAY['en','es']::language_code[],
   'https://asambleamn.org/', NULL, NULL, 'MN', 'US', FALSE),

  ('hotline', 'United We Dream — MigraWatch',
   'National hotline for ICE-activity reports; human-verified triage.',
   ARRAY['en','es']::language_code[],
   'https://unitedwedream.org/tools/migrawatch/', '1-844-363-1423', '24/7', NULL, 'US', FALSE),

  ('bond_fund', 'Envision Freedom Fund',
   'National immigration bond fund.',
   ARRAY['en','es']::language_code[],
   'https://www.envisionfreedom.org/', NULL, NULL, NULL, 'US', FALSE),

  ('bond_fund', 'Freedom for Immigrants — National Bond Fund',
   'National immigration bond fund and visitation network.',
   ARRAY['en','es']::language_code[],
   'https://www.freedomforimmigrants.org/national-bond-fund', NULL, NULL, NULL, 'US', FALSE),

  ('kyr_material', 'ILRC Red Cards',
   'Printable know-your-rights cards in 19+ languages.',
   ARRAY['en','es','zh','ar','so']::language_code[],
   'https://www.ilrc.org/red-cards', NULL, NULL, NULL, 'US', FALSE),

  ('kyr_material', 'ACLU — Know Your Rights (Immigrants'' Rights)',
   'Know-your-rights material for encounters with law enforcement and ICE.',
   ARRAY['en','es']::language_code[],
   'https://www.aclu.org/know-your-rights/immigrants-rights', NULL, NULL, NULL, 'US', FALSE)
ON CONFLICT DO NOTHING;
