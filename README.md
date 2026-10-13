# Sync Video Runner for VR

[![CI](https://github.com/MarshMeg/sync-video-runner-for-vr/actions/workflows/ci.yml/badge.svg)](https://github.com/MarshMeg/sync-video-runner-for-vr/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Russian version: [README.ru.md](README.ru.md)

`sync-video-runner-for-vr` is a small Go service with a terminal UI for coordinating synchronized video start across multiple phones/VR devices.

It does two things:
1. Accepts device registration via `GET /update-ip?id=<device_id>`.
2. Sends `GET /start` to all registered devices when you press `Ctrl+R`.

## Requirements

1. Go `1.25+` (see `go.mod`).
2. A terminal that supports ANSI escape sequences (Linux, macOS, Windows 10+).

## Quick Start

1. Run the coordinator:

```bash
go run ./cmd
```

2. Register each device (from device app, browser, or curl):

```bash
curl "http://<SERVER_IP>:50050/update-ip?id=1"
curl "http://<SERVER_IP>:50050/update-ip?id=2"
```

3. In the coordinator UI:
   - `Ctrl+R` sends `GET /start` to every registered device (`<device_ip>:8080/start`).
   - `Ctrl+E` clears the device list.
   - `Ctrl+C` exits.

## HTTP API

### `GET /update-ip?id=<int>`

Registers or updates a device in memory:
1. `id` is required and must be an integer.
2. Device IP is taken from the request source address.
3. Device port is fixed to `8080`.

Responses:
1. `201 Created` on success.
2. `400 Bad Request` if `id` is missing or is not an integer.
3. `500 Internal Server Error` if the source address cannot be determined.

## How It Works

1. **In-memory store.** Devices live in a map guarded by a mutex (`internal/server/store.go`). Nothing is written to disk: restart the coordinator and devices register again. Snapshots handed to the UI are copies, so shared state is only touched under the lock.
2. **Concurrent start.** `Ctrl+R` takes a snapshot of the device list and sends `GET /start` to every device from its own goroutine, so all requests are in flight at the same time. Each request has a 3 second timeout: an unreachable device is reported in the status line and does not delay the others.
3. **Non-blocking UI.** The HTTP server runs next to the Bubble Tea event loop; the broadcast runs as a Bubble Tea command, so the dashboard keeps refreshing (every 500 ms) while requests are in progress.
4. **Graceful shutdown.** On exit the HTTP server is stopped with `http.Server.Shutdown`, letting in-flight registrations finish.

## Tests

```bash
go test -race ./...
```

Only the standard library is used (`testing`, `net/http/httptest`):
1. Store: table-driven tests plus a concurrent readers/writers test for the race detector.
2. `/update-ip` handler: success, missing and non-numeric `id`, re-registration, IP detection from `RemoteAddr`.
3. `/start` broadcast: run against `httptest.Server` devices, including a failing device, an unreachable one, a timeout, and a check that requests really overlap.

CI runs `gofmt`, `go vet`, `go test -race` and a build on pushes to `main` and on pull requests.

## Build Binaries

Use the provided script:

```bash
./scripts/build-all.sh
```

Artifacts are created in `dist/`:
1. `vr-sync-run-linux-amd64`
2. `vr-sync-run-windows-amd64.exe`
3. `vr-sync-run-darwin-amd64`

Optional compression:
1. Install `upx` and keep default `USE_UPX=1`.
2. Disable with `USE_UPX=0 ./scripts/build-all.sh`.

Pushing a `v*` tag builds the same binaries in GitHub Actions and attaches them to a GitHub Release.

## Project Structure

1. `cmd/main.go` - app entry point (starts HTTP server and TUI).
2. `internal/server/` - in-memory device store, HTTP handler and the `/start` broadcast.
3. `internal/tui/` - terminal dashboard and hotkey handling.
4. `scripts/build-all.sh` - multi-platform build script.
5. `.github/workflows/` - CI and release workflows.

## License

[MIT](LICENSE)
