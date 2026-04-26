package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

var (
	ErrBlobNotFound    = fmt.Errorf("blob not found")
	ErrManifestNotFound = fmt.Errorf("manifest not found")
	ErrInvalidName    = fmt.Errorf("invalid name")
	ErrInvalidTag     = fmt.Errorf("invalid tag")
)

type Manifest struct {
	SchemaVersion int        `json:"schemaVersion"`
	MediaType   string      `json:"mediaType"`
	Config     *Descriptor `json:"config"`
	Layers     []*Descriptor `json:"layers"`
	Created    time.Time   `json:"created"`
}

func (m *Manifest) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *Manifest) Unmarshal(data []byte) error {
	return json.Unmarshal(data, m)
}

type Descriptor struct {
	MediaType string `json:"mediaType"`
	Digest   string `json:"digest"`
	Size    int64  `json:"size"`
}

type Blob struct {
	Digest   string
	Size    int64
	Content io.ReadCloser
}

type Image struct {
	Name   string
	Tag    string
	Manifest *Manifest
}

func (i *Image) String() string {
	return fmt.Sprintf("%s:%s", i.Name, i.Tag)
}

func (i *Image) Digests() []string {
	var digests []string
	for _, l := range i.Manifest.Layers {
		digests = append(digests, l.Digest)
	}
	return digests
}

type TagList struct {
	Name string   `json:"name"`
	Tags []Tag  `json:"tags"`
}

type Tag struct {
	Name     string    `json:"name"`
	Digest   string    `json:"digest"`
	Size    int64     `json:"size"`
	Created time.Time `json:"created"`
}

type Registry struct {
	storage BlobStore
	manifests ManifestStore
	tags    TagStore
}

func NewRegistry() *Registry {
	return &Registry{
		storage:   NewBlobStore(),
		manifests: NewManifestStore(),
		tags:    NewTagStore(),
	}
}

func (r *Registry) HandleBlobUpload(w http.ResponseWriter, req *http.Request) {
	name := getName(req.URL.Path)
	if name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}

	uploadURL := fmt.Sprintf("/v2/%s/blobs/uploads", name)
	switch req.Method {
	case "GET":
		http.Redirect(w, req, uploadURL, http.StatusFound)
	case "POST":
		uploadID := uuid.MustV4(uuid.NewV4()).String()
		url := fmt.Sprintf("%s?upload=%s", uploadURL, uploadID)
		w.Header().Set("Location", url)
		w.Header().Set("Docker-Upload-UUID", uploadID)
		w.WriteHeader(http.StatusAccepted)
	}
}

func (r *Registry) HandleBlob(w http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.URL.Path, "/blobs/")
	if len(parts) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	digest := parts[1]

	blob, err := r.storage.Get(digest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", blob.Size))
	io.Copy(w, blob.Content)
}

func (r *Registry) HandleManifest(w http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.URL.Path, "/manifests/")
	if len(parts) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	ref := parts[1]

	manifest, err := r.manifests.Get(ref)
	if err != nil {
		http.Error(w, "manifest not found", http.StatusNotFound)
		return
	}

	data, _ := json.Marshal(manifest)
	w.Header().Set("Content-Type", manifest.MediaType)
	w.Header().Set("Docker-Content-Digest", ref)
	w.Write(data)
}

func (r *Registry) HandleTags(w http.ResponseWriter, req *http.Request) {
	name := getName(req.URL.Path)
	tags, _ := r.tags.List(name)

	response := TagList{
		Name: name,
		Tags: tags,
	}

	data, _ := json.Marshal(response)
	w.Write(data)
}

func (r *Registry) HandleAPI(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/v2/":
		w.Write([]byte(`{"versions": ["v2"]}`))
	case "/v2/" + req.URL.Path:
		r.route(w, req)
	default:
		http.NotFound(w, req)
	}
}

func (r *Registry) route(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/v2/")

	switch {
	case strings.HasPrefix(path, "/blobs/uploads"):
		r.HandleBlobUpload(w, req)
	case strings.HasPrefix(path, "/blobs/"):
		r.HandleBlob(w, req)
	case strings.HasPrefix(path, "/manifests/"):
		r.HandleManifest(w, req)
	case strings.HasSuffix(path, "/tags/list"):
		r.HandleTags(w, req)
	default:
		http.NotFound(w, req)
	}
}

type BlobStore struct {
	blobs map[string]*Blob
}

func NewBlobStore() BlobStore {
	return BlobStore{
		blobs: make(map[string]*Blob),
	}
}

func (s *BlobStore) Put(digest string, size int64, content io.Reader) error {
	s.blobs[digest] = &Blob{
		Digest: digest,
		Size:  size,
	}
	return nil
}

func (s *BlobStore) Get(digest string) (*Blob, error) {
	blob, ok := s.blobs[digest]
	if !ok {
		return nil, ErrBlobNotFound
	}
	return blob, nil
}

func (s *BlobStore) Delete(digest string) error {
	delete(s.blobs, digest)
	return nil
}

func (s *BlobStore) Exists(digest string) bool {
	_, ok := s.blobs[digest]
	return ok
}

type ManifestStore struct {
	manifests map[string]*Manifest
}

func NewManifestStore() ManifestStore {
	return ManifestStore{
		manifests: make(map[string]*Manifest),
	}
}

func (s *ManifestStore) Put(ref string, manifest *Manifest) error {
	s.manifests[ref] = manifest
	return nil
}

func (s *ManifestStore) Get(ref string) (*Manifest, error) {
	manifest, ok := s.manifests[ref]
	if !ok {
		return nil, ErrManifestNotFound
	}
	return manifest, nil
}

func (s *ManifestStore) Delete(ref string) error {
	delete(s.manifests, ref)
	return nil
}

type TagStore struct {
	tags map[string][]Tag
}

func NewTagStore() TagStore {
	return TagStore{
		tags: make(map[string][]Tag),
	}
}

func (s *TagStore) Add(name, tag, digest string, size int64) {
	s.tags[name] = append(s.tags[name], Tag{
		Name:    tag,
		Digest:  digest,
		Size:   size,
		Created: time.Now(),
	})
}

func (s *TagStore) List(name string) []Tag {
	return s.tags[name]
}

func (s *TagStore) Get(name, tag string) (Tag, bool) {
	for _, t := range s.tags[name] {
		if t.Name == tag {
			return t, true
		}
	}
	return Tag{}, false
}

func getName(path string) string {
	path = strings.TrimPrefix(path, "/v2/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 1 {
		return ""
	}
	return parts[0]
}

func getRef(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func ValidateName(name string) error {
	if strings.Contains(name, "..") {
		return ErrInvalidName
	}
	return nil
}

func ValidateTag(tag string) error {
	if tag == "" || strings.Contains(tag, "..") {
		return ErrInvalidTag
	}
	return nil
}

func init() {}