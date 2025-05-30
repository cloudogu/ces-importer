# ces-importer Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Fixed
- [#32] copy sparse files with rsync

## [v0.0.1] - 2025-05-28
### Added
- Initial release
- [#1] Runs as Helm chart and adds basic configuration points as `Values.yaml`
  - adds also crucial data for running in a CI server
- [#4] Start export routine according to exporter endpoints
  - with this feature, the importing system reacts on the exporting system by requesting HTTP endpoints