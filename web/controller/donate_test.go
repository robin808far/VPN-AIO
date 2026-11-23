package controller

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// donateLine matches one README donate row: 🔹USDT-TRC20: ```TXEh…```
var donateLine = regexp.MustCompile("^\\s*🔹\\s*([^:]+):\\s*`{1,3}([^`]+)`{1,3}\\s*$")

// repoReadmes lists every README in the repository root, relative to this
// package's directory.
func repoReadmes(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob("../../README*.md")
	if err != nil {
		t.Fatalf("glob READMEs: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no README*.md found at repository root")
	}
	return matches
}

// TestNoDonateSectionInReadmes pins the removal of donation information: the
// addresses belong to the upstream project, not this fork, so no README may
// carry a "## Donate"-style section and no donate addresses may appear in any
// of them, in any language.
func TestNoDonateSectionInReadmes(t *testing.T) {
	for _, path := range repoReadmes(t) {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, line := range strings.Split(string(raw), "\n") {
			trimmed := strings.TrimSpace(line)
			if donateLine.MatchString(trimmed) {
				t.Errorf("%s still carries a donate address row: %q", path, trimmed)
			}
		}
		if donateHeading.MatchString(string(raw)) {
			t.Errorf("%s still carries a Donate heading", path)
		}
	}
}

// donateHeading matches a "## Donate"-style heading regardless of language
// (Donaciones, دونیت, Bağış, 捐赠, …).
var donateHeading = regexp.MustCompile(`(?m)^##\s*[^\n]*(?:[Dd]onat|تبرّع|دونیت|حمایت|Пожертвова|Bağış|捐赠)`)

// TestNoDonateDialogInPanel pins the removal on the web side: the index page
// must not render a donate button or modal, and the controller must not pass
// donate data to it.
func TestNoDonateDialogInPanel(t *testing.T) {
	raw, err := os.ReadFile("../html/index.html")
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	html := string(raw)
	for _, marker := range []string{"openDonateModal", "donateModal", "donateList", "donate-modal"} {
		if strings.Contains(html, marker) {
			t.Errorf("index.html still references %q", marker)
		}
	}
}
