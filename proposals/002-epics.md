# Proposal 002: Epics

## Summary

Group related issues into epics for large features.

## Motivation

Large features span multiple issues. Epics help organize them.

## Proposed Solution

Add epic hierarchy.

### Features

- Create epics
- Add issues to epics
- Epic progress tracking
- Epic timeline
- Links between epics

### Syntax

```yaml
# In issue body
Epic: Feature X
Epic-Links: 
  - REF-123
```

### UI

```
Epic: User Authentication
+--- Issue: Login page (In Progress)
+--- Issue: OAuth (Done)
+--- Issue: Password reset (Todo)
+--- Issue: 2FA (Todo)
```

### Progress

| Status | Issues |
|--------|--------|
| Complete | 2/4 |
| In Progress | 1/4 |
| Todo | 1/4 |

### Data Model

```sql
CREATE TABLE epics (
    id BIGINT PRIMARY KEY,
    title VARCHAR(200),
    description TEXT,
    repo_id BIGINT,
    milestone_id BIGINT
);

CREATE TABLE epic_issues (
    epic_id BIGINT,
    issue_id BIGINT
);
```

### Similar Projects

- Jira Epics
- GitHub Issues

## Status

**Proposed** - Not yet implemented