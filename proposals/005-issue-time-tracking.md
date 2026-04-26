# Proposal 005: Issue Time Tracking

## Summary

Track estimated vs actual time spent on issues.

## Motivation

Teams need time estimation for planning.

## Proposed Solution

Add time tracking fields.

### Features

- Estimated time
- Time spent
- Time tracking comments
- Time reports
- Time charts

### Syntax

```yaml
# In issue body
Time Estimate: 3d
Time Spent: 2h 30m
Remaining: 2d 1h 30m
```

### UI

```
Issue: Implement login
+----------------+----------+
| Estimate       | 3d       |
| Time Spent     | 2h 30m  |
| Remaining     | 2d 1h30m|
+----------------+----------+

| Date       | Author   | Time   |
|-----------|---------|-------|
| Apr 1     | alice   | 1h    |
| Apr 2     | bob     | 1h 30m|
```

### Similar Projects

- Jira Time Tracking
- GitHub / GitLab Time Tracking

## Status

**Proposed** - Not yet implemented