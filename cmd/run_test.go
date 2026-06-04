package cmd

import "testing"

func TestParseRunArgsTypeForms(t *testing.T) {
	forms := [][]string{
		{"bot", "-t", "agent"},
		{"bot", "-t=agent"},
		{"bot", "-tagent"},
		{"bot", "--type", "agent"},
		{"bot", "--type=agent"},
	}
	for _, args := range forms {
		p, err := parseRunArgs(args)
		if err != nil {
			t.Errorf("%v: %v", args, err)
			continue
		}
		if p.name != "bot" || p.typ != "agent" {
			t.Errorf("%v: name=%q typ=%q", args, p.name, p.typ)
		}
	}
}

func TestParseRunArgsVarsAndFile(t *testing.T) {
	p, err := parseRunArgs([]string{"triage", "--tone=blunt", "--set", "file=README", "--file", "log.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if p.name != "triage" {
		t.Errorf("name = %q", p.name)
	}
	if p.vars["tone"] != "blunt" {
		t.Errorf("tone = %q", p.vars["tone"])
	}
	if p.vars["file"] != "README" {
		t.Errorf("set file = %q", p.vars["file"])
	}
	if p.file != "log.txt" {
		t.Errorf("file = %q", p.file)
	}
}

func TestParseRunArgsJSON(t *testing.T) {
	p, err := parseRunArgs([]string{"triage", "--json", "--tone=blunt"})
	if err != nil {
		t.Fatal(err)
	}
	if !p.json {
		t.Error("--json not parsed")
	}
	if p.vars["tone"] != "blunt" {
		t.Errorf("tone = %q", p.vars["tone"])
	}
	// Absent by default.
	p2, _ := parseRunArgs([]string{"triage"})
	if p2.json {
		t.Error("json should default to false")
	}
}
