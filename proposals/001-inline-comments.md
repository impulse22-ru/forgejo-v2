# Proposal 001: Inline Code Comments

## Summary

Allow commenting on specific lines of code in pull requests.

## Motivation

Reviewers want to comment on specific lines, not just overall PR.

## Proposed Solution

Add inline commenting for diffs.

### Features

- Line-specific comments
- Side-by-side diff view
- Suggestion mode with code proposal
- Conversation threads on lines
- Resolve/unresolve inline comments

### UI Flow

1. Open PR diff view
2. Click on line number
3. Add comment
4. Submit review

### Data Model

```sql
CREATE TABLE pull_request_comments (
    id BIGINT PRIMARY KEY,
    pull_request_id BIGINT NOT NULL,
    commit_sha VARCHAR(40),
    path VARCHAR(255),
    line_start INT,
    line_end INT,
    parent_id BIGINT,
    body TEXT,
    author_id BIGINT,
    created_at TIMESTAMP,
    resolved_at TIMESTAMP
);
```

### API

```json
POST /repos/{owner}/{repo}/pulls/{index}/comments
{
  "commit_id": "abc123",
  "path": "src/main.go",
  "line": 42,
  "body": "Consider using const instead"
}
```

### Similar Projects

- GitHub Inline Comments
- GitLab Mr Comments

## Status

**Proposed** - Not yet implemented