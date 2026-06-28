package main

import (
	"os"
	"strings"
	"testing"
)

func TestRenderHTML(t *testing.T) {
	tables := []Table{{
		Schema: "public", Name: "orders",
		Columns: []Column{
			{Name: "id", Type: "integer", PK: true},
			{Name: "user_id", Type: "integer", FK: &FKRef{Schema: "public", Table: "users", Column: "id"}},
			{Name: "note", Type: "text", Nullable: true},
		},
		Uniques: [][]string{{"site_id", "name"}},
		Checks:  []string{"qty > 0"},
	}}

	f, err := os.CreateTemp(t.TempDir(), "*.html")
	if err != nil {
		t.Fatal(err)
	}
	if err := renderHTML(f, tables); err != nil {
		t.Fatal(err)
	}
	f.Close()
	out, _ := os.ReadFile(f.Name())
	s := string(out)

	for _, want := range []string{
		`id="public.orders"`,            // anchor target
		`href="#public.orders"`,         // sidebar link
		`href="#public.users"`,          // FK link points at referenced table
		`<span class="badge pk">PK</span>`, // pk badge
		`UNIQUE (site_id, name)`,           // composite unique stays composite
		`qty &gt; 0`,                       // check escaped
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in output", want)
		}
	}
	// composite unique must NOT leak into a per-column badge
	if strings.Contains(s, `<span class="badge">UNIQUE</span>`) {
		t.Error("composite unique was rendered as a per-column UNIQUE badge")
	}
}

// identifiers with HTML metacharacters must be escaped, not injected verbatim.
func TestRenderHTMLEscapesIdentifiers(t *testing.T) {
	tables := []Table{{
		Schema: "public", Name: "ev<il>",
		Columns: []Column{{Name: "c<ol>", Type: "te<xt>"}},
	}}
	var b strings.Builder
	if err := renderHTML(&b, tables); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	if strings.Contains(s, "<il>") || strings.Contains(s, "<ol>") || strings.Contains(s, "<xt>") {
		t.Error("identifier rendered unescaped")
	}
	if !strings.Contains(s, "ev&lt;il&gt;") || !strings.Contains(s, "c&lt;ol&gt;") {
		t.Error("expected identifiers escaped to &lt;..&gt;")
	}
}
