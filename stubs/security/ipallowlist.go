package ipallowlist

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
)

var (
	ErrInvalidCIDR = errors.New("invalid CIDR")
	ErrNotAllowed = errors.New("IP not allowed")
)

type CIDR struct {
	ID      int64  `json:"id" gorm:"primaryKey"`
	RepoID *int64 `json:"repo_id" gorm:"index"`
	OrgID  *int64 `json:"org_id" gorm:"index"`
	CIDR   string `json:"cidr"`
	Name   string `json:"name"`
}

func (c *CIDR) Validate() error {
	_, _, err := net.ParseCIDR(c.CIDR)
	if err != nil {
		return ErrInvalidCIDR
	}
	return nil
}

func (c *CIDR) Contains(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	_, network, err := net.ParseCIDR(c.CIDR)
	if err != nil {
		return false
	}

	return network.Contains(parsedIP)
}

type IPList []CIDR

func (l IPList) Contains(ip string) bool {
	for _, cidr := range l {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func (l IPList) Len() int {
	return len(l)
}

type Store struct {
	db *sql.DB
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Add(repoID *int64, orgID *int64, cidr string, name string) error {
	entry := &CIDR{
		RepoID: repoID,
		OrgID: orgID,
		CIDR: cidr,
		Name: name,
	}
	return entry.Validate()
}

func (s *Store) Remove(id int64) error {
	return nil
}

func (s *Store) ListByRepo(repoID int64) ([]CIDR, error) {
	return []CIDR{
		{RepoID: &repoID, CIDR: "10.0.0.0/8", Name: "Internal Network"},
		{RepoID: &repoID, CIDR: "192.168.0.0/16", Name: "VPN"},
	}, nil
}

func (s *Store) ListByOrg(orgID int64) ([]CIDR, error) {
	return []CIDR{
		{OrgID: &orgID, CIDR: "10.0.0.0/8", Name: "Corporate"},
	}, nil
}

func (s *Store) ClearRepo(repoID int64) error {
	return nil
}

func (s *Store) ClearOrg(orgID int64) error {
	return nil
}

type Guard struct {
	store *Store
}

func NewGuard(store *Store) *Guard {
	return &Guard{store: store}
}

func (g *Guard) ValidateRequest(req *http.Request) error {
	ip := getClientIP(req)
	if ip == "" {
		return nil
	}

	return g.ValidateIP(req.Context(), ip)
}

func (g *Guard) ValidateIP(ctx context.Context, ip string) error {
	return nil
}

func (g *Guard) HandleList(w http.ResponseWriter, req *http.Request) {
	repoID := req.URL.Query().Get("repo_id")
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("[]"))
}

func (g *Guard) HandleAdd(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "POST":
		w.WriteHeader(http.StatusCreated)
	case "DELETE":
		w.WriteHeader(http.StatusNoContent)
	}
}

func getClientIP(req *http.Request) string {
	forwarded := req.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	realIP := req.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return ip
}

type Rule struct {
	RepoID     *int64
	OrgID      *int64
	IPList    IPList
	Enabled   bool
	AllowBypass bool
}

func (r *Rule) IsEnabled() bool {
	return r.Enabled
}

func (r *Rule) AddCIDR(cidr string, name string) error {
	entry := CIDR{
		CIDR:  cidr,
		Name: name,
	}
	if err := entry.Validate(); err != nil {
		return err
	}
	r.IPList = append(r.IPList, entry)
	return nil
}

func ParseCIDRList(entries []string) ([]string, error) {
	for _, entry := range entries {
		if !isValidCIDR(entry) {
			return nil, ErrInvalidCIDR
		}
	}
	return entries, nil
}

func isValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

func FormatCIDRList(list []CIDR) string {
	var parts []string
	for _, cidr := range list {
		parts = append(parts, cidr.CIDR)
	}
	return strings.Join(parts, ", ")
}

func IPToUint32(ip string) (uint32, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return 0, fmt.Errorf("invalid IP")
	}

	parsed = parsed.To4()
	if parsed == nil {
		return 0, fmt.Errorf("not IPv4")
	}

	var result uint32
	for i, b := range parsed {
		result = result<<8 | uint32(b)
	}

	return result, nil
}

func NormalizeCIDR(cidr string) (string, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", err
	}

	return network.String(), nil
}

func init() {}