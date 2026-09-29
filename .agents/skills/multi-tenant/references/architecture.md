# Multi-Tenant Architecture Reference

## Isolation models

- Shared database/shared schema: simplest operations and migrations; every
  tenant-scoped table and query needs reliable tenant predicates and constraints.
- Schema per tenant: stronger database boundaries with more provisioning and
  migration complexity.
- Database per tenant: strongest separation and operational isolation; requires
  connection, migration, backup, and routing management.
- Hybrid: use only when the boundary and operational reason are explicit.

Do not present one model as universally correct. Choose based on compliance,
blast radius, tenant count, query volume, operational maturity, and cost.

## Tenant context contract

Represent resolved context with at least:

- `tenantId`: canonical internal identifier, not an arbitrary display name.
- `source`: how it was resolved, retained for audit/debugging where safe.
- `actorId` and roles/scopes.
- `isPlatformAdmin` or an equivalent narrowly authorized capability.
- request/correlation ID and locale only when needed by downstream behavior.

Make context immutable after resolution. A route parameter may identify a
resource, but it must be checked against the resolved tenant rather than
replacing it.

## Enforcement patterns

- Shared-schema SQL: use tenant-aware repository methods, composite unique keys,
  foreign keys, and database row-level security when it fits the database.
- Schema/database routing: resolve and validate the target before selecting a
  connection; never derive credentials or connection names directly from input.
- Cache keys: use a stable format such as `tenant:{tenantId}:resource:{id}` and
  invalidate within the same tenant scope.
- Object storage: use tenant-scoped prefixes and verify ownership before reads,
  signed URLs, copies, or deletes.
- Events/jobs: include tenant ID, actor/capability context, schema version, and
  an idempotency key; re-check authorization at consumption time.

## Failure and security review

Fail closed on absent or conflicting tenant context. Watch for IDOR, confused
deputy behavior, cross-tenant joins, unscoped background tasks, global caches,
tenant data in error messages, and platform-admin bypasses without audit logs.

