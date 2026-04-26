# Proposal 001: Container Registry

## Summary

Host OCI/Docker container images in ForgeJo.

## Motivation

Host container images alongside source code.

## Proposed Solution

Implement OCI Distribution-compliant registry.

### Endpoints

```
GET /v2/                          # API version
GET /v2/{name}/tags/list           # List tags
GET /v2/{name}/manifests/{ref}     # Get manifest
PUT /v2/{name}/manifests/{ref}     # Push manifest
GET /v2/{name}/blobs/{digest}     # Get blob
POST /v2/{name}/blobs/uploads/     # Start upload
PATCH /v2/{name}/blobs/{ref}      # Upload chunk
```

### Configuration

```yaml
[container_registry]
ENABLED = true
HOST = containers.example.com
```

### Storage

- Local filesystem
- S3-compatible
- Azure Blob
- GCS

### Features

- Docker/OCI manifest support
- Image layering
- Tag management
- Vulnerability scanning
- Access control

### Similar Projects

- GitHub Container Registry
- GitLab Container Registry

## Status

**Proposed** - Not yet implemented