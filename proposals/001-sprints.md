# Proposal 001: Sprint Planning

## Summary

Add sprint and iteration planning to track work in progress.

## Motivation

Teams need agile planning with sprints.

## Proposed Solution

Implement sprint/iteration system.

### Features

- Create sprints
- Assign issues to sprints
- Sprint burndown chart
- Sprint velocity
- Backlog management

### UI

```
Sprint: Sprint 23 (Apr 21 - May 4)
+----------------------------------+
| To Do    | In Progress | Done    |
+----------+------------+--------+
| Issue 123| Issue 124  | Issue 1 |
| Issue 125|           |        |
+----------+------------+--------+
```

### Data Model

```sql
CREATE TABLE sprints (
    id BIGINT PRIMARY KEY,
    name VARCHAR(100),
    start_date DATE,
    end_date DATE,
    goal TEXT,
    repo_id BIGINT
);

CREATE TABLE sprint_issues (
    sprint_id BIGINT,
    issue_id BIGINT,
    position INT
);
```

### Similar Projects

- GitHub Projects
- Jira Sprints

## Status

**Proposed** - Not yet implemented