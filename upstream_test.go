package router

import "testing"

func TestNewCreatesUpstreamCollection(t *testing.T) {
	flags := []string{
		"service=db#path=/#type=http#strip_prefix=/",
	}

	us, err := NewUpstreams(flags)
	if err != nil {
		t.Fatal(err)
	}

	if us[0].Path != "/" {
		t.Fatal("Path should be equal to /")
	}
}

func TestNewSortsCorrectly(t *testing.T) {
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
