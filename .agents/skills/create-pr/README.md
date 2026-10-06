# create-pr

An agent skill that opens a GitHub pull request from the current branch with a typed title, a three-point body, a version bump, and a changelog entry. Modeled on [antfu-create-pr](https://github.com/antfu/skills/blob/main/skills/antfu-create-pr/SKILL.md).

## What it does

- Picks the PR type from the diff: `feat`, `chore`, `refactor`, `fix`, or `docs`.
- Asks one question before using `fix`: is this a live production bug?
- Writes a title in Conventional Commits style (lowercase, imperative, no trailing period, under ~70 characters).
- Writes a body with exactly three summary bullets, the version change, and a verification line.
- Bumps the version and adds the same three points to `CHANGELOG.md`.
- Pushes the branch and runs `gh pr create`.

## Types

| Type | Use for | Rule | Bump |
| --- | --- | --- | --- |
| `feat(scope)` | New features | Any file count | minor |
| `chore(scope)` | Small updates and version changes | Max 2 files | patch |
| `refactor(scope)` | Many code changes or updates | Min 3 files, up to the whole project | patch |
| `fix(scope)` | Bug fix in live production | User confirms it is live | patch |
| `docs` | Documentation only | No scope | none |

## Structure

```
create-pr/
├── SKILL.md
├── README.md
└── references/
    ├── type-selection.md
    ├── changelog-and-versioning.md
    └── pr-body.md
```

`SKILL.md` holds the workflow and rules. The reference files load only when the workflow reaches the matching step.

## Requirements

- `git` and the GitHub CLI (`gh`), authenticated with `gh auth login`.
- A push remote for the current branch.
- A version source in the repo (`package.json`, `pyproject.toml`, or `Cargo.toml`). The skill asks if none is found.

## Install

Claude.ai: upload `create-pr.skill` from Settings, Skills.

Claude Code: copy the folder to `~/.claude/skills/create-pr/` for all projects, or `.claude/skills/create-pr/` for one repo.

## Usage

Ask the agent in plain language, for example:

- "open a PR for this branch"
- "push this and make a PR"
- "write the PR for these changes"

Example result for a bug fix confirmed as live:

```
Title: fix(payments): avoid duplicate capture on webhook retry

## Summary
- skip capture when the payment is already settled
- key webhook retries by provider event id
- add a regression test for duplicate callbacks

closes #482

## Version
2.3.0 -> 2.3.1, see CHANGELOG.md

## Verification
npm test -- payments: 14 passed
npm run typecheck: no errors
Not tested against the staging gateway.
```

## Customizing

- Change the type rules or bump table in `SKILL.md` and `references/type-selection.md`.
- Change the changelog headings in `references/changelog-and-versioning.md`.
- Change the body shape in `references/pr-body.md`. A repo PR template always takes precedence over it.

## Limits

- File count is mechanical, so a 2-file change that rewrites a core algorithm still lands as `chore`. Override the type when you disagree.
- Version and changelog edits in every PR conflict when PRs run in parallel. For busy repos, consider changesets or release-please instead.
