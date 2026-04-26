# Proposal 005: Artifacts Retention Policies

## Summary

Configure automatic cleanup of old workflow artifacts.

## Motivation

Storage costs accumulate with old artifacts. Auto-cleanup needed.

## Proposed Solution

Add retention configuration.

### Configuration

#### Repository Level

```yaml
# .github/artifact-retention.yml
artifacts:
  retention-days: 90
  retention-minimum: 7
  retention-maximum: 180
```

#### Workflow Level

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test" > artifact.txt
      - uses: actions/upload-artifact@v3
        with:
          name: my-artifact
          path: artifact.txt
          retention-days: 30
```

### Features

- Global default retention
- Per-artifact retention
- Minimum retention boundary
- Maximum retention cap
- Retention reports

### Similar Projects

- GitHub Artifacts Retention
- GitLab Artifacts Expiration

### Implementation Plan

1. Add retention configuration schema
2. Create cleanup scheduler
3. Add UI for retention settings
4. Add retention reports
5. Add notifications

## Status

**Proposed** - Not yet implemented