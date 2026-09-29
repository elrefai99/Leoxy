# Go Backend Reference

## net/http

Resolve tenant context in middleware after authentication and pass an immutable
context value down the request chain. Define typed context keys in one package,
avoid string keys, and make missing context an explicit error. Keep handlers
from reading raw headers or claims repeatedly.

## Gin

Use middleware to resolve and validate tenant context, then expose it through a
typed helper or a controlled `gin.Context` key. Treat values from `gin.Context`
as request-scoped and do not copy mutable context into goroutines without a
deliberate immutable snapshot.

## Go-specific concerns

- Pass tenant scope explicitly to repositories and use cases where practical.
- Never store tenant context in package globals.
- Propagate context with `context.Context` to database calls and outbound HTTP.
- Ensure goroutines, workers, and scheduled tasks receive tenant context in their
  job payload rather than depending on the originating request.
- Include tenant scope in cache keys, object paths, logs, and metrics labels only
  when cardinality and privacy are acceptable.
- Test middleware, authorization, query scoping, cancellation, and worker paths.

