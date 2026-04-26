# Proposal 002: Conan Package Support

## Summary

Add Conan C++ package manager support.

## Motivation

C++ developers need package management.

## Proposed Solution

Implement Conan registry.

### Endpoints

```
GET /conan/v2/conans/{name}/{version}/{channel}/latest
POST /conan/v2/conans/{name}/{version}/{channel}/upload
GET /conan/v2/conans/{name}/{version}/{channel}/download
```

### Configuration

```yaml
[packages.conan]
ENABLED = true
```

### Conanfile Support

```
[requires]
mylib/1.0.0

[generators]
cmake_find_package
```

### Similar Projects

- JFrog Artifactory
- Conan Center

## Status

**Proposed** - Not yet implemented