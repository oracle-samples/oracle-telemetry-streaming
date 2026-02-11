# Grafana Oracle Telemetry Streaming Plugin

A Grafana Data Source Backend Plugin that enables visualization of telemetry and streaming data from Oracle services inside Grafana dashboards.

This plugin allows users to connect Grafana to Oracle telemetry and streaming infrastructure and build real-time dashboards for monitoring and observability use cases.

---

## Features

- Backend-powered Grafana data source
- Integration with Oracle telemetry/streaming services
- Secure data retrieval
- Cross-platform backend binaries (Linux, Windows, macOS)
- Built using Grafana Plugin SDK for Go

---

## Getting Started

A data source backend plugin consists of both frontend and backend components.

### Frontend

1. Install dependencies

   yarn install

2. Build plugin in development mode

   yarn dev

   or run in watch mode:

   yarn watch

3. Build plugin for production

   yarn build

---

### Backend

1. Update the Grafana plugin SDK dependency:

   go get -u github.com/grafana/grafana-plugin-sdk-go  
   go mod tidy

2. Build backend binaries:

   mage -v

3. List available Mage targets:

   mage -l

---

## Documentation

Developer documentation for building Grafana backend plugins is available at:

https://grafana.com/docs/grafana/latest/developers/plugins/backend/  
https://grafana.com/docs/grafana/latest/developers/plugins/backend/grafana-plugin-sdk-for-go/

---

## Examples

Example dashboards and configuration samples can be added here to demonstrate:

- Connecting to Oracle telemetry services
- Query configuration
- Dashboard creation

---

## Help

For issues, please open a GitHub issue in this repository.

If this project is officially supported by Oracle, refer to Oracle support channels.

---

## Contributing

This project welcomes contributions from the community.  
Before submitting a pull request, please review our contribution guide in CONTRIBUTING.md.

---

## Security

Please consult SECURITY.md for our responsible security vulnerability disclosure process.

---

## License

Copyright (c) 2026 Oracle and/or its affiliates.

Released under the Universal Permissive License v1.0 as shown at  
https://oss.oracle.com/licenses/upl/
