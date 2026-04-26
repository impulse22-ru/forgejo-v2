# Proposal 003: Custom Issue Statuses

## Summary

Customizable Kanban statuses per project.

## Motivation

Different teams have different workflows.

## Proposed Solution

Allow custom statuses.

### Configuration

```yaml
project_settings:
  issue_statuses:
    - name: Backlog
      color: gray
      order: 0
    - name: To Do  
      color: blue
      order: 1
    - name: In Progress
      color: yellow
      order: 2
    - name: Code Review
      color: purple
      order: 3
    - name: Done
      color: green
      order: 4
    - name: Won't Fix
      color: red
      order: 5
```

### Features

- Custom names
- Custom colors
- Custom order
- Status transitions
- WIP limits

### State Machine

```yaml
transitions:
  Backlog: [To Do]
  To Do: [Backlog, In Progress]
  In Progress: [To Do, Code Review]
  Code Review: [In Progress, Done]
  Done: []
  Won't Fix: []
```

## Status

**Proposed** - Not yet implemented