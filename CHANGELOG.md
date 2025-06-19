# ces-importer Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Fixed
- [#67] race condition while waiting for pvc resizes

## [v0.0.5] - 2025-06-18
### Fixed
- [#60] remove config values from logs
- [#61] increase timout when waiting for migration job logs to 500 seconds
- [#63] use installed version instead of spec version when validating dogus

## [v0.0.4] - 2025-06-16
### Fixed
- [#50] do not send mails, when no mail-server is configured
- [#52] wait for requested minimal data volume size when resizing pvcs
- [#55] notify user when migration job gets deleted
- [#57] use configured value for verbose flag instead of hardcoded value

### Added
- [#50] add "Date"-header to mail

### Changed
- [#54] stream logs and log them directly in the coordinator
- [#54] cleanup migration job 60 seconds after completion
- [#57] support multiple exclude patterns in the values.yaml for the job configuration

## [v0.0.3] - 2025-06-12
### Fixed
- [#48] add permissions for the service account to handle backup schedules
- [#48] wait for backup schedule to be deleted to create a new one

## [v0.0.2] - 2025-06-05
### Fixed
- [#32] copy sparse files with rsync
- [#38] stop dogus after system-validation
  - This is needed to wait for dogu-volume-resizes to complete

### Changed
- [#26] retrieve mail server password from a secret
- [#26] include a Message-ID in emails

## [v0.0.1] - 2025-05-28
### Added
- Initial release
- [#1] Runs as Helm chart and adds basic configuration points as `Values.yaml`
  - adds also crucial data for running in a CI server
- [#4] Start export routine according to exporter endpoints
  - with this feature, the importing system reacts on the exporting system by requesting HTTP endpoints