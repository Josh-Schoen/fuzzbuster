# Legal Review Checklist

This checklist must be fully green before Fuzzbuster's canonical instance accepts public traffic.

## Fiscal sponsorship

- [ ] Fiscal-sponsor application submitted (Hack Club Bank or Open Collective Foundation as primary; NEO Philanthropy or movement-aligned 501(c)(3) as fallback).
- [ ] Sponsor approval received in writing.
- [ ] Stripe / Open Collective ledger live and linked from the public donations page.
- [ ] Publicly named treasurer / point-of-contact at the sponsor.

## Counsel

- [ ] EFF intake submitted (`eff.org/issues/coders`); confirmation logged.
- [ ] **ACLU of Minnesota** (`aclu-mn.org`) intake submitted; confirmation logged.
- [ ] Sponsor's general counsel has read `docs/safety-policy.md` and signed off in writing.
- [ ] DMCA / safe-harbor designated agent registered (if user-submitted content qualifies).

## Policy artifacts (public)

- [ ] `docs/safety-policy.md` published at a stable URL on the canonical instance.
- [ ] Terms of Service published.
- [ ] Privacy Policy published, consistent with safety policy (no IP logging at app layer, no device IDs, no third-party analytics).
- [ ] Transparency-report template ready; first quarterly transparency report scheduled.
- [ ] Subpoena / law-enforcement-request response procedure documented and rehearsed.

## Technical guarantees verified

- [ ] PII scrubber golden-set leak rate = 0 (`make test-pii`).
- [ ] Time-decay verified: no sighting older than 8h is queryable from the public API (`make test-decay`).
- [ ] Geo coarsening verified: no sighting record contains a coordinate finer than the 200m grid (`make test-geo`).
- [ ] Reverse-proxy IP-logging disabled or rotated within 24h.
- [ ] Backups encrypted; key escrow with sponsor's counsel confirmed.

## Partnerships (Minnesota launch)

At least one of the following is required for a non-"unverified" public launch banner:

- [ ] ILCM (Immigrant Law Center of Minnesota) listed in the legal-aid directory with their consent.
- [ ] The Advocates for Human Rights listed in the legal-aid directory with their consent.
- [ ] Mid-Minnesota Legal Aid listed with their consent.
- [ ] One of: Unidos MN, COPAL, Navigate MN, CLUES, Asamblea de Derechos Civiles, CTUL, MIRAC, or Twin Cities Rapid Response Network signed off as a partner-feed source.

## Take-down readiness

- [ ] Codebase mirror exists on at least one non-GitHub forge (Codeberg or self-hosted Gitea).
- [ ] Container images pushed to at least one non-GitHub registry.
- [ ] DNS held by an account separate from the operator's primary identity.
- [ ] Documented rehosting runbook (~30 min RTO).
