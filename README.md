# Registry & Packages

This module adds package registry and vulnerability scanning features.

## Features

| # | Feature | Description |
|---|---------|-------------|
| 1 | Container Registry | OCI/Docker image hosting |
| 2 | Conan support | C++ package manager |
| 3 | Maven proxy | Proxy for Maven Central |
| 4 | Vuln scanning | Package vulnerability detection |
| 5 | Dependabot | Auto-update dependencies |

## Getting Started

```bash
ls proposals/
ls stubs/
```

## Architecture

```
registry-packages/
├── proposals/     # Feature specifications
├── stubs/       # Code placeholders
└── README.md    # This file
```

## See Also

- [ForgeJo Packages](https://forgejo.org/docs/latest/usage/packages/)
- [OCI Distribution Spec](https://github.com/opencontainers/distribution-spec)