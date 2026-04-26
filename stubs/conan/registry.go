package conan

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

var (
	ErrPackageNotFound = fmt.Errorf("package not found")
	ErrVersionNotFound = fmt.Errorf("version not found")
)

type Package struct {
	Name     string    `json:"name"`
	Versions []Version `json:"versions"`
	Recipe   *Recipe  `json:"recipe,omitempty"`
}

type Version struct {
	Version   string   `json:"version"`
	Channel  string   `json:"channel"`
	Revision string   `json:"revision"`
	Time    string   `json:"time"`
}

type Recipe struct {
	ID        string `json:"id"`
	Revision string `json:"revision"`
	Time    string `json:"time"`
}

type ConanFile struct {
	ConanName    string   `json:"name"`
	Version     string   `json:"version"`
	Channel    string   `json:"channel"`
	Recipe      *Recipe  `json:"recipe"`
	License     string   `json:"license"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Homepage   string   `json:"homepage"`
	Issues     string   `json:"issues"`
	Repository *Repository `json:"repository"`
}

type Repository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func (p *Package) String() string {
	return fmt.Sprintf("ConanPackage{name=%s, versions=%d}", p.Name, len(p.Versions))
}

func (p *Package) GetLatest() *Version {
	if len(p.Versions) == 0 {
		return nil
	}
	return &p.Versions[0]
}

type Registry struct {
	packages map[string]*Package
}

func NewRegistry() *Registry {
	return &Registry{
		packages: make(map[string]*Package),
	}
}

func (r *Registry) HandleConan(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path

	switch {
	case strings.HasSuffix(path, "/latest"):
		r.handleGetLatest(w, req)
	case strings.HasSuffix(path, "/files"):
		r.handleGetFiles(w, req)
	case strings.HasSuffix(path, "/download"):
		r.handleDownload(w, req)
	default:
		http.NotFound(w, req)
	}
}

func (r *Registry) handleGetLatest(w http.ResponseWriter, req *http.Request) {
	name := extract ConanName(req.URL.Path)
	pkg, ok := r.packages[name]
	if !ok {
		http.Error(w, "package not found", http.StatusNotFound)
		return
	}

	data, _ := json.Marshal(pkg.GetLatest())
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (r *Registry) handleGetFiles(w http.ResponseWriter, req *http.Request) {
	name := extractConanName(req.URL.Path)
	pkg, ok := r.packages[name]
	if !ok {
		http.Error(w, "package not found", http.StatusNotFound)
		return
	}

	data, _ := json.Marshal(pkg.Versions)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (r *Registry) handleDownload(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "download endpoint", http.StatusNotImplemented)
}

func (r *Registry) RegisterPackage(pkg *Package) {
	r.packages[pkg.Name] = pkg
}

func (r *Registry) GetPackage(name string) (*Package, bool) {
	pkg, ok := r.packages[name]
	return pkg, ok
}

func extractConanName(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if p == "packages" && i+1 < len(parts) {
			name := strings.Split(parts[i+1], "/")
			if len(name) >= 1 {
				return name[0]
			}
		}
	}
	return path
}

func ParseConanReference(ref string) (name, version, channel string) {
	parts := strings.SplitN(ref, "/", 3)
	if len(parts) >= 1 {
		name = parts[0]
	}
	if len(parts) >= 2 {
		version = parts[1]
	}
	if len(parts) >= 3 {
		channel = parts[2]
	}
	return name, version, channel
}

func GenerateConanReference(name, version, channel string) string {
	if channel != "" {
		return fmt.Sprintf("%s/%s@%s", name, version, channel)
	}
	return fmt.Sprintf("%s/%s", name, version)
}

func (r *Registry) HandleAPI(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/v2/conans":
		r.handleListPackages(w, req)
	default:
		http.NotFound(w, req)
	}
}

func (r *Registry) handleListPackages(w http.ResponseWriter, req *http.Request) {
	pkgNames := make([]string, 0)
	for name := range r.packages {
		pkgNames = append(pkgNames, name)
	}
	data, _ := json.Marshal(pkgNames)
	w.Write(data)
}

func init() {}