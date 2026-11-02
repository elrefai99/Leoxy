## [1.11.2] - 2026-11-02

### Added
- add the restart CLI command
- add load testing tools, live endpoint checks, and cross-platform CPU/RAM monitoring
- add race and end-to-end CI workflows

### Changed
- correct rate limit documentation to use requests per minute

### Fixed
- fix load balancing and rate limit/burst behavior
- fix server issues found during testing, including readiness checks and pprof
- repair build, network test, Docker scripts, and container configurations
- fix load test reporting and success status handling

## [Unreleased]

### Changed
- upgrade Go to 1.27.3 and refresh dependencies
- bump version to 1.9.1

### Fixed
- correct Docker Compose build context and Dockerfile path

## [3.8.2] - 2026-10-26

### Fixed
- fix prefix handling for root and non-root paths
- correct path trimming in proxy handlers
- update version references

# Changelog

## [3.8.0] - 2026-10-23

### Added
- build and publish Docker images through the reusable CD workflow
- limit Docker publishing to pushes on main and use lowercase image names
- create GitHub releases from version.json on main pushes

### Fixed
- match configured literal access paths exactly
