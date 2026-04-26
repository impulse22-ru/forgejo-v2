# Proposal 004: Composite Actions

## Summary

Create custom composite actions with multiple steps.

## Motivation

Build reusable action components without Docker/JS.

## Proposed Solution

Allow combining multiple steps in one action.

### Syntax

```yaml
# action.yml
name: 'Setup Node & Cache'
description: 'Setup Node.js with caching'

inputs:
  node-version:
    description: 'Node version'
    required: true
    default: '18'

outputs:
  cache-hit:
    description: 'Whether cache was hit'

runs:
  using: 'composite'
  steps:
    - name: Setup Node
      uses: actions/setup-node@v3
      with:
        node-version: ${{ inputs.node-version }}

    - name: Cache npm
      id: cache-npm
      uses: actions/cache@v3
      with:
        path: ~/.npm
        key: ${{ runner.os }}-npm-${{ hashFiles('**/package-lock.json') }}
        restore-keys: |
          ${{ runner.os }}-npm-

    - name: Set output
      shell: bash
      run: echo "cache-hit=${{ steps.cache-npm.outputs.cache-hit }}" >> $GITHUB_OUTPUT
```

### Usage

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup environment
        uses: ./actions/setup-node@v4
        with:
          node-version: '18'
```

### Features

- Multiple steps in one action
- Input/output parameters
- Environment variables
- Conditional steps

## Status

**Proposed** - Not yet implemented