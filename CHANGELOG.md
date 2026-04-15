# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

### Added

- Protocol-capable Modbus clients for TCP / RTU over TCP / RTU / ASCII.
- PLC clients for Omron FINS and Allen-Bradley with retry/reconnect behavior.
- Siemens client implementation based on `gos7` for core S7 read/write operations.
- Failure-injection tests for protocol exception paths, CRC/LRC checks, malformed responses, and reconnect retries.
- GitHub project metadata: CI workflow, issue templates, PR template, contributing/security docs.

### Changed

- Upgraded protocol and parity documentation to include method-level support coverage.
- Expanded API and README examples for practical client usage.

### Removed

- Legacy PLC stub-only implementation file in favor of concrete client implementations.
