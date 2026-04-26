# Proposal 004: Package Vulnerability Scanning

## Summary

Scan packages for known vulnerabilities.

## Motivation

Security vulnerabilities in dependencies need detection.

## Proposed Solution

Implement vulnerability scanner.

### Features

- CVE database integration
- Lockfile scanning
- Auto-scan on push
- Security advisory alerts
- Fix suggestions

### Supported Ecosystems

| Ecosystem | Lockfile |
|----------|---------|
| npm | package-lock.json |
| pip | Pipfile.lock |
| go | go.mod |
| cargo | Cargo.lock |
| nuget | packages.config |

### Configuration

```yaml
[security.vulnerability]
ENABLED = true
DATABASE = ghsa
AUTO_SCAN = true
```

### Output

```json
{
  "vulnerability": {
    "id": "GHSA-1234-abcd-5678",
    "package": "lodash",
    "severity": "HIGH",
    "cve": "CVE-2024-1234",
    "fix_version": "4.17.21"
  }
}
```

### Similar Projects

- GitHub Dependabot
- Snyk
- npm audit

## Status

**Proposed** - Not yet implemented