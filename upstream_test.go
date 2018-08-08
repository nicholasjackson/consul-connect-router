package router

import (
	"testing"
)

func createUpstreams() (Upstreams, error) {
	flags := []string{
		"service=blah#path=/blah#strip_prefix=/blah#port=9080#type=grpc",
		"service=something#path=/api/something",
		"service=frontend#path=/",
		"service=api#path=/api",
	}

	us, err := NewUpstreams(flags)
	if err != nil {
		return nil, err
	}

	return us, nil
}

func TestNewCreatesUpstreamCollection(t *testing.T) {
	us, err := createUpstreams()

	if err != nil {
		t.Fatalf("Should not have returned error %v", err)
	}

	if len(us) != 4 {
		t.Fatal("Should have created 4 upstreams")
	}
}

func TestSetsService(t *testing.T) {
	us, _ := createUpstreams()
	u := us.FindUpstream("/blah")

	if u.Service != "blah" {
		t.Fatalf("Expected: service, got: %v", u.Service)
	}
}

func TestSetsPort(t *testing.T) {
	us, _ := createUpstreams()
	u := us.FindUpstream("/blah")

	if u.Port != 9080 {
		t.Fatalf("Expected: port 8080, got: %v", u.Port)
	}
}

func TestSetsStripPrefix(t *testing.T) {
	us, _ := createUpstreams()
	u := us.FindUpstream("/blah")

	if u.StripPrefix != "/blah" {
		t.Fatalf("Expected: prefix /blah, got: %v", u.StripPrefix)
	}
}

func TestSetsType(t *testing.T) {
	us, _ := createUpstreams()
	u := us.FindUpstream("/blah")

	if u.Type != GRPC {
		t.Fatalf("Expected: type grpc, got: %v", u.Type)
	}
}

func TestSetsDefaultPort(t *testing.T) {
	us, _ := createUpstreams()
	u := us.FindUpstream("/something")

	if u.Port != 8080 {
		t.Fatalf("Expected: port 9080, got: %v", u.Port)
	}
}

func TestSetsDefaultStripPrefix(t *testing.T) {
	us, _ := createUpstreams()
	u := us.FindUpstream("/something")

	if u.StripPrefix != "" {
		t.Fatalf("Expected: prefix , got: %v", u.StripPrefix)
	}
}

func TestSetsDefaultType(t *testing.T) {
	us, _ := createUpstreams()
	u := us.FindUpstream("/something")

	if u.Type != HTTP {
		t.Fatalf("Expected: type http, got: %v", u.Type)
	}
}

func TestNewSortsCorrectly(t *testing.T) {
	us, _ := createUpstreams()

	if us[0].Path != "/api/something" {
		t.Fatal("Paths should have been sorted", us[0])
	}
}

func TestFindReturnsPath(t *testing.T) {
	flags := []string{
		"service=blah#path=/blah",
		"service=something#path=/api/something",
		"service=frontend#path=/",
		"service=api#path=/api",
	}

	us, err := NewUpstreams(flags)
	if err != nil {
		t.Fatal(err)
	}

	u := us.FindUpstream("/blah")
	if u.Path != "/blah" {
		t.Fatal("Should have returned correct upstream")
	}
}
