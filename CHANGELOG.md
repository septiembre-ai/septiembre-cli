# Changelog

All notable changes to the septiembre CLI are documented in this file.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- `teams list --org <slug>` — list all teams in an organization; output defaults to JSON, supports `--output table`.
- `apps create` — create a new application with flags `--name`, `--type`, `--region`, `--runtime`, `--team`. Team is auto-selected when the org has exactly one team; `--team <slug-or-id>` required when multiple teams exist. `--runtime` is required for non-web app types (`api`, `web-ssr`, `sse`). Missing `--runtime` exits 4 before any API call.
- `apps create --wait` — poll `domain_status` until `active`; timeout exits 1 with `code: wait_timeout`, domain provisioning failure exits 1 with `code: domain_failed`. Configurable with `--wait-interval` (default 5s) and `--wait-timeout` (default 10m).
- `apps delete <app-id> --yes` — initiate async teardown; `--yes` is required (exit 4 without it). 202 response renders `{"status":"deleting"}` and never claims the app has been removed synchronously.
- `apps list` now includes the composed `url` field (same `https://{subdomain}.{suffix}` logic as `apps get`).
- `apps get` now includes a composed `url` field (`https://{subdomain}.septiembre.co`) when the app has a non-null subdomain. The domain suffix is overridable via `SEPTIEMBRE_DOMAIN_SUFFIX` env var or the `domain_suffix` config key.
- `apps delete` now exits 1 with `code: teardown_dispatch_failed` when the API returns `{"status":"dispatch_failed"}` on 202, indicating the app was marked for deletion but infrastructure teardown failed to start. Normal dispatch (`{"status":"deleting"}`) still exits 0.
- `deploys trigger --wait` — poll deployment status until a terminal state (`success`, `failed`, `cancelled`); success exits 0, non-success exits 1 with `code: deploy_failed`, timeout exits 1 with `code: wait_timeout`. Configurable with `--wait-interval` (default 5s) and `--wait-timeout` (default 15m).
