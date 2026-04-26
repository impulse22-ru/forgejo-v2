# Proposal 004: IP Allowlist

## Summary

Restrict access by IP address.

## Motivation

Enterprise needs IP-based access control.

## Proposed Solution

Add IP allowlist feature.

### Configuration

```yaml
[security.ip_allowlist]
ENABLED = true
ALLOWED_IPS:
  - 192.168.1.0/24
  - 10.0.0.0/8
  - 203.0.113.1
```

### Per-Organization

```yaml
org_settings:
  ip_allowlist:
    - 10.0.0.0/8
    - 172.16.0.0/12
```

### Features

- CIDR notation
- Per-repo/org settings
- VPN support
- Audit log entries
- 2FA bypass option

### Similar Projects

- GitHub IP Allowlist
- GitLab Admin Settings

## Status

**Proposed** - Not yet implemented