# Request Info API Design

## Goal

Create a very small Go API that returns information about the incoming HTTP request as JSON. Deploy it as a Minato app named `request-info` in tenant `acme` from a public GitHub repository under `stianfro/request-info`.

## Scope

The first version includes one HTTP handler that responds on every path. It is meant for quick inspection of request metadata when testing routing, headers, protocol behavior, and Minato deployment.

The app does not include authentication, persistence, a frontend, or request body echoing.

## Architecture

- Go module at the repository root.
- Entrypoint in `cmd/request-info/main.go`.
- Request inspection package in `internal/requestinfo`.
- HTTP server uses the Go standard library only.
- The server listens on the `PORT` environment variable, defaulting to `8080` for local use.
- Minato routes to container port `8080`.

## API Behavior

Every request receives a `200 OK` JSON response with:

- method
- URL path
- raw query string
- host
- protocol
- remote address
- request URI
- headers
- content length
- transfer encoding
- TLS enabled flag
- server timestamp

Sensitive request headers are redacted before being returned. Redacted headers include `Authorization`, `Cookie`, `Set-Cookie`, `Proxy-Authorization`, and `X-Api-Key`, using case-insensitive matching.

The response includes headers sorted by name so test output and client output are stable.

## Error Handling

The handler only fails if JSON encoding fails while writing the response. In that case it logs the error and returns `500 Internal Server Error` if the response has not already been written.

Startup fails fast if the configured port is invalid or the server cannot bind.

## Development Tasks

A `justfile` manages development commands:

- `just fmt`: run `gofmt`.
- `just lint`: run `go vet`.
- `just test`: run `go test ./...`.
- `just run`: run the API locally.
- `just ci`: run formatting check, lint, and tests.

No YAML files are planned. If YAML is added later, it must be validated with `yq`.

## Testing

Tests cover:

- JSON response shape for a sample request.
- Header redaction.
- Stable handling of request metadata.
- Port parsing defaults.

Final verification before handoff must run `just ci`.

## Deployment Plan

1. Create and commit the Go app and `justfile`.
2. Create public GitHub repository `stianfro/request-info` with `gh`.
3. Push `main`.
4. Create the Minato app `request-info` in tenant `acme` without an initial image.
5. Start a Minato build from `https://github.com/stianfro/request-info`, ref `main`, builder `go`.
6. Poll the build until it succeeds.
7. Deploy the succeeded build.
8. Poll the app until it reports a live state.
9. Report repository URL, Minato tenant, app name, build ID, and live status.
