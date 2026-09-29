---
name: multi-tenant
metadata:
  author: Mohamed
description: >
  Design, implement, review, and test multi-tenant applications across Node.js
  (Express.js, NestJS, Vue.js) and Go (net/http, Gin). Use when a system must
  isolate tenants, resolve tenant context, enforce tenant-scoped access, or
  support tenant-aware data, caching, jobs, files, and observability.
---

# Multi-Tenant Skill

Build tenant isolation as a security boundary, not merely as a `tenantId` field.
Identify the tenancy model, trust boundary, and enforcement point before writing
code. Prefer central enforcement with explicit tenant context over repeating
ad-hoc checks in handlers or components.

## Operating workflow

1. Establish the tenant model: shared database/shared schema, schema per tenant,
   database per tenant, or a deliberate hybrid.
2. Define the trusted tenant-resolution sources: authenticated claims, mapped
   domain/subdomain, API key, or an explicitly allowed administrative override.
   Never trust a client-supplied tenant identifier without authorization.
3. Map every tenant-bearing resource, including database rows, cache keys,
   object-storage paths, events, queues, logs, metrics, and audit records.
4. Choose one request/context representation and propagate it through handlers,
   services, repositories, background jobs, and outbound calls.
5. Enforce isolation at the lowest practical layer, then add authorization at
   the use-case layer. Reject missing, ambiguous, or mismatched tenant context.
6. Test both positive access and cross-tenant denial, including jobs, caches,
   exports, bulk operations, and privileged workflows.

## Non-negotiable invariants

- Every tenant-scoped read and write has an explicit tenant boundary.
- Tenant identity is authenticated and authorized independently of user identity.
- Repository/query APIs make tenant scope difficult to omit.
- Cache, queue, file, and event identifiers include tenant scope where relevant.
- Background work carries tenant context in its payload and revalidates access.
- Logs and metrics support tenant diagnosis without leaking sensitive data.
- Platform administrators use a separate, audited, least-privilege path.

## Stack routing

Load only the references relevant to the current implementation:

- Data model, isolation, migrations, and threat decisions: `references/architecture.md`.
- Express.js or NestJS backend work: `references/node-backends.md`.
- Vue.js frontend work: `references/vue-frontend.md`.
- Go `net/http` or Gin backend work: `references/go-backends.md`.
- Multi-agent planning, ownership, and review: use `Skills/multi-agents/SKILL.md`.

When multiple stacks are involved, keep the tenant contract consistent across
services and document which service is authoritative for tenant resolution.

## Response modes

- For architecture requests, state the tenancy model, trust boundaries, data
  flow, failure behavior, and a concise folder layout before code.
- For implementation requests, inspect existing authentication, persistence,
  middleware/guards, and testing conventions before changing them.
- For reviews, look specifically for tenant-filter omissions, confused deputy
  paths, unsafe overrides, cache leakage, and context loss in asynchronous work.

