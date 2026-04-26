# Developer Experience

This module adds improvements for developer workflow and code review.

## Features

| # | Feature | Description |
|---|---------|-------------|
| 1 | Inline code comments | Comment on specific lines in PR |
| 2 | Draft PRs | Work in progress PRs |
| 3 | PR auto-complete | Auto-merge when checks pass |
| 4 | CODEOWNERS improvements | Enhanced CODEOWNERS |
| 5 | Review assignment | Auto-assign reviewers |

## Getting Started

```bash
ls proposals/
ls stubs/
```

## Architecture

```
devexp/
├── proposals/     # Feature specifications
├── stubs/       # Code placeholders
└── README.md    # This file
```

## See Also

- [ForgeJo PR Documentation](https://forgejo.org/docs/latest/usage/pull/)
- [GitHub CODEOWNERS](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners)