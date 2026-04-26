# Proposal 003: Reusable Workflows

## Summary

Create reusable workflow templates to avoid duplication.

## Motivation

DRY principle - avoid copying same jobs across workflows.

## Proposed Solution

Allow referencing other workflow files.

### Syntax

```yaml
# .github/workflows/call-test.yml
name: Test Workflow

on:
  workflow_call:
    inputs:
      node_version:
        type: string
        default: '18'
    secrets:
      NPM_TOKEN:
        required: true

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup Node
        uses: actions/setup-node@v3
        with:
          node-version: ${{ inputs.node_version }}
      - run: npm test
```

### Calling Reusable Workflow

```yaml
# .github/workflows/ci.yml
name: CI

on: [push, pull_request]

jobs:
  test-node18:
    uses: ./.github/workflows/call-test.yml
    with:
      node_version: '18'
    secrets:
      NPM_TOKEN: ${{ secrets.NPM_TOKEN }}

  test-node20:
    uses: ./.github/workflows/call-test.yml
    with:
      node_version: '20'
    secrets:
      NPM_TOKEN: ${{ secrets.NPM_TOKEN }}
```

### Features

- Input parameters
- Secrets passing
- Output variables
- Output artifacts

## Status

**Proposed** - Not yet implemented