# Changelog and versioning

Every PR updates `CHANGELOG.md` at the repo root and bumps the version. The exception is `docs`, which bumps nothing.

## Contents
- Bump rules
- Steps
- Entry format
- Examples
- Edge cases

## Bump rules

| Type | Bump |
| --- | --- |
| `feat` | minor |
| `fix` | patch |
| `refactor` | patch |
| `chore` | patch |
| `docs` | none |
| breaking (`!`) | major |

For versions below 1.0.0, keep the same mapping unless the repo's history shows otherwise.

## Steps

1. Read the current version from the manifest the repo already uses (`package.json`, `pyproject.toml`, `Cargo.toml`). If none exists, ask the user which version source to use.
2. Apply the bump. For npm projects use `npm version <new> --no-git-tag-version` so the lockfile root version stays in sync. Do not create tags and do not publish.
3. Add the entry at the top of `CHANGELOG.md`, below the header. If the file is missing, create it with a `# Changelog` header.
4. Get the date with `date +%F`. Never guess it.
5. For `docs`, add the entry under `## [Unreleased]` instead of a new version.

## Entry format

Keep a Changelog style. Reuse the same three points as the PR body, word for word.

```
## [1.4.0] - 2026-10-06

### Added
- <change 1>
- <change 2>
- <change 3>
```

Heading by type:

| Type | Heading |
| --- | --- |
| `feat` | Added |
| `fix` | Fixed |
| `refactor` | Changed |
| `chore` | Changed |
| `docs` | Documentation |

## Examples

Fix, current version 2.3.0:

```
## [2.3.1] - 2026-10-06

### Fixed
- skip capture when the payment is already settled
- key webhook retries by provider event id
- add a regression test for duplicate callbacks
```

Docs, no version change:

```
## [Unreleased]

### Documentation
- explain how to create a worktree per branch
- document the cleanup step after merge
- link the setup guide from the contributing page
```

## Edge cases

- Existing changelog uses a different format: match its format and keep the three points.
- Existing `## [Unreleased]` section on a non-docs PR: move its entries into the new version heading, then add this PR's entries.
- Unrelated unreleased version bump already on the base branch: bump from the base branch's version, not from the local one.
- Breaking change: confirm with the user, bump major, and add a `### Breaking` heading above the other headings.
