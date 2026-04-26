package artifacts

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/storage"
)

var (
	ErrArtifactNotFound = fmt.Errorf("artifact not found")
	ErrInvalidArtifact = fmt.Errorf("invalid artifact")
	ErrStorageError     = fmt.Errorf("storage error")
)

type Artifact struct {
	ID          int64     `json:"id"`
	RunID      int64     `json:"run_id"`
	JobID      int64     `json:"job_id"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	DownloadCount int    `json:"download_count"`
	ContentType string   `json:"content_type"`
	Filename   string    `json:"filename"`
	ArchiveType string   `json:"archive_type"`
}

func (a *Artifact) String() string {
	return fmt.Sprintf("Artifact{id=%d, name=%s, size=%d}", a.ID, a.Name, a.Size)
}

func (a *Artifact) Path() string {
	return fmt.Sprintf("runs/%d/artifacts/%d", a.RunID, a.ID)
}

func (a *Artifact) IsExpired() bool {
	if a.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*a.ExpiresAt)
}

type ArtifactFile struct {
	ID         int64  `json:"id"`
	ArtifactID int64  `json:"artifact_id"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Checksum   string `json:"checksum"`
}

type ArtifactList []Artifact

func (l ArtifactList) Len() int           { return len(l) }
func (l ArtifactList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l ArtifactList) Less(i, j int) bool   { return l[i].CreatedAt.After(l[j].CreatedAt) }

type RetentionPolicy struct {
	RepoID       int64  `json:"repo_id"`
	RetentionDays int   `json:"retention_days"`
	MinRetention int   `json:"min_retention_days"`
	MaxRetention int   `json:"max_retention_days"`
	Enabled     bool   `json:"enabled"`
}

func (r *RetentionPolicy) Validate() error {
	if r.RetentionDays < 0 {
		return fmt.Errorf("retention days must be positive")
	}
	if r.MinRetention < 0 {
		return fmt.Errorf("min retention days must be positive")
	}
	if r.MaxRetention < 0 {
		return fmt.Errorf("max retention days must be positive")
	}
	if r.MinRetention > r.MaxRetention {
		return fmt.Errorf("min retention cannot exceed max retention")
	}
	return nil
}

func (r *RetentionPolicy) ShouldRetain(createdAt time.Time) bool {
	if !r.Enabled {
		return false
	}

	age := time.Since(createdAt)
	ageDays := int(age.Hours() / 24)

	if ageDays < r.MinRetention {
		return true
	}

	if ageDays > r.MaxRetention {
		return false
	}

	return true
}

func (r *RetentionPolicy) ExpiresAt(created time.Time) *time.Time {
	if !r.Enabled || r.RetentionDays == 0 {
		return nil
	}

	expires := created.AddDate(0, 0, r.RetentionDays)
	return &expires
}

type ArtifactStore struct {
	storage   storage.Storage
	policies map[int64]*RetentionPolicy
}

func NewArtifactStore() *ArtifactStore {
	return &ArtifactStore{
		policies: make(map[int64]*RetentionPolicy),
	}
}

func (s *ArtifactStore) SetStorage(storage storage.Storage) {
	s.storage = storage
}

func (s *ArtifactStore) Save(artifact *Artifact, data io.Reader) error {
	if s.storage == nil {
		return ErrStorageError
	}

	artifactPath := artifact.Path()
	if err := s.storage.Save(artifactPath, data); err != nil {
		return fmt.Errorf("failed to save artifact: %w", err)
	}

	return nil
}

func (s *ArtifactStore) Load(artifact *Artifact) (io.ReadCloser, error) {
	if s.storage == nil {
		return nil, ErrStorageError
	}

	artifactPath := artifact.Path()
	return s.storage.Open(artifactPath)
}

func (s *ArtifactStore) Delete(artifact *Artifact) error {
	if s.storage == nil {
		return ErrStorageError
	}

	artifactPath := artifact.Path()
	return s.storage.Delete(artifactPath)
}

func (s *ArtifactStore) List(repoID int64, runID int64) ([]Artifact, error) {
	var artifacts []Artifact
	// This would query from database in real implementation
	_ = repoID
	_ = runID
	return artifacts, nil
}

func (s *ArtifactStore) SetPolicy(repoID int64, policy *RetentionPolicy) {
	policy.RepoID = repoID
	s.policies[repoID] = policy
}

func (s *ArtifactStore) GetPolicy(repoID int64) *RetentionPolicy {
	return s.policies[repoID]
}

func (s *ArtifactStore) ApplyRetention(ctx context.Context, repoID int64) (int, error) {
	policy, ok := s.policies[repoID]
	if !ok || !policy.Enabled {
		return 0, nil
	}

	artifacts, err := s.List(repoID, 0)
	if err != nil {
		return 0, err
	}

	var deleted int
	for _, artifact := range artifacts {
		if artifact.IsExpired() || !policy.ShouldRetain(artifact.CreatedAt) {
			if err := s.Delete(&artifact); err != nil {
				log.Error("Failed to delete artifact %d: %v", artifact.ID, err)
				continue
			}
			deleted++
		}
	}

	return deleted, nil
}

type Uploader struct {
	artifact *Artifact
	files    []*ArtifactFile
	writer  *zip.Writer
}

func NewUploader(name string) *Uploader {
	return &Uploader{
		artifact: &Artifact{
			Name:      name,
			CreatedAt: time.Now(),
		},
		files: make([]*ArtifactFile, 0),
	}
}

func (u *Uploader) AddFile(name string, size int64, checksum string) {
	u.files = append(u.files, &ArtifactFile{
		Name:       name,
		Size:      size,
		Checksum:  checksum,
	})
}

func (u *Uploader) Done() error {
	u.artifact.Size = 0
	for _, file := range u.files {
		u.artifact.Size += file.Size
	}
	return nil
}

func init() {}

type Downloader struct {
	artifact *Artifact
}

func NewDownloader(artifact *Artifact) *Downloader {
	return &Downloader{artifact: artifact}
}

func (d *Downloader) Download() (io.ReadCloser, error) {
	return nil, nil
}

type ArtifactRetention struct {
	store *ArtifactStore
}

func NewArtifactRetention(store *ArtifactStore) *ArtifactRetention {
	return &ArtifactRetention{store: store}
}

func (r *ArtifactRetention) Run(ctx context.Context) (int, error) {
	var totalDeleted int

	for repoID := range r.store.policies {
		deleted, err := r.store.ApplyRetention(ctx, repoID)
		if err != nil {
			log.Error("Failed to apply retention for repo %d: %v", repoID, err)
			continue
		}
		totalDeleted += deleted
	}

	return totalDeleted, nil
}

func CreateRetentionPolicyDefault() *RetentionPolicy {
	return &RetentionPolicy{
		RetentionDays: 90,
		MinRetention: 7,
		MaxRetention: 180,
		Enabled:     true,
	}
}

func (p *RetentionPolicy) ToSettings() string {
	if !p.Enabled {
		return "disabled"
	}
	return fmt.Sprintf("%d days (min: %d, max: %d)", p.RetentionDays, p.MinRetention, p.MaxRetention)
}

func ValidateRetentionDays(days int) error {
	if days < 1 {
		return fmt.Errorf("retention days must be at least 1")
	}
	if days > 365 {
		return fmt.Errorf("retention days cannot exceed 365")
	}
	return nil
}

func GetArtifactStats(artifacts []Artifact) map[string]any {
	stats := map[string]any{
		"total":         len(artifacts),
		"total_size":    int64(0),
		"total_expired": int(0),
	}

	for _, a := range artifacts {
		stats["total_size"].(int64) += a.Size
		if a.IsExpired() {
			stats["total_expired"].(int)+++
		}
	}

	return stats
}

func SortArtifacts(artifacts []Artifact, sortBy string, ascending bool) {
	sort.Slice(artifacts, func(i, j int) bool {
		var cmp bool
		switch sortBy {
		case "name":
			cmp = artifacts[i].Name < artifacts[j].Name
		case "size":
			cmp = artifacts[i].Size < artifacts[j].Size
		case "created":
			cmp = artifacts[i].CreatedAt.Before(artifacts[j].CreatedAt)
		default:
			cmp = artifacts[i].CreatedAt.After(artifacts[j].CreatedAt)
		}
		return cmp != ascending
	})
}

func UploadArtifactMultipart(w *multipart.Writer, name string, data []byte, contentType string) error {
	part, err := w.CreateFormFile(name, contentType)
	if err != nil {
		return err
	}

	_, err = part.Write(data)
	return err
}

func GetArtifactPath(repoID, runID, artifactID int64) string {
	return filepath.Join(strconv.FormatInt(repoID, 10), "runs", strconv.FormatInt(runID, 10), "artifacts", strconv.FormatInt(artifactID, 10))
}

func ParseArtifactFilename(filename string) (string, string) {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	return name, ext
}

func init() {}