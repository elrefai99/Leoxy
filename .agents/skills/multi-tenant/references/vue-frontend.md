# Vue.js Frontend Reference

The frontend selects or displays the active tenant, but the backend remains the
source of truth for identity and authorization.

- Load the active tenant from an authenticated session or server response.
- Store only the minimum tenant metadata needed for navigation and display.
- Scope client caches, query keys, stores, and persisted state by tenant ID.
- Clear or replace tenant-scoped state on logout and tenant switching.
- Cancel or ignore in-flight requests from the previous tenant.
- Render capabilities returned by the server, not guessed roles from local state.
- Never place secrets or privileged tenant claims in URL parameters or local
  storage unless the application threat model explicitly accepts that risk.

Test switching, refresh, deep links, stale cached data, unauthorized routes,
and concurrent requests across tenants.

