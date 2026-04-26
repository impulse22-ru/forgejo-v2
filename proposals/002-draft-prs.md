# Proposal 002: Draft Pull Requests

## Summary

Mark pull requests as drafts to indicate work in progress.

## Motivation

- Signal to reviewers that PR is not ready
- Avoid premature reviews
- Better PR workflow

## Proposed Solution

Add draft PR status flag.

### Features

- Draft/Ready toggle
- Auto-convert to ready on push
- Block merge of drafts
- Show draft status in list
- Notification of draft status

### Syntax

```yaml
# GitHub-flavored
- title: "WIP: Add feature X"
- label: "draft"

# Web UI
[ ] This PR is a draft
```

### State Transitions

```
Draft -> Ready -> Review -> Approved -> Merged
  ^__________|
     (mark ready)
```

### Options

| Option | Description |
|--------|-------------|
| Auto-mark ready | Convert after changes |
| Require approval | Block until approved |
| Allow to draft | Allow merge from draft |

### Similar Projects

- GitHub Draft PRs
- GitLab Draft MRs

## Status

**Proposed** - Not yet implemented