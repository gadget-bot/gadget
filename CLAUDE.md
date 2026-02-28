# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Gadget is a Go Slack bot framework built on the Slack Events API, inspired by the Lita framework. It provides regex-based message routing, group-based permissions, and MySQL persistence via GORM.

## Build & Development Commands

```bash
make build        # Compile static binary to dist/
make test         # Run tests with coverage
make lint         # Run golint
make all          # clean → verify → lint → test → build
make container    # Build Docker image
make start-db     # Start local MariaDB container
make stop-db      # Stop local MariaDB container
make tools        # Install dev tools (golint)
```

Run a single test (no make target exists for this):
```bash
go test -v ./router/ -run TestFunctionName
```

**Always prefer `make` targets over calling `go` commands directly.** The Makefile ensures consistent flags, ldflags, and environment settings. Only fall back to raw `go` commands when no suitable make target exists (e.g., running a single test).

## Architecture

### Core Flow

HTTP POST `/gadget` → signature verification → event parsing → route matching → permission check → plugin execution (in goroutine)

### Key Packages

- **core/** — Bot initialization (`Setup()` / `SetupWithConfig()`), HTTP server, event dispatch. Entry point for understanding the system.
- **router/** — Route definitions, regex matching, priority-based sorting, permission checking via `Can()`. Routes are maps keyed by name. Two route types: `MentionRoute` (app mentions) and `ChannelMessageRoute` (channel messages), both embedding a base `Route` struct.
- **models/** — GORM models for `User` and `Group` with many-to-many relationship (`user_groups` join table). Groups drive the permission system.
- **plugins/** — Built-in plugins: `groups` (group management), `user_info` (user lookup), `fallback` (default reply), `permission_denied` (access denied reply).

### Plugin System

Plugins are functions matching a specific signature, registered as routes during setup:

```go
// MentionRoute plugin signature
func(router Router, route Route, api slack.Client, ev slackevents.AppMentionEvent, message string)

// ChannelMessageRoute plugin signature
func(router Router, route Route, api slack.Client, ev slackevents.MessageEvent, message string)
```

Routes are added via `Router.AddMentionRoute()` / `Router.AddMentionRoutes()` and their channel message equivalents.

### Permissions

- Routes define required permissions as a slice of group names
- `Router.Can()` checks: globalAdmins → empty permissions (allow all) → "*" wildcard → group membership match
- `DeniedMentionRoute` handles unauthorized access

### Configuration

Environment variables: `SLACK_OAUTH_TOKEN`, `SLACK_SIGNING_SECRET`, `GADGET_GLOBAL_ADMINS` (comma-separated user IDs), `GADGET_DB_USER`, `GADGET_DB_PASS`, `GADGET_DB_HOST`, `GADGET_DB_NAME`, `GADGET_LISTEN_PORT` (default 3000), `GADGET_LOG_LEVEL` (default `info`; valid values: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`).

## Dependencies

- `github.com/slack-go/slack` — Slack API client
- `gorm.io/gorm` + `gorm.io/driver/mysql` — ORM and database
- `github.com/rs/zerolog` — Structured logging
- `github.com/stretchr/testify` — Test assertions

## Conventions

- Static binary: `CGO_ENABLED=0`
- Route plugin execution is always async (goroutines)
- GORM `FirstOrCreate` pattern for safe upserts
- Feature branch + PR merge workflow

## GitHub Issues

When opening issues for this project (or related projects like Penny):
- Always apply an issue type and the best fitting label
- Scan existing issues to identify relationships (sub-issues, duplicates, related issues)
- Ask before changing existing relationships
