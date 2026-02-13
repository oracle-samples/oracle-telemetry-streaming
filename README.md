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

# Installation

## System Requirements

### Frontend Requirements

- Node.js **16.x** (required) (suggested v16.20.2)
- Yarn **1.x** (suggested 1.22.11)
- npm 8.x (bundled with Node 16) (suggested 8.19.4)
These are fully compatible with @grafana/toolkit v8.5.27 and webpack 4.41.5

Recommended setup using nvm:

```bash
nvm install 16
nvm use 16
nvm alias default 16
```
Verify:

```bash
node --version
yarn --version
```

### Backend Requirements

- Go 1.24.0
- Go modules enabled

Verify:

```bash
go version
```

## Cloning the Repository

```bash
git clone <repository-url>
cd <repository-folder>
```

# Build Instructions

## Frontend Build

Install Dependencies

```bash
yarn install
```

Development Mode

```bash
yarn dev
```

Or watch mode:

```bash
yarn watch
```

Run Tests

```bash
yarn test
```

Production Build

```bash
yarn build
```

This generates optimized frontend assets inside the `dist/` directory.


## Backend Build

The backend component is implemented in Go using the Grafana Plugin SDK.

### Install Dependencies

```bash
go mod tidy
```

### Build Backend Binary

```bash
go build -o dist/gpx_oracle-telemetry_linux_amd64 ./pkg
```

This generates the backend plugin binary inside the `dist/` directory.


---

# Running the Plugin in Grafana
1. Download a stable version of Grafana (recommended: version 8 through 12) from:
   https://grafana.com/grafana/download
2. Extract Grafana to a directory (e.g., `<GRAFANA_HOME>`).
3. Copy the plugin folder into:

   ```
   <GRAFANA_HOME>/data/plugins/
   ```

4. Enable unsigned plugins in:

   ```
   <GRAFANA_HOME>/conf/defaults.ini
   ```

   Add or update:

   ```
   allow_loading_unsigned_plugins = oracle-oracle-telemetry
   ```

5. Start or restart the Grafana server.

6. Log in to Grafana (default credentials: `admin/admin`) and add the data source from the UI.

---

# Project Structure

```
.
├── .github/              # GitHub workflows and automation
├── img/                  # Plugin images and assets
├── pkg/                  # Go backend source code
├── src/                  # Frontend source (React/TypeScript)
├── docs/                 # Plugin architecture and documentation in detail.
├── dist/                 # Generated build artifacts (not committed)
├── Magefile.go           # Mage build targets for backend
├── plugin.json           # Grafana plugin definition
├── package.json          # Frontend dependencies
├── go.mod / go.sum       # Go module definitions
├── tsconfig.json         # TypeScript configuration
├── jest.config.js        # Frontend test configuration
├── .nvmrc                # Node version pin (Node 16)
├── CONTRIBUTING.md       # Contribution guidelines
├── SECURITY.md           # Security disclosure policy
├── LICENSE               # License information
└── README.md
```

---

# Documentation

Grafana Backend Plugin Documentation:

- https://grafana.com/docs/grafana/latest/developers/plugins/backend/
- https://grafana.com/docs/grafana/latest/developers/plugins/backend/grafana-plugin-sdk-for-go/
- Additional documentation is available in the `docs/` directory:

---

# Contributing

Contributions are welcome.

To submit improvements or fixes, please follow the steps below:
1. Clone the Repository
   ```
   git clone <REPO_LINK>
   cd <REPO_NAME>
   ```
2. Create a New Branch
   ```
   git checkout -b <your-branch-name>
   ```
3. Install Frontend Dependencies, Ensure you are using **Node 16**.
   ```
   yarn install
   ```
4. Make Your Changes. Implement your changes in the new branch. Before submitting a pull request, verify that both frontend and backend build successfully.
   ### Build Frontend
   ```bash
   yarn build
   ```
   ### Build Backend
   ```bash
   go build -o dist/gpx_oracle-telemetry_linux_amd64 ./pkg
   ```
5. Commit and Push Your Changes
   ```
   git add .
   git commit -m "Describe your changes clearly"
   git push origin <your-branch-name>
   ```
6. Open a Pull Request
  Create a pull request from your branch to the `main` branch.

Please ensure:
- The project builds successfully
- Add Test whenever possible.
- Tests pass.
- No unintended files (e.g., `node_modules/`, `dist/`) are committed
   

This project welcomes contributions from the community. Before submitting a pull request, please review our [contribution guide](./CONTRIBUTING.md)

---

# Security

If you discover a security vulnerability, please follow the responsible disclosure process described in our [security guide](./SECURITY.md)`.

---

# License

Copyright (c) 2026 Oracle and/or its affiliates.

Released under the Universal Permissive License v1.0  
https://oss.oracle.com/licenses/upl/

See [LICENSE](./LICENSE.txt)
