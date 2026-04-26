package oci

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type Manifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	MediaType   string           `json:"mediaType"`
	Config    Descriptor       `json:"config"`
	Layers    []Descriptor    `json:"layers"`
	Created   time.Time       `json:"created"`
}

type Descriptor struct {
	MediaType string `json:"mediaType"`
	Digest   string `json:"digest"`
	Size    int64  `json:"size"`
}

type Blob struct {
	Digest   string
	Size    int64
	Content []byte
}

type Image struct {
	Name     string
	Tag     string
	Manifest Manifest
	Blobs   []Blob
}

func (m *Manifest) ComputeDigest() string {
	data := fmt.Sprintf("%d%s%d", m.SchemaVersion, m.MediaType, m.Config.Size)
	hash := sha256.Sum256([]byte(data))
	return "sha256:" + hex.EncodeToString(hash[:])
}

type Registry struct {
	images map[string]Image
}

func NewRegistry() *Registry {
	return &Registry{
		images: make(map[string]Image),
	}
}

func (r *Registry) PutImage(name, tag string, manifest Manifest) {
	r.images[name+":"+tag] = Image{
		Name:     name,
		Tag:     tag,
		Manifest: manifest,
	}
}

func (r *Registry) GetImage(name, tag string) (Image, bool) {
	img, ok := r.images[name+":"+tag]
	return img, ok
}

func (r *Registry) ListTags(name string) []string {
	var tags []string
	for key := range r.images {
		if len(key) > len(name)+1 && key[:len(name)] == name {
			tags = append(tags, key[len(name)+1:])
		}
	}
	return tags
}

type BlobStore struct {
	blobs map[string]Blob
}

func NewBlobStore() *BlobStore {
	return &BlobStore{
		blobs: make(map[string]Blob),
	}
}

func (b *BlobStore) Put(digest string, size int64, content []byte) {
	b.blobs[digest] = Blob{
		Digest:   digest,
		Size:    size,
		Content: content,
	}
}

func (b *BlobStore) Get(digest string) (Blob, bool) {
	blob, ok := b.blobs[digest]
	return blob, ok
}

func (b *BlobStore) Exists(digest string) bool {
	_, ok := b.blobs[digest]
	return ok
}