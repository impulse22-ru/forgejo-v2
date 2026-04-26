# Proposal 002: Scheduled Workflows

## Summary

Run workflows on a schedule using cron syntax.

## Motivation

Scheduled tasks like daily builds, weekly releases, periodic cleanup.

## Proposed Solution

Add schedule trigger to workflows.

### Syntax

```yaml
on:
  schedule:
    - cron: '0 0 * * *'        # Daily at midnight
    - cron: '0 0 * * 0'        # Weekly on Sunday
    - cron: '0 0 1 * *'        # Monthly on 1st
    - cron: '0 0 * * 5'       # Every Friday at midnight
```

### Features

- Cron expression support
- Multiple schedules
- Timezone configuration
- Manual trigger option

### Configuration

```yaml
jobs:
  cleanup:
    runs-on: ubuntu-latest
    steps:
      - name: Cleanup old artifacts
        run: |
          # cleanup logic
```

### Similar Projects

- GitHub Scheduled Triggers
- GitLab Pipelines Schedules

## Status

**Proposed** - Not yet implemented