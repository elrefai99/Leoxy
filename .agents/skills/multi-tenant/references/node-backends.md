# Node.js Backend Reference

## Express.js

Resolve tenant context after authentication and before application routes.
Attach a typed, immutable context to the request or an established request
context mechanism. Keep handlers thin and pass tenant scope into services and
repositories. Do not let repositories infer tenant identity from mutable global
state.

Validate domain/subdomain mappings server-side, normalize hostnames, and reject
unknown or ambiguous mappings. For route-level tenant IDs, require equality with
the resolved context unless the caller has an audited platform capability.

## NestJS

Use guards for authentication and tenant-resolution authorization, then expose
the resolved context through a request-scoped provider or a carefully bounded
request context. Keep tenant predicates in repositories/query services rather
than relying only on controllers. Interceptors can add safe audit metadata, but
must not be the sole isolation mechanism.

## Shared Node concerns

- Do not use process-global mutable `currentTenant` state.
- Propagate context through queues, scheduled jobs, and outbound requests.
- Include tenant scope in cache keys and deduplicate/idempotency keys.
- Test middleware/guards, service authorization, repository queries, and async
  consumers separately.
- For Vue-backed APIs, expose only tenant capabilities and safe summaries; do
  not use frontend tenant state as an authorization source.

