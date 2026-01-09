# ces-importer Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Changed
- [#114] reword dogu and component validation messages
- [#116] default smtp address is now smtp.example.com instead of smtp.mailserver.com

### Added 
- [#116] make mail delivery after import optional
  - added new configuration option in values.yaml: `config.smtp.enabled` with default value `false`
- [#120] add configuration to exclude dogus
  -  added new configuration option in values.yaml: `config.job.excludedDogus` to exclude dogus from migration 

## [v2.1.0] - 2026-01-05
### Changed
- [#112] Stop blueprint during migration

## [v2.0.1] - 2025-12-03
### Fixed 
- [#110] only try to resize pvcs if they have data that needs to be imported

## [v2.0.0] - 2025-10-10
### Changed
- [#104] use cas-dogu for connection-check (nginx-dogu is obsolete)
  - This is a breaking change because it requires the importing system to ces-gateway and ces-assets instead of the nginx-ingress and nginx-static dogus.

### Added
- [#108] add configuration for custom TLS CAs

## [v1.3.0] - 2025-09-30
> [!CAUTION] This release should not be used because of breaking changes. Use v2.0.0 instead.
### Added
- [#108] add configuration for custom TLS CAs

## [v1.2.4] - 2025-09-10
> [!CAUTION] This release should not be used because of breaking changes. Use v2.0.0 instead.
### Changed
- [#104] use cas-dogu for connection-check (nginx-dogu is obsolete)

## [v1.2.3] - 2025-09-03
### Fixed
- [#105] wrong namespace for ignored monitoring dogu

## [v1.2.2] - 2025-08-28
### Changed
- [#102] only migrate alternative FQDNs on fqdn-change in finale migration

## [v1.2.1] - 2025-07-17
### Fixed
- [#100] update fqdn before certificate to prevent race-condition while updating global-config

## [v1.2.0] - 2025-07-15
### Fixed
- [#98] preflight check now works in mn
- [#94] add metadata mapping for logLevel

## [v1.1.1] - 2025-07-14
### Fixed 
- [#96] remove leading slashes from config keys when changing fqdn

## [v1.1.0] - 2025-07-09
### Fixed
- [#88] remove html error pages from error responses
- [#92] Log sync errors when they occur

### Added
- [#86] preflight check can now be configured
- [#94] add metadata mapping for logLevel

### Changed
- [#90] adjust mail-text for migration-result-mail

## [v1.0.1] - 2025-06-23
### Fixed
- [#79] use "false" as default verbose-logging flag
- [#81] retry globalConfig SaveOrMerge for every error instead of single Kubernetes ConflictError

## [v1.0.0] - 2025-06-23
- 🎉🎉 First release 🎉🎉

### Fixed
- [#71] retry writing fqdn to global-config on conflict-error

## [v0.0.6] - 2025-06-20
### Fixed
- [#66] fix missing subject in mails
- [#66] fix mails don't have log files attached
- [#66] fix too long token error when log lines are too long
- [#67] race condition while waiting for pvc resizes
- [#69] always run data and config migration before exiting because of errors
- [#74] use changeFQDN value from configuration

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