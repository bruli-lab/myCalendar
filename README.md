# myCalendar

myCalendar is a small Go service that reads your Google Calendar events and Google Tasks and sends a daily and a weekly agenda to an [ntfy](https://ntfy.sh) topic.

## Features

- Daily summary of today's events and tasks, and weekly summary of the next seven days.
- One markdown message per summary, with an **Events** and a **Tasks** section. All-day events, event times in your timezone and links to every event and task.
- Google OAuth2 authorization through a link published to ntfy. Tokens are stored in a local file and refreshed only when they expire.
- If Google revokes the refresh token, the service publishes a new authorization link, both at startup and while it is running, without restarting.
- Configurable calendar, timezone, schedules and job timeout.
- OpenTelemetry traces and JSON logs.
- `GET /healthz` for liveness and readiness probes.

## How it works

```
cron ──▶ SendSummary command ──▶ Google Calendar + Google Tasks
                │
                ▼
        SummaryGenerated event ──▶ PublishSummary listener ──▶ ntfy
```

1. On startup the service refreshes the stored token. When there is no token file, or Google answers `invalid_grant`, it starts a temporary callback server and publishes the authorization link to ntfy.
2. You open the link and authorize read-only access. Google redirects to `CALLBACK_URL` with a code and the `state` generated for that link. Requests with a different `state` are rejected.
3. The tokens are written to `TOKENS_FILE` with `0600` permissions.
4. The scheduler sends the summaries. Each one fetches the events and tasks of its period, raises a `SummaryGenerated` domain event, and a listener formats and publishes it.

## Architecture

Hexagonal architecture with DDD, CQS and an internal event bus, built on [go-core](https://github.com/bruli-lab/go-core).

| Package | Responsibility |
|---|---|
| `internal/domain` | `Auth`, `Period`, `Event`, `Task` and the `Summary` aggregate that records `SummaryGenerated`. No infrastructure dependencies. |
| `internal/app` | Command handlers: `RefreshToken`, `CreateTokens`, `SendSummary`, and the ports they need. |
| `internal/infra/google` | Calendar and Tasks repositories, OAuth token repository and a persistent token source. |
| `internal/infra/listener` | `PublishSummary`, which formats a summary as markdown and publishes it. |
| `internal/infra/ntfy` | ntfy publisher (JSON API, markdown enabled). |
| `internal/infra/http` | Health server, OAuth callback server and the `Authorizer` that drives the OAuth flow. |
| `internal/infra/disk` | Token file repository with atomic writes. |
| `internal/config` | Environment configuration, validated at startup. |
| `cmd/myCalendar` | Composition root. |

## Requirements

- Go 1.27.4 or later
- [Task](https://taskfile.dev)
- Docker, for the local environment and the infra and functional tests
- A Google Cloud OAuth2 client with the Calendar and Tasks APIs enabled. Add `CALLBACK_URL` to its **Authorized redirect URIs**.

## Configuration

All configuration comes from environment variables.

| Variable | Required | Default | Description |
|---|---|---|---|
| `CLIENT_ID` | yes | | Google OAuth2 client ID |
| `CLIENT_SECRET` | yes | | Google OAuth2 client secret |
| `CALLBACK_HOST` | yes | | Address of the temporary callback server, e.g. `:8081` |
| `CALLBACK_URL` | yes | | Public URL Google redirects to, e.g. `https://mycalendar.example.com/callback` |
| `TOKENS_FILE` | yes | | Path of the token file |
| `SERVER_HOST` | yes | | Address of the health server, e.g. `:8080` |
| `LOG_LEVEL` | yes | | `debug`, `info`, `warn` or `error` |
| `NTFY_URL` | yes | | ntfy server URL |
| `NTFY_TOPIC` | yes | | ntfy topic |
| `NTFY_USER` | yes | | ntfy user |
| `NTFY_PASSWORD` | yes | | ntfy password |
| `CALENDAR_ID` | no | `primary` | Google Calendar to read |
| `TIMEZONE` | no | `Europe/Madrid` | Timezone of the schedules and of the summaries |
| `DAILY_SCHEDULE` | no | `0 8 * * 0,2-6` | Cron expression of the daily summary |
| `WEEKLY_SCHEDULE` | no | `0 8 * * 1` | Cron expression of the weekly summary |
| `JOB_TIMEOUT` | no | `2m` | Maximum duration of a summary job |
| `GOOGLE_API_URL` | no | | Overrides the Google endpoints. Only used by the functional tests; when set it must be an absolute `http` or `https` URL |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | no | | OpenTelemetry collector endpoint (standard `OTEL_*` variables apply) |

The service fails to start when a required variable is missing or empty, or when `LOG_LEVEL`, `TIMEZONE` or a schedule is invalid.

## Running locally

```bash
cp .env.example .env
```

Fill `CLIENT_ID` and `CLIENT_SECRET` in `.env`. The example sends both summaries every minute so you can try it right away.

Run the app with `go run`, with ntfy and Grafana LGTM in containers:

```bash
task run
```

Or run the whole stack in containers:

```bash
task docker-up
```

Then open the notifications at http://localhost:8086/mycalendar, click the authorization link and authorize the access. Traces are available in Grafana at http://localhost:3002.

## Development

| Task | Description |
|---|---|
| `task test` | Unit tests with race detector and coverage |
| `task test-infra` | Infrastructure tests: token file, Google adapters, OAuth flow and ntfy (Docker) |
| `task test-functional` | Builds the app and runs it end to end against ntfy and a fake Google API (Docker) |
| `task coverage` | Opens the coverage report |
| `task generate` | Regenerates the moq mocks |
| `task fmt` / `task fmt-check` | Formats with gofumpt / fails if something is not formatted |
| `task lint` | golangci-lint |
| `task security` | govulncheck |
| `task check` | fmt, lint, security and unit tests |
| `task docker-up` / `task docker-down` / `task docker-logs` | Local stack with Docker Compose |

Tests follow the Given / When / Then style, unit tests run in parallel, mocks are generated with [moq](https://github.com/matryer/moq) and assertions use [testify](https://github.com/stretchr/testify).

## CI/CD

The GitHub Actions workflow runs on every pull request and on every push to `main`:

1. Format check and lint, unit tests, infra and functional tests, govulncheck and a Trivy scan of the repository.
2. On `main`, [semantic-release](https://semantic-release.gitbook.io) creates the version from the Conventional Commits.
3. The `arm64` image is built, scanned with Trivy and pushed to `ghcr.io/bruli-lab/mycalendar`.
4. `manifests/deployment.yaml` is updated with the new tag, and ArgoCD deploys it to the cluster.

## Deployment

`manifests/` contains the Kustomize manifests for Kubernetes: namespace, deployment, service, a persistent volume for the token file and ExternalSecrets for the Google and ntfy credentials. The container runs as a non-root user on a distroless image with a read-only root filesystem.

## License

See [LICENSE](LICENSE).
