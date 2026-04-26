# Proposal 003: Code Scanning

## Summary

Integrate SAST tools for code analysis.

## Motivation

Detect security issues in code.

## Proposed Solution

Integrate code scanning tools.

### Supported Tools

| Language | Tools |
|----------|-------|
| Go | gosec, staticcheck |
| JavaScript | eslint-security |
| Python | bandit |
| Java | spotbugs |
| Ruby | brakeman |

### Configuration

```yaml
[security.code_scanning]
ENABLED = true
TOOLS = gosec,bandit
AUTO_SCAN = true
ALERT_THRESHOLD = HIGH
```

### GitHub Advanced Security

```yaml
# .github/code-scanning.yml
name: Code Scanning
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
jobs:
  codeql:
    uses: forgejo/security-scans/codeql.yml@main
```

### Output

```json
{
  "rule_id": "G304",
  "message": "Potential file inclusion via variable",
  "severity": "MEDIUM",
  "file": "src/main.go",
  "line": 42
}
```

## Status

**Proposed** - Not yet implemented