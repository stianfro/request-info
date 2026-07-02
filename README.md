# request-info

A small Go API that returns information about the incoming HTTP request as JSON.

## Local development

Run checks:

```bash
just ci
```

Run the server:

```bash
just run
```

Use a custom port:

```bash
PORT=9090 just run
```

Send a request:

```bash
curl -H 'X-Demo: hello' 'http://localhost:8080/example?debug=true'
```

## Response

The API returns JSON with request metadata:

- method
- path
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

Sensitive headers are redacted before they are returned.
