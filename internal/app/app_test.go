package app

import (
	"strings"
	"testing"
	"time"

	"github.com/ef1m03/kordon/internal/guard"
)

func TestMask(t *testing.T) {
	in := "2026-11-03 14:05:33 IP 64.137.54.87 v6 2a02:2698:1:2::5 net 5c:62:8b:c0:3e:c6 allow 10.0.0.0/8"
	got := Mask(in)
	for _, leak := range []string{"64.137", "2a02", "5c:62", "10.0.0.0"} {
		if strings.Contains(got, leak) {
			t.Fatalf("%q still in %q", leak, got)
		}
	}
	if !strings.Contains(got, "14:05:33") {
		t.Fatalf("time got masked: %q", got)
	}
}

func TestStats(t *testing.T) {
	t.Setenv("KORDON_DIR", t.TempDir())
	s := LoadStats()
	day1 := time.Date(2026, 11, 3, 12, 0, 0, 0, time.Local)
	s.Record(guard.Counters{Freezes: 2, Saves: 1, FrozenSeconds: 10}, day1)
	s.Record(guard.Counters{Freezes: 3, Saves: 2, FrozenSeconds: 15}, day1.Add(24*time.Hour))
	last := s.Last(3, day1.Add(24*time.Hour))
	if len(last) != 3 || last[1].Freezes != 2 || last[2].Freezes != 1 || last[2].FrozenSeconds != 5 {
		t.Fatalf("%+v", last)
	}
	// Survives a restart (the guard's counters start from zero again).
	s2 := LoadStats()
	s2.Record(guard.Counters{Freezes: 1}, day1.Add(24*time.Hour))
	if s2.Total().Freezes != 4 {
		t.Fatalf("total %+v", s2.Total())
	}
}
