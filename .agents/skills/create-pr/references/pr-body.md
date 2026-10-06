# PR body

## Contents
- Template
- Three-bullet rules
- Examples by type
- Verification line

## Template

```
## Summary
- <change 1>
- <change 2>
- <change 3>

closes #123

## Version
<old> -> <new>, see CHANGELOG.md

## Verification
<exact commands and results, plus anything unverified>
```

If the repo has a PR template, keep its headings and fill them with the same content. Omit the issue line when there is none. Docs PRs show `no version change` under `## Version`.

## Three-bullet rules

- Always exactly three bullets.
- More than three changes: group them by behavior, not by file.
- Fewer than three: split by reason (what changed, why this approach, what it affects). Do not pad with restatements.
- One sentence per bullet. State the changed behavior and, where it matters, why.
- Never a raw file list.
- Lowercase, imperative, no trailing period, same as the title.
- The same three bullets go into `CHANGELOG.md`.

## Examples by type

`feat(auth): add refresh token rotation`

- issue a new refresh token on every refresh and revoke the previous one
- store token families in redis so reuse of a revoked token invalidates the family
- expose rotation window through the existing auth config

`chore(deps): bump bullmq and release 2.3.1`

- bump bullmq to the latest minor to pick up the stalled job fix
- raise the package version to 2.3.1
- refresh the lockfile

`refactor(listings): split search service into query and ranking`

- extract query building into its own module with no behavior change
- move ranking weights into a typed config object
- update call sites and tests to the new module boundaries

`fix(payments): avoid duplicate capture on webhook retry`

- skip capture when the payment is already settled
- key webhook retries by provider event id
- add a regression test for duplicate callbacks

`docs: clarify worktree setup`

- explain how to create a worktree per branch
- document the cleanup step after merge
- link the setup guide from the contributing page

## Verification line

- List the exact commands run and the result of each.
- State what was not verified, for example "not tested against the staging payment gateway".
- Do not claim more than the commands prove.
