## [Unreleased]

### Added
- add an Astro landing page with self-hosted fonts, brand assets, and SEO metadata
- publish Starlight guides for installation, configuration, usage, runtime reference, and FAQ
- build and deploy the website to GitHub Pages with a website path filter

## [1.13.2] - 2026-11-05

## Added
- add total RAM tracking for different OS platforms and update sample structure ([a392c8e](https://github.com/elrefai99/Leoxy/commit/a392c8e84705cf41ac93080ae3cd2a6a3412584d))
- update Dockerfiles for Alpine and Windows, add health checks and user permissions ([6cb1a8d](https://github.com/elrefai99/Leoxy/commit/6cb1a8d9af67e9e3f7cdef543a1ded33d11cd9d9))
- add version 1.12.2 and update versions documentation ([6b9d94e](https://github.com/elrefai99/Leoxy/commit/6b9d94e9172a6ce74b29b0002d2f1cd1028cb5b3))
- update configuration file path and add new leoxy_config.yaml ([3ef70f3](https://github.com/elrefai99/Leoxy/commit/3ef70f31e8cfdecf9fa1c913f93fd1e6827d345a))
- add benchmark tools for Leoxy with comprehensive metrics and profiling ([2cf9c4b](https://github.com/elrefai99/Leoxy/commit/2cf9c4b2eb4f85a61c3cfc8b1dd514b9a950ef8a))
- implement watcher functionality with memory monitoring and reporting ([b93459c](https://github.com/elrefai99/Leoxy/commit/b93459cd926ac0c958b2d728e750b92c7e353b1d))
- add timeout configuration to default server settings ([a31c990](https://github.com/elrefai99/Leoxy/commit/a31c99098918492c5489c50ff4515f78d0448c4c))
- remove timeout configuration from default server settings ([0d45bf9](https://github.com/elrefai99/Leoxy/commit/0d45bf91e1047c44f13fca695940ae7c08ffb148))
- implement configurable timeouts for server connections and requests ([f5e3e50](https://github.com/elrefai99/Leoxy/commit/f5e3e501541e702752452e42143bd4f350effa9c))
- initialize watcher package ([7765ae8](https://github.com/elrefai99/Leoxy/commit/7765ae8dbf4559bab6fd5351f09ba21e570f6745))
- add issue templates for bug reports and feature requests ([4f50524](https://github.com/elrefai99/Leoxy/commit/4f505247f354ffe9a7c83a7a7c3b12950a61c88b))

## Fixed

- add Redis connection configuration and update rate limiting functions ([6e003a3](https://github.com/elrefai99/Leoxy/commit/6e003a36e6c9e3ae92edc5ccc606dbaf557d59bc))
- enhance error handling in watcher report and webhook functions ([163dc73](https://github.com/elrefai99/Leoxy/commit/163dc73131af69794ff2944a6f67c8f460e6d255))
- enhance issue templates with clearer descriptions and options ([403efe8](https://github.com/elrefai99/Leoxy/commit/403efe8cba6a57f0e0e3c1463acf87d18f412ec8))

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
