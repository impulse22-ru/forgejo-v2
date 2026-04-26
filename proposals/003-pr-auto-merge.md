# Proposal 003: PR Auto-Complete

## Summary

Automatically merge PRs when all checks pass and reviews are approved.

## Motivation

Reduce manual work for maintainers. Auto-merge when ready.

## Proposed Solution

Add auto-merge functionality.

### Syntax

```yaml
# In PR body
- [x] Enable auto-merge
- Auto-mergemethod: merge|squash|rebase

# Or via API
PUT /repos/{owner}/{repo}/pulls/{index}/merge
{
  "do": "merge",
  "auto_merge": true
}
```

### Configuration

| Setting | Description |
|---------|-------------|
| Auto-merge | Enable auto-merge |
| Delete branch | Delete after merge |
| Merge method | merge/squash/rebase |
| Required reviews | Number of approvals |
| Required checks | Passing status checks |

### Requirements

- All required reviews approved
- All required checks passing
- No conflicts
- No blocking reviews

### Webhook Events

```json
{
  "event": "pull_request",
  "action": "auto_merge",
  "pull_request": {
    "number": 1,
    "merged": true
  }
}
```

### Similar Projects

- GitHub AutoMerge
- GitLab Auto-MR

## Status

**Proposed** - Not yet implemented