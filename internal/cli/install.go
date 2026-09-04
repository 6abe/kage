package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/6abe/kage/internal/host"
	"github.com/6abe/kage/internal/install"
)

func rejectInstall(inv invocation, stderr io.Writer) int {
	if inv.window != "" {
		return writeFail(stderr, "unexpected flag: --window", usage)
	}
	if code := rejectSeeOnly(inv, stderr); code != 0 {
		return code
	}
	if inv.clear {
		return writeFail(stderr, "unexpected flag: --clear", usage)
	}
	if len(inv.rest) > 1 {
		return writeFail(stderr, "unexpected arguments: "+strings.Join(inv.rest, " "), usage)
	}
	return 0
}

func runInstall(h host.Host, inv invocation, stdout, stderr io.Writer) int {
	name := ""
	if len(inv.rest) == 1 {
		name = inv.rest[0]
	}
	r, err := install.Install(h, name)
	return writeInstall(r, err, inv.human, stdout, stderr)
}

func runUninstall(h host.Host, inv invocation, stdout, stderr io.Writer) int {
	name := ""
	if len(inv.rest) == 1 {
		name = inv.rest[0]
	}
	r, err := install.Uninstall(h, name)
	return writeInstall(r, err, inv.human, stdout, stderr)
}

func writeInstall(r install.Result, err error, human bool, stdout, stderr io.Writer) int {
	if err != nil {
		hint := ""
		if errors.Is(err, install.ErrUnknownClient) {
			hint = "grok, claude, cursor, or codex"
		}
		return writeFail(stderr, err.Error(), hint)
	}
	if human {
		fmt.Fprintf(stdout, "%-16s %s\n", "client", r.Client)
		fmt.Fprintf(stdout, "%-16s %s\n", "skill", r.Skill)
		fmt.Fprintf(stdout, "%-16s %s\n", "mcp", r.MCP)
		return 0
	}
	return writeJSON(stdout, r)
}
