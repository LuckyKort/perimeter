package update

import "testing"

func TestNotes(t *testing.T) {
	md := "# Что нового\n\n## 1.3.0 - 04.11.2026\n\n- **Новое окно в трее.** Клик по значку.\n- Ссылка [на GitHub](https://x) и `код`.\n\n## 1.2.2 - 03.11.2026\n\n- старое\n"
	n := Notes(Section(md, "1.3.0"))
	if len(n) != 2 || n[0].Title != "Новое окно в трее" || n[0].Text != "Клик по значку." || n[1].Title != "Ссылка на GitHub и код." {
		t.Fatalf("%+v", n)
	}
	if n := Notes("- Первое предложение. Второе."); n[0].Title != "Первое предложение" || n[0].Text != "Второе." {
		t.Fatalf("split: %+v", n)
	}
	body := "- **Браузеры.** Chrome.\n\n---\n\n## Установка\n- не заметка\n"
	if got := Notes(ReleaseNotes(body)); len(got) != 1 || got[0].Title != "Браузеры" {
		t.Fatalf("release notes: %+v", got)
	}
}
