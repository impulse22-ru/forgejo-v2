package conan

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

var (
	ErrPackageNotFound = fmt.Errorf("package not found")
)

type Package struct {
	Name      string    `json:"name"`
	Versions  []Version `json:"versions"`
	Recipe   *Recipe  `json:"recipe,omitempty"`
}

type Version struct {
	Version   string `json:"version"`
	Channel  string `json:"channel"`
	Revision string `json:"revision"`
	Time    string `json:"time"`
}

type Recipe struct {
	ID        string `json:"id"`
	Revision string `json:"revision"`
	Time    string `json:"time"`
}

type Registry struct {
	packages map[string]*Package
}

func NewRegistry() *Registry {
	return &Registry{
		packages: make(map[string]*Package),
	}
}

func (r *Registry) HandleAPI(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/v2/conans" {
		list := make([]string, 0, len(r.packages))
		for name := range r.packages {
			list = append(list, name)
		}
		data, _ := json.Marshal(list)
		w.Write(data)
	}
}

func init() {}