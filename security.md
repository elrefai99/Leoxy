# Leoxy security notes

## Supported versions

Only the latest release receives security fixes.

## Reporting a vulnerability

Do not open a public issue.

Report privately through [GitHub security advisories](https://github.com/elrefai99/Leoxy/security/advisories/new).
If that is unavailable, email elrefai99@gmail.com.

Include:
- Leoxy version (`leoxy version`) and deployment method
- Redacted `leoxy/config.yaml`
- Steps to reproduce and observed impact

Redact `jwt_secret`, `api_keys`, `redis_password`, and private upstream URLs.

## Response

- Acknowledgement within 72 hours
- Status update within 7 days
- Fix and advisory published after a patch release is available

## Scope

In scope:
- Authentication bypass (API key, JWT, mTLS when Leoxy terminates TLS)
- Rate limit, body limit, or CIDR allow/deny bypass
- Client IP spoofing through X-Forwarded-For, X-Real-IP, or CF-Connecting-IP from a peer that is not in `trusted_proxy_cidrs`
- Access to `/metrics`, `/health/ready`, or `/leoxy` from outside loopback and `internal_cidrs`
- Path prefix or path matching flaws that expose unintended routes
- Secret exposure through logs, metrics, or error responses
- Unbounded resource use in TCP/UDP proxying despite the connection and session limits

Out of scope:
- Exposing Leoxy directly to the internet without a TLS terminator or network controls
- `require_mtls` behind a terminator that does not forward client certificates
- Weak or committed secrets, or world-readable config files
- Empty or overly broad `trusted_proxy_cidrs`, and empty `internal_cidrs` behind a proxy
- `redis_fail_open: true` admitting traffic while Redis is down
- TCP/UDP listeners without `allow_cidrs`, and UDP source spoofing; stream listeners bypass HTTP authentication by design
- `/ping` and `/health/live`, which are public by design
- Vulnerabilities in upstream applications or dependencies without a Leoxy-specific impact
- Denial of service through traffic volume alone
