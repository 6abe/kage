package skill_test

import (
	"strings"
	"testing"

	"github.com/6abe/kage/skill"
)

func TestCanonicalSkill(t *testing.T) {
	s := skill.Markdown
	wantHead := "---\nname: kage\ndescription: See and control the Omarchy/Hyprland desktop."
	if !strings.HasPrefix(s, wantHead) {
		t.Fatalf("frontmatter:\n%s", s[:min(len(s), 200)])
	}
	for _, n := range []string{
		"the agent",
		"Prefer the `kage` CLI via the shell if MCP is missing",
		"kage see --annotate",
		"omarchy screenshot",
		"omarchy capture screenshot region",
		"slurp",
		"see again",
		"Do not assume the click worked",
		"allow_input",
		"--yes",
		"scrot",
		"see only",
		"focused window",
	} {
		if !strings.Contains(s, n) {
			t.Errorf("missing %q", n)
		}
	}
	for _, n := range []string{"Peekaboo", "peekaboo", "Grok"} {
		if strings.Contains(s, n) {
			t.Errorf("provider-neutral skill must not contain %q", n)
		}
	}
}
