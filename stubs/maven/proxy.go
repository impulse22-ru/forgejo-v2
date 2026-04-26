package maven

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/storage"
)

var (
	ErrArtifactNotFound = fmt.Errorf("artifact not found")
	ErrProxyError   = fmt.Errorf("proxy error")
)

type ProxyConfig struct {
	RemoteURL    string
	CacheEnabled bool
	CachePath   string
	DockerHost  string
}

type Artifact struct {
	GroupID    string `json:"group_id"`
	ArtifactID string `json:"artifact_id"`
	Version   string `json:"version"`
	Packaging string `json:"packaging"`
	Classifier string `json:"classifier,omitempty"`
	Extension string `json:"extension"`
}

func (a *Artifact) Filename() string {
	filename := fmt.Sprintf("%s-%s", a.ArtifactID, a.Version)
	if a.Classifier != "" {
		filename += fmt.Sprintf("-%s", a.Classifier)
	}
	filename += fmt.Sprintf(".%s", a.Extension)
	return filename
}

func (a *Artifact) Path() string {
	groupPath := strings.ReplaceAll(a.GroupID, ".", "/")
	return fmt.Sprintf("%s/%s/%s/%s", groupPath, a.ArtifactID, a.Version, a.Filename())
}

func (a *Artifact) POMPath() string {
	groupPath := strings.ReplaceAll(a.GroupID, ".", "/")
	return fmt.Sprintf("%s/%s/%s/%s.pom", groupPath, a.ArtifactID, a.Version, a.ArtifactID)
}

type MavenProxy struct {
	config  *ProxyConfig
	local   storage.Storage
	remote  *http.Client
	cache   *Cache
}

func NewMavenProxy(config *ProxyConfig) *MavenProxy {
	return &MavenProxy{
		config: config,
		remote: &http.Client{Timeout: 30 * time.Second},
		cache:  NewCache(),
	}
}

func (m *MavenProxy) Handle(w http.ResponseWriter, req *http.Request) {
	artifact, err := m.parsePath(req.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch req.Method {
	case "GET":
		m.handleGet(w, artifact)
	case "HEAD":
		m.handleHead(w, artifact)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *MavenProxy) handleGet(w http.ResponseWriter, artifact *Artifact) {
	content, err := m.fetchArtifact(artifact)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer content.Close()

	w.Header().Set("Content-Type", getContentType(artifact.Extension))
	io.Copy(w, content)
}

func (m *MavenProxy) handleHead(w http.ResponseWriter, artifact *Artifact) {
	info, err := m.getArtifactInfo(artifact)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))
	w.Header().Set("ETag", fmt.Sprintf(`"%s"`, info.ETag))
}

func (m *MavenProxy) parsePath(urlPath string) (*Artifact, error) {
	cleanPath := strings.TrimPrefix(urlPath, "/maven2/")
	parts := strings.Split(cleanPath, "/")

	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid path: %s", urlPath)
	}

	groupID := strings.Join(parts[:len(parts)-3], ".")
	artifactID := parts[len(parts)-3]
	version := parts[len(parts)-2]
	filename := parts[len(parts)-1]

	ext := strings.TrimPrefix(path.Ext(filename), "")
	classifier := ""

	if strings.Contains(filename, "-") {
		nameParts := strings.SplitN(strings.TrimSuffix(filename, path.Ext(filename)), "-", 2)
		if len(nameParts) > 1 {
			classifier = nameParts[1]
		}
	}

	return &Artifact{
		GroupID:    groupID,
		ArtifactID: artifactID,
		Version:   version,
		Extension: ext,
		Packaging: ext,
		Classifier: classifier,
	}, nil
}

func (m *MavenProxy) fetchArtifact(artifact *Artifact) (io.ReadCloser, error) {
	if m.config.CacheEnabled {
		localPath := artifact.Path()
		if f, err := m.local.Open(localPath); err == nil {
			return f, nil
		}
	}

	return m.fetchRemote(artifact)
}

func (m *MavenProxy) fetchRemote(artifact *Artifact) (io.ReadCloser, error) {
	url := m.config.RemoteURL + "/" + artifact.Path()
	resp, err := m.remote.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProxyError, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrArtifactNotFound
	}

	if m.config.CacheEnabled {
		m.cacheArtifact(artifact, resp.Body)
	}

	return resp.Body, nil
}

func (m *MavenProxy) cacheArtifact(artifact *Artifact, content io.Reader) error {
	if m.local == nil {
		return nil
	}

	reader := io.TeeReader(content, &strings.Builder{})
	localPath := artifact.Path()

	return m.local.Save(localPath, reader)
}

func (m *MavenProxy) getArtifactInfo(artifact *Artifact) (*ArtifactInfo, error) {
	localPath := artifact.Path()
	if f, err := m.local.Open(localPath); err == nil {
		defer f.Close()
		info, _ := f.Stat()
		return &ArtifactInfo{
			Size: info.Size(),
			ETag: generateETag(localPath),
		}, nil
	}

	return nil, ErrArtifactNotFound
}

type ArtifactInfo struct {
	Size int64
	ETag string
}

func generateETag(identifier string) string {
	hash := md5.Sum([]byte(identifier))
	return hex.EncodeToString(hash[:])
}

func getContentType(ext string) string {
	types := map[string]string{
		"jar":  "application/java-archive",
		"pom":  "application/xml",
		"xml":  "application/xml",
		"md":   "text/markdown",
		"txt":  "text/plain",
	}
	if t, ok := types[ext]; ok {
		return t
	}
	return "application/octet-stream"
}

type Cache struct {
	store map[string]*CachedArtifact
}

func NewCache() *Cache {
	return &Cache{
		store: make(map[string]*CachedArtifact),
	}
}

func (c *Cache) Get(key string) (*CachedArtifact, bool) {
	art, ok := c.store[key]
	if !ok {
		return nil, false
	}
	if art.Expired() {
		delete(c.store, key)
		return nil, false
	}
	return art, true
}

func (c *Cache) Put(key string, art *CachedArtifact) {
	c.store[key] = art
}

type CachedArtifact struct {
	Data   []byte
	ETag   string
	Time   time.Time
	Expiry time.Duration
}

func (c *CachedArtifact) Expired() bool {
	if c.Expiry == 0 {
		return false
	}
	return time.Since(c.Time) > c.Expiry
}

type Checksums struct {
	SHA1 string
	MD5 string
}

func CalculateChecksums(content io.Reader) (*Checksums, error) {
	h1 := sha1.New()
	m5 := md5.New()

	multiWriter := io.MultiWriter(h1, m5)

	_, err := io.Copy(multiWriter, content)
	if err != nil {
		return nil, err
	}

	return &Checksums{
		SHA1: hex.EncodeToString(h1.Sum(nil)),
		MD5:  hex.EncodeToString(m5.Sum(nil)),
	}, nil
}

func init() {}