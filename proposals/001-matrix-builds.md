# Proposal 001: Matrix Builds

## Summary

Support matrix strategy for running jobs with multiple combinations.

## Motivation

Run tests across multiple OS, Node versions, Python versions in one workflow.

## Proposed Solution

Extend workflow syntax to support matrix.

### Syntax

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
        node: [14, 16, 18, 20]
        include:
          - os: ubuntu-latest
            node: 20
            coverage: true
    steps:
      - uses: actions/checkout@v3
      - name: Use Node.js ${{ matrix.node }}
        uses: actions/setup-node@v3
        with:
          node-version: ${{ matrix.node }}
      - run: npm test
```

### Features

- Multiple axes
- Exclude combinations
- Include extra combinations
- Fail-fast option
- Matrix result aggregation

### Similar Projects

- GitHub Matrix Strategy
- GitLab Matrix

### Implementation Plan

1. Extend YAML parser for matrix
2. Create matrix expansion logic
3. Add job scheduler support
4. Add UI for matrix results
5. Add include/exclude support

## Status

**Proposed** - Not yet implemented