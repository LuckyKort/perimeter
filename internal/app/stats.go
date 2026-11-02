package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/ef1m03/kordon/internal/guard"
)

// Day is one day of statistics.
type Day struct {
	Freezes       int     `json:"freezes"`        // times the apps were frozen (VPN dropped, wrong IP...)
	Saves         int     `json:"saves"`          // ...and the right IP came back in time
	Blocks        int     `json:"blocks"`         // ...and it didn't, so the apps were closed
	Killed        int     `json:"killed"`         // launches stopped while blocked
	FrozenSeconds float64 `json:"frozen_seconds"` // total time apps spent frozen
}

// Stats keeps per-day totals in stats.json, the last 90 days.
type Stats struct {
	mu    sync.Mutex
	Days  map[string]*Day `json:"days"`
	last  guard.Counters
	saved time.Time
}

const keepDays = 90

func statsPath() string { return filepath.Join(Dir(), "stats.json") }

func LoadStats() *Stats {
	s := &Stats{Days: map[string]*Day{}}
	if b, err := os.ReadFile(statsPath()); err == nil {
		_ = json.Unmarshal(b, s)
		if s.Days == nil {
			s.Days = map[string]*Day{}
		}
	}
	return s
}

// Record adds what changed in the guard's counters since the last call to today.
func (s *Stats) Record(c guard.Counters, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := guard.Counters{
		Freezes: c.Freezes - s.last.Freezes, Saves: c.Saves - s.last.Saves,
		Blocks: c.Blocks - s.last.Blocks, Killed: c.Killed - s.last.Killed,
		FrozenSeconds: c.FrozenSeconds - s.last.FrozenSeconds,
	}
	s.last = c
	if d.Freezes == 0 && d.Saves == 0 && d.Blocks == 0 && d.Killed == 0 && d.FrozenSeconds <= 0 {
		return
	}
	key := now.Format("2006-01-02")
	day := s.Days[key]
	if day == nil {
		day = &Day{}
		s.Days[key] = day
	}
	day.Freezes += d.Freezes
	day.Saves += d.Saves
	day.Blocks += d.Blocks
	day.Killed += d.Killed
	day.FrozenSeconds += max(0, d.FrozenSeconds)
	events := d.Freezes+d.Saves+d.Blocks+d.Killed > 0
	if events || now.Sub(s.saved) > 30*time.Second {
		s.save(now)
	}
}

func (s *Stats) save(now time.Time) {
	cutoff := now.AddDate(0, 0, -keepDays).Format("2006-01-02")
	for k := range s.Days {
		if k < cutoff {
			delete(s.Days, k)
		}
	}
	b, _ := json.Marshal(s)
	tmp := statsPath() + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, statsPath())
	}
	s.saved = now
}

// DayStat is one row for the panel's chart.
type DayStat struct {
	Date string `json:"date"`
	Day
}

// Last returns the last n days, oldest first, with empty days filled in.
func (s *Stats) Last(n int, now time.Time) []DayStat {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]DayStat, 0, n)
	for i := n - 1; i >= 0; i-- {
		k := now.AddDate(0, 0, -i).Format("2006-01-02")
		ds := DayStat{Date: k}
		if d := s.Days[k]; d != nil {
			ds.Day = *d
		}
		out = append(out, ds)
	}
	return out
}

// Total sums all kept days.
func (s *Stats) Total() Day {
	s.mu.Lock()
	defer s.mu.Unlock()
	var t Day
	keys := make([]string, 0, len(s.Days))
	for k := range s.Days {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		d := s.Days[k]
		t.Freezes += d.Freezes
		t.Saves += d.Saves
		t.Blocks += d.Blocks
		t.Killed += d.Killed
		t.FrozenSeconds += d.FrozenSeconds
	}
	return t
}
