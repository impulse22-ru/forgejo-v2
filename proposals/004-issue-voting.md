# Proposal 004: Issue Voting

## Summary

Allow users to vote on issues to prioritize.

## Motivation

Community feedback helps prioritize work.

## Proposed Solution

Add issue voting feature.

### Features

- Upvote issues
- Vote count display
- Sort by votes
- Vote notifications
- Vote limits

### UI

```
[+15] Bug: Application crashes on startup
      -------------------------
      Started by @alice, +14 others
      
      Votes: ████████████████░░ 15
```

### API

```json
POST /repos/{owner}/{repo}/issues/{index}/votes
{
  "direction": "up"
}

DELETE /repos/{owner}/{repo}/issues/{index}/votes
```

### Sorting

```
?state=open&sort=votes&direction=desc
```

### Similar Projects

- Feature requests in various platforms

## Status

**Proposed** - Not yet implemented