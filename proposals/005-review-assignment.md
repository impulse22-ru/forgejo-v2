# Proposal 005: Review Assignment Rules

## Summary

Automatically assign reviewers based on rules.

## Motivation

Streamline review process with automatic assignment.

## Proposed Solution

Add rule-based reviewer assignment.

### Rules

```yaml
# .github/review-assignment.yml
rules:
  - name: security
    condition:
      changed_files:
        - '**/auth*.go'
        - '**/password*'
        - '**/crypto*'
    reviewers:
      - team: security-team
        required: 2

  - name: backend
    condition:
      changed_files:
        - '**/*.go'
    reviewers:
      - user: alice
      - user: bob

  - name: size
    condition:
      additions: > 400
    reviewers:
      - team: senior-team
```

### Features

- File-based rules
- Size-based rules
- Path rules
- Time-based rules
- Round-robin selection

### Priority

1. Exact match rules
2. Wildcard rules  
3. Default rules
4. CODEOWNERS fallback

### Similar Projects

- GitHub Reviewers
- GitLab Reviewers

## Status

**Proposed** - Not yet implemented