# Grafana Oracle Telemetry Streaming Plugin

A Grafana Data Source Backend Plugin that enables visualization of telemetry and streaming data from Oracle services inside Grafana dashboards.

This plugin allows users to connect Grafana to Oracle telemetry and streaming infrastructure and build real-time dashboards for monitoring and observability use cases.

---

## Overview

This repository contains a Grafana backend data source plugin composed of:

- A **frontend component** built with Grafana Toolkit (React + TypeScript)
- A **backend component** written in Go using the Grafana Plugin SDK

The plugin enables secure data retrieval and visualization of Oracle telemetry and streaming data inside Grafana.

The backend plugin uses Oracle database connectivity through the `godror` driver and Oracle Instant Client libraries.

---

## Installation

### System Requirements

#### Frontend Requirements

- Node.js **16.x** (required) (suggested v16.20.2)
- Yarn **1.x** (suggested 1.22.11)
- npm 8.x (bundled with Node 16) (suggested 8.19.4)

These are fully compatible with @grafana/toolkit v8.5.27 and webpack 4.41.5.

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

#### Backend Requirements

- Go 1.24.0
- Go modules enabled

Verify:

```bash
go version
```

#### Oracle Client Runtime Requirements

The backend datasource plugin uses the `godror` Oracle driver and requires Oracle Instant Client libraries on the Grafana host.

Install Oracle Instant Client (Basic or Basic Lite) and ensure the Oracle client libraries are available through `LD_LIBRARY_PATH`.

Example installation on Oracle Linux / RHEL:

```bash
# Download Oracle Instant Client RPM
wget https://download.oracle.com/otn_software/linux/instantclient/2380000/oracle-instantclient-basiclite-23.8.0.25.04-1.el9.x86_64.rpm

# Install package
sudo dnf install -y oracle-instantclient-basiclite-23.8.0.25.04-1.el9.x86_64.rpm
```

Configure runtime library path:

```bash
echo 'export LD_LIBRARY_PATH=/usr/lib/oracle/23/client64/lib:$LD_LIBRARY_PATH' >> ~/.bashrc
source ~/.bashrc
```

Optional:

```bash
echo 'export PATH=/usr/lib/oracle/23/client64/bin:$PATH' >> ~/.bashrc
```

Verify:

```bash
echo $LD_LIBRARY_PATH
```
#### Oracle Network Configuration

Additional Oracle client configuration may be required depending on the Oracle environment.

For wallet-based connections, such as Oracle Autonomous Database or TCPS-enabled databases, extract the complete wallet archive into a directory and set `TNS_ADMIN` to that directory.

The extracted wallet directory should contain files such as:

* `tnsnames.ora`
* `sqlnet.ora`
* Oracle wallet files

Example:

```bash
export TNS_ADMIN=/path/to/extracted/wallet
```

After extracting the wallet, open the `sqlnet.ora` file in the same directory.

By default, it may contain a wallet location similar to:

```text
WALLET_LOCATION = (SOURCE = (METHOD = file) (METHOD_DATA = (DIRECTORY="?/network/admin")))
```

Update the `DIRECTORY` value so that it points to the directory where the wallet was extracted. This should normally be the same directory assigned to `TNS_ADMIN`.

Example:

```text
WALLET_LOCATION = (SOURCE = (METHOD = file) (METHOD_DATA = (DIRECTORY="/path/to/extracted/wallet")))
SSL_SERVER_DN_MATCH=yes
```

For example:

```bash
export TNS_ADMIN=/scratch/user/project/tklocal/wallet93
```

The corresponding `sqlnet.ora` entry should be:

```text
WALLET_LOCATION = (SOURCE = (METHOD = file) (METHOD_DATA = (DIRECTORY="/scratch/user/project/tklocal/wallet93")))
SSL_SERVER_DN_MATCH=yes
```

Ensure that:

1. The entire wallet archive is extracted into the directory.
2. `TNS_ADMIN` points to the extracted wallet directory.
3. The `WALLET_LOCATION` value in `sqlnet.ora` points to the same directory.

### Cloning the Repository

```bash
git clone <repository-url>
cd <repository-folder>
```

## Build Instructions

### Frontend Build

Install Dependencies

```bash
yarn install
```

Development Mode

```bash
yarn dev
```

This starts frontend development mode with hot reload.

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

### Backend Build

The backend component is implemented in Go using the Grafana Plugin SDK.

#### Install Dependencies

```bash
go mod tidy
```

#### Build Backend Binary

```bash
go build -o dist/gpx_oracle-telemetry_linux_amd64 ./pkg
```

This generates the backend plugin binary inside the `dist/` directory.

---

## Running the Plugin in Grafana

1. Download a stable version of Grafana (recommended: version 8 through 12) from:
   https://grafana.com/grafana/download

Example:

```bash
wget https://dl.grafana.com/enterprise/release/grafana-enterprise-8.5.13.linux-amd64.tar.gz
tar -zxf grafana-enterprise-8.5.13.linux-amd64.tar.gz
```

2. Extract Grafana to a directory (e.g., `<GRAFANA_HOME>`).

3. Copy the whole oracle-telemetry-streaming-main directory, including all files and subdirectories inside it, into Grafana’s data/plugins directory.

Do not copy only the contents of the plugin folder.

The resulting directory structure should be:

   ```
   <GRAFANA_HOME>/data/plugins/oracle-telemetry-streaming-main/
   ```

Example:

```bash
mkdir -p <GRAFANA_HOME>/data/plugins/

cp -r oracle-telemetry-streaming-main \
  <GRAFANA_HOME>/data/plugins/
```
After copying, verify that the plugin directory exists:

```bash
ls "$GRAFANA_HOME/data/plugins/oracle-telemetry-streaming-main"
```

4. Enable unsigned plugins in:

   ```
   <GRAFANA_HOME>/conf/defaults.ini
   ```

Add or update:

```ini
[plugins]
allow_loading_unsigned_plugins = oracle-oracle-telemetry
```

5. Start or restart the Grafana server.

Example:

```bash
cd <GRAFANA_HOME>/bin

./grafana-server
```

Or run in background:

```bash
nohup ./grafana-server &
```

6. Open Grafana in browser:

```text
http://<SERVER_IP>:3000
```

Default credentials:

```text
admin/admin
```

7. Log in to Grafana and add the data source from the UI.

8. (Optional) Open firewall port if firewall rules are enabled:

```bash
sudo firewall-cmd --permanent --add-port=3000/tcp
sudo firewall-cmd --reload
```

Verify:

```bash
sudo firewall-cmd --list-ports
```

---

## Project Structure

```text
.
├── .github/              # GitHub workflows and automation
├── img/                  # Plugin images and assets
├── pkg/                  # Go backend source code
├── src/                  # Frontend source (React/TypeScript)
├── docs/                 # Detailed plugin architecture and documentation
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

## Documentation

Grafana Backend Plugin Documentation:

- https://grafana.com/docs/grafana/latest/developers/plugins/backend/
- https://grafana.com/docs/grafana/latest/developers/plugins/backend/grafana-plugin-sdk-for-go/

Additional documentation is available in the `docs/` directory.

---

## Contributing

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
- Add tests whenever possible.
- Tests pass.
- No unintended files (e.g., `node_modules/`, `dist/`) are committed

This project welcomes contributions from the community. Before submitting a pull request, please review our [contribution guide](./CONTRIBUTING.md)

---

## Security

If you discover a security vulnerability, please follow the responsible disclosure process described in our [security guide](./SECURITY.md).

---

## License

Copyright (c) 2026 Oracle and/or its affiliates.

Released under the Universal Permissive License v1.0

https://oss.oracle.com/licenses/upl/

See [LICENSE](./LICENSE.txt)
