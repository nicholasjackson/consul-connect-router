package router

import (
	"sort"
	"strings"
)

type ConnectionType string

const HTTP ConnectionType = "http"

const GRPC ConnectionType = "grpc"

// Upstream defines a struct to encapsulate upstream info
type Upstream struct {
	Host        string
	Path        string
	Type        ConnectionType
	StripPrefix string
}

// Upstreams is a collection of Upstream
type Upstreams []Upstream

// FindUpstream finds the correct upstream based on the given path
func (u Upstreams) FindUpstream(path string) *Upstream {
	for _, us := range u {
		if strings.HasPrefix(path, us.Path) {
			return &us
		}
	}

	return nil
}

// Len is part of sort.Interface.
func (u Upstreams) Len() int {
	return len(u)
}

// Swap is part of sort.Interface.
func (u Upstreams) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (u Upstreams) Less(i, j int) bool {
	return len(u[j].Path) < len(u[i].Path)
}

// NewUpstreams parses the command line flags and creates a sorted Upstream slice
func NewUpstreams(u []string) (Upstreams, error) {
	us := Upstreams{}

	for _, v := range u {
		// split into kv pairs
		parts := strings.Split(v, "#")
		u := Upstream{}
		for _, p := range parts {
			kv := strings.Split(p, "=")

			switch kv[0] {
			case "service":
				u.Host = kv[1]
			case "path":
				u.Path = kv[1]
			case "type":
				switch kv[1] {
				case "http":
					u.Type = HTTP
				case "grpc":
					u.Type = GRPC
				}
			case "strip_prefix":
				u.StripPrefix = kv[1]
			}
		}
		us = append(us, u)
	}

	// sort the upstreams to ensure that find always returns the longest path first
	sort.Sort(us)

	return us, nil
}
