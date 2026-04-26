# Proposal 001: Audit Log UI

## Summary

Web interface for browsing audit logs.

## Motivation

Admins need easy access to audit logs.

## Proposed Solution

Add admin UI for audit logs.

### Features

- Searchable log view
- Filter by action
- Filter by user
- Filter by date range
- Export to CSV/JSON
- Log retention settings

### UI

```
Audit Logs
+------------------------------------------+
| Search: [________________] [Search]       |
| Filter: [Action v] [User v] [Date v]    |
+------------------------------------------+
| Time     | User | Action    | Details    |
|----------|------|------------|------------|
| 10:30    | alice| repo.create| my-repo    |
| 10:25    | bob  | user.login |            |
| 10:20    | alice| issue.close| #123       |
+------------------------------------------+
```

### Data Model

```sql
CREATE TABLE audit_log (
    id BIGINT PRIMARY KEY,
    created_at TIMESTAMP,
    user_id BIGINT,
    action VARCHAR(50),
    details JSONB,
    ip VARCHAR(45)
);
```

### Similar Projects

- GitHub Audit Log
- GitLab Audit Events

## Status

**Proposed** - Not yet implemented