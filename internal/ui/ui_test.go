package ui

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"testing"
	"time"

	"perimeter/internal/config"
	"perimeter/internal/guard"
)

func TestMessagesComplete(t *testing.T) {
	for k := range msgs["ru"] {
		if msgs["en"][k] == "" {
			t.Errorf("en is missing %q", k)
		}
	}
	for k := range msgs["en"] {
		if msgs["ru"][k] == "" {
			t.Errorf("ru is missing %q", k)
		}
	}
}

func TestEveryEventHasText(t *testing.T) {
	for _, lang := range []string{"ru", "en"} {
		Lang = lang
		for k := guard.EvAutoAllowed; k <= guard.EvAllowedCountry; k++ {
			e := guard.Event{At: time.Now(), Kind: k, IP: "1.2.3.4", Country: "NL", Apps: []string{"Claude"}, Dur: 20 * time.Second, Notify: true}
			line := EventLine(e)
			if len(line) < 12 {
				t.Errorf("%s: empty history line for kind %d", lang, k)
			}
			if title, body, ok := Notification(e); ok && (title == "" || body == "") {
				t.Errorf("%s: empty notification for kind %d", lang, k)
			}
		}
	}
	Lang = "ru"
}

func TestCountry(t *testing.T) {
	Lang = "ru"
	useFlags = true
	if got := Country("NL"); got != "🇳🇱 Нидерланды" {
		t.Errorf("ru: %q", got)
	}
	Lang = "en"
	useFlags = false
	if got := Country("DE"); got != "Germany (DE)" {
		t.Errorf("en/windows: %q", got)
	}
	Lang, useFlags = "ru", true
}

func TestStatusLine(t *testing.T) {
	Lang = "ru"
	s := guard.Snapshot{Status: guard.Suspect, Reason: guard.IPNotAllowed, GraceLeft: 17 * time.Second, Config: config.Default()}
	if got := StatusLine(s); got != "IP не разрешён: заморожено, закрою через 17 с" {
		t.Errorf("%q", got)
	}
}

func TestICO(t *testing.T) {
	ico := ICO(Badge(Green, "NL", 16), Badge(Red, "", 32))
	var hdr [3]uint16
	binary.Read(bytes.NewReader(ico), binary.LittleEndian, &hdr)
	if hdr[1] != 1 || hdr[2] != 2 {
		t.Fatalf("bad ico header %v", hdr)
	}
	off := binary.LittleEndian.Uint32(ico[6+12 : 6+16])
	if _, err := png.Decode(bytes.NewReader(ico[off:])); err != nil {
		t.Fatalf("first image is not a PNG: %v", err)
	}
	for _, tone := range []Tone{Green, Gray, Amber, Red} {
		b, template := BarIcon(tone, 3)
		if template != (tone == Green || tone == Gray) {
			t.Fatalf("tone %d: template %v", tone, template)
		}
		if _, err := png.Decode(bytes.NewReader(b)); err != nil {
			t.Fatal(err)
		}
	}
	if BarLit(10*time.Second, 20*time.Second) != 4 || BarLit(0, 20*time.Second) != 0 || BarLit(20*time.Second, 20*time.Second) != 8 {
		t.Fatal("BarLit")
	}
}
