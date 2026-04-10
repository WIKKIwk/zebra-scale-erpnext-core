# Agent Handoff

## Scope

This workspace currently has two git-tracked codebases that matter:

- `gscale-zebra`
- `hard_con_v2` (nested repo used as the current mobile client)

There is also a separate local ERPNext/Frappe bench checkout on this machine, but it is **not** tracked by this repository.

## Current Repo State

- `gscale-zebra`
  - branch: `main`
  - clean working tree
  - HEAD: `42d2f7c` (`Refine local dev run targets`)

- `hard_con_v2`
  - branch: `main`
  - clean working tree
  - HEAD: `70dc1e1` (`Split dashboard and add control panel`)

## What Was Completed

### 1. Local gscale dev flow

`Makefile` in `gscale-zebra` was adjusted so local dev/testing is easier.

Important behavior:

- `make run-dev` now brings up the local fake stack for development
- it is intended for:
  - polygon simulator
  - mobileapi
  - scale side flow
- mobile app runtime is not part of that target

### 2. Mobile UI split

`hard_con_v2` now has a cleaner three-panel structure:

- `Server`
- `Line`
- `Control`

`Control` is currently UI-only and is meant as the future destination for a subset of the Telegram bot workflow:

- product selection
- batch start/stop
- live kg display

The current app keeps the existing server and line pages, and adds a separate control page instead of overloading a single dashboard.

### 3. ERP bench recovery on this machine

A separate local ERPNext bench checkout was repaired enough to run locally on this Mac.

Important local-only outcomes:

- a fresh site was created:
  - site: `erpfresh.localhost`
  - login: `Administrator`
  - password: `1`
- `bench start` and `bench stop` were made usable from the bench root on this machine
- assets were rebuilt so the login page is styled again

These ERP-side changes are **not committed here**, because that bench checkout is outside this repo and was not git-managed in this session.

## Important Architecture Decision

The next service should **not** live inside `gscale-zebra`.

The agreed direction is:

- `gscale-zebra` bot remains the main orchestrator
- ERPNext stays on its own machine / bench host
- a new **standalone Go read-only service** should live on the ERP side
- bot and mobile app will call that service over HTTP
- ERP write operations should continue to go through the existing ERP API flow

This means:

- do **not** continue by extending `mobileapi` for the ERP read gateway
- do **not** move ERP write logic into direct DB writes
- do **not** collapse ERP-side service and gscale runtime into one process

## Bot Workflow Summary

The Telegram bot workflow is already mostly usable as the business source of truth.

Current flow:

1. `/batch`
2. item search
3. warehouse selection
4. `Material Receipt` start
5. wait for stable positive qty from bridge state
6. create EPC
7. create ERP draft
8. set `print_request`
9. wait for print result
10. submit or delete draft

This is why the upcoming service can stay read-only for now:

- item search can be offloaded
- warehouse shortlist can be offloaded
- filtering/ranking can be centralized
- core batch orchestration can remain in bot/app logic until later

## ERP Bench Notes For The Next Agent

Do **not** rely on a hardcoded filesystem path for the ERP repo.

Find the ERP bench checkout by looking for a local repository/folder with this shape:

- `Procfile`
- `apps/`
- `sites/`
- `config/`
- `restart_bench.sh`
- `stop_bench.sh`

Operational notes:

- run `bench start` from the bench root
- run `bench stop` from the bench root
- default site is `erpfresh.localhost`
- if UI assets break again, rerun `bench build`

There were local ERP bench adjustments made for this machine:

- broken Linux-origin paths were repaired
- asset links were repaired
- `watch` was intentionally disabled in `Procfile` to keep local start stable

So if the next agent sees missing live-reload behavior, that is intentional and not the first thing to undo.

## Recommended Next Steps

### Immediate next implementation target

Create a new standalone Go service on the ERP side.

Initial version should do only:

1. health endpoint
2. MariaDB read-only connection
3. item search endpoint
4. warehouse shortlist endpoint for a selected item

### Suggested endpoint contract

- `GET /healthz`
- `GET /v1/items?query=...`
- `GET /v1/items/{item_code}/warehouses?query=...`

### Suggested rollout order

1. scaffold standalone service repo/folder
2. add config + DB connection
3. implement item search
4. implement warehouse shortlist
5. test against local ERP bench
6. switch bot read path from ERP HTTP search to new service
7. then wire the mobile control panel to the same service

## Things To Avoid

- Do not hardcode a local ERP path into documentation or code comments.
- Do not move ERP write operations to direct SQL.
- Do not assume the ERP bench repo is under git control.
- Do not remove the current bot workflow before the new service proves parity for read operations.
