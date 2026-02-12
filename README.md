# Grafana Oracle Telemetry Streaming Plugin

A Grafana Data Source Backend Plugin that enables visualization of telemetry and streaming data from Oracle services inside Grafana dashboards.

This plugin allows users to connect Grafana to Oracle telemetry and streaming infrastructure and build real-time dashboards for monitoring and observability use cases.

---

# Overview

This repository contains a Grafana backend data source plugin composed of:

- A **frontend component** built with Grafana Toolkit (React + TypeScript)
- A **backend component** written in Go using the Grafana Plugin SDK

The plugin enables secure data retrieval and visualization of Oracle telemetry and streaming data inside Grafana.

---

# System Requirements

## Frontend Requirements

- Node.js **16.x** (required)
- Yarn **1.x**
- npm 8.x (bundled with Node 16)


Recommended setup using nvm:

```bash
nvm install 16
nvm use 16
```

Verify:

```bash
node --version
yarn --version
```

---

## Backend Requirements

- Go **1.20 or later**
- Go modules enabled

Verify:

```bash
go version
```

---

# Cloning the Repository

```bash
git clone <repository-url>
cd <repository-folder>
```

---

# Frontend Development

The frontend uses `@grafana/toolkit`.

## Install Dependencies

```bash
yarn install
```

## Development Mode

```bash
yarn dev
```

Or watch mode:

```bash
yarn watch
```

## Run Tests

```bash
yarn test
```

## Production Build

```bash
yarn build
```

This generates optimized frontend assets inside the `dist/` directory.

---

# Backend Development

The backend component is implemented in Go using the Grafana Plugin SDK.

## Install Dependencies

```bash
go mod tidy
```

## Build Backend Binary

```bash
go build -o dist/gpx_oracle-telemetry_linux_amd64 ./pkg
```

This generates the backend plugin binary inside the `dist/` directory.


---

# Running the Plugin in Grafana

1. Copy the plugin folder to the Grafana plugins directory:

   ```
   $GRAFANA_HOME/data/plugins/
   ```

2. Enable unsigned plugins in `GRAFANA_HOME/conf/defaults.ini`:

   ```
   allow_loading_unsigned_plugins = <plugin-id>
   ```
   where plugin-id is "oracle-oracle-telemetry"
3. Restart Grafana.
4. Log in to Grafana and add the data source from the UI.(default grafana login is admin/admin)

---

# Project Structure

```
.
├── pkg/                  # Go backend source
├── src/                  # Frontend source (React/TypeScript)
├── dist/                 # Generated build artifacts
├── plugin.json           # Grafana plugin definition
├── package.json          # Frontend dependencies
├── go.mod                # Go module definition
└── README.md
```

---

# Build Workflow Summary

Frontend:

```bash
yarn install
yarn build
```

Backend:

```bash
go mod tidy
go build -o dist/gpx_oracle-telemetry_linux_amd64 ./pkg
```

---

# Documentation

Grafana Backend Plugin Documentation:

- https://grafana.com/docs/grafana/latest/developers/plugins/backend/
- https://grafana.com/docs/grafana/latest/developers/plugins/backend/grafana-plugin-sdk-for-go/

---

# Contributing

Contributions are welcome.

Before submitting a pull request:

1. Ensure Node 16 is used.
2. Run:
   ```bash
   yarn build
   ```
3. Build backend:
   ```bash
   go build ./pkg
   ```
4. Ensure tests pass:
   ```bash
   yarn test
   ```

Please follow the contribution guidelines described in `CONTRIBUTING.md`.

---

# Security

If you discover a security vulnerability, please follow the responsible disclosure process described in `SECURITY.md`.

---

# License

Copyright (c) 2026 Oracle and/or its affiliates.

Released under the Universal Permissive License v1.0  
https://oss.oracle.com/licenses/upl/
