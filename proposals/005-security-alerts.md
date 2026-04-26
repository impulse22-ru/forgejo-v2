# Proposal 005: Security Alerts

## Summary

Repository-level security vulnerability alerts.

## Motivation

Notify users about vulnerabilities in their dependencies.

## Proposed Solution

Implement security advisory system.

### Alert Types

| Type | Source |
|------|--------|
| Dependency | GitHub Advisory DB |
| Secret | Secret Scanning |
| Code | Code Scanning |
| CVE | NVD Database |

### Configuration

```yaml
[security.alerts]
ENABLED = true
NOTIFY_STARTERS = true
NOTIFY_WATCHERS = true
NOTIFY_EMAIL = true
```

### UI

```
Security Alerts
+------------------------------------------+
| [Critical] CVE-2024-1234                 |
| Package: lodash < 4.17.21                  |
| Fix: Update to 4.17.21                    |
+------------------------------------------+
| [High] Secret detected                    |
| Type: AWS Key                             |
| File: config/prod.yml:42                 |
+------------------------------------------+
```

### Similar Projects

- GitHub Security Alerts
- GitLab Security Dashboard

## Status

**Proposed** - Not yet implemented