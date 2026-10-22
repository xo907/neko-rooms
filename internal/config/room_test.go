package config

import (
	"net/url"
	"testing"
)

func TestGetRoomUrl(t *testing.T) {
	// no instance url, no traefik: relative, works on any address
	r := Room{PathPrefix: "/room"}
	if got := r.GetRoomUrl("movies"); got != "/room/movies/" {
		t.Errorf("relative url: got %q", got)
	}

	// explicit instance url
	u, _ := url.Parse("https://rooms.example.com/")
	r.InstanceUrl = u
	if got := r.GetRoomUrl("movies"); got != "https://rooms.example.com/room/movies/" {
		t.Errorf("instance url: got %q", got)
	}

	// wildcard subdomains
	u, _ = url.Parse("https://*.example.com/")
	r.InstanceUrl = u
	if got := r.GetRoomUrl("movies"); got != "https://movies.example.com/" {
		t.Errorf("subdomain url: got %q", got)
	}

	// traefik without instance url keeps absolute links
	r = Room{PathPrefix: "/", Traefik: Traefik{Enabled: true, Domain: "example.com"}}
	if got := r.GetRoomUrl("movies"); got != "http://example.com/movies/" {
		t.Errorf("traefik url: got %q", got)
	}
}
