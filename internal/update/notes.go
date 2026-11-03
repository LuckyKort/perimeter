package update

import (
	"regexp"
	"strings"
)

// Note is one line of "what's new": a short title and the rest of the sentence.
type Note struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// Section returns a version's part of CHANGELOG.md (or of a release body, which
// starts with it): the lines under "## <version>" up to the next heading or "---".
func Section(md, version string) string {
	var out []string
	inside := false
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "## ") {
			if inside {
				break
			}
			f := strings.Fields(line)
			inside = len(f) > 1 && f[1] == version
			continue
		}
		if inside && strings.TrimSpace(line) == "---" {
			break
		}
		if inside {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

var (
	reLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	reBold = regexp.MustCompile(`^\*\*(.+?)\*\*[\s.:-]*`)
)

// Notes turns the bullet list of a changelog section into notes. A bullet that
// starts with **bold** gets that as its title; otherwise its first sentence is.
func Notes(section string) []Note {
	var notes []Note
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		line = reLink.ReplaceAllString(strings.TrimPrefix(line, "- "), "$1")
		line = strings.ReplaceAll(line, "`", "")
		n := Note{Title: line}
		if m := reBold.FindStringSubmatch(line); m != nil {
			n.Title = strings.TrimRight(m[1], ".:")
			n.Text = strings.ReplaceAll(strings.TrimSpace(line[len(m[0]):]), "**", "")
		} else {
			// No bold title: the first sentence is the title, the rest the text.
			n.Title = strings.ReplaceAll(n.Title, "**", "")
			if i := strings.Index(n.Title, ". "); i > 0 {
				n.Title, n.Text = n.Title[:i], strings.TrimSpace(n.Title[i+2:])
			}
		}
		notes = append(notes, n)
	}
	return notes
}

// ReleaseNotes is the changelog part of a GitHub release body: everything before
// the "---" that separates it from the install instructions.
func ReleaseNotes(body string) string {
	if i := strings.Index(body, "\n---"); i >= 0 {
		body = body[:i]
	}
	return body
}
