package youtube

import (
	"context"
	"testing"
)

func TestParseISO8601Duration(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"PT4M13S", 4*60 + 13},
		{"PT1H2M3S", 3600 + 2*60 + 3},
		{"PT45S", 45},
		{"PT10M", 600},
		{"PT2H", 7200},
		{"", 0},
		{"garbage", 0},
	}
	for _, c := range cases {
		if got := parseISO8601Duration(c.in); got != c.want {
			t.Errorf("parseISO8601Duration(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestParseInt64(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"123", 123},
		{"", 0},
		{"not-a-number", 0},
		{"0", 0},
	}
	for _, c := range cases {
		if got := parseInt64(c.in); got != c.want {
			t.Errorf("parseInt64(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestResolveChannelID_BareChannelID(t *testing.T) {
	c := NewClient("unused-in-this-path")
	got, err := c.ResolveChannelID(context.Background(), "UCabcdefghijklmnopqrstuv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "UCabcdefghijklmnopqrstuv" {
		t.Errorf("got %q, want the bare channel ID unchanged", got)
	}
}
