package markdown

import (
	"strings"
	"testing"
)

func TestToHTML(t *testing.T) {
	input := []byte(
		`## some headline
followed by some paragraph with [a link](https://example.local)
and some list:
- first
- second
  - nested on second level
	- nested on third level
	- ~~strikethrough~~
  - yet another second level item, **but** with a [a link](https://example.local)
- end

### h3 headline/header

<a href="https://example.local">embedded HTML is allowed</a>
  `,
	)
	expected := []byte(
		`<h2 id="some-headline">some headline</h2>
<p>followed by some paragraph with <a href="https://example.local">a link</a>
and some list:</p>
<ul>
<li>first</li>
<li>second
<ul>
<li>nested on second level
<ul>
<li>nested on third level</li>
<li><del>strikethrough</del></li>
</ul>
</li>
<li>yet another second level item, <strong>but</strong> with a <a href="https://example.local">a link</a></li>
</ul>
</li>
<li>end</li>
</ul>
<h3 id="h3-headlineheader">h3 headline/header</h3>
<p><a href="https://example.local">embedded HTML is allowed</a></p>`,
	)

	got, err := ToHTML(input)
	if err != nil {
		t.Errorf("ToHTML() error = %v", err)
		return
	}
	if strings.TrimSpace(string(got)) != strings.TrimSpace(string(expected)) {
		t.Errorf("ToHTML() got = %v, want %v", string(got), string(expected))
	}
}

func TestToHTMLSanitizesUnsafeHTML(t *testing.T) {
	input := []byte(
		`## unsafe
<script>alert("xss")</script>
<iframe src="https://example.local/embed"></iframe>
<a href="javascript:alert(1)" onclick="alert(1)" target="_blank">bad link</a>
<img src="https://example.local/logo.png" alt="logo" onerror="alert(1)">
<table cellpadding="5"><tbody><tr><td colspan="2">allowed table</td></tr></tbody></table>
`,
	)

	got, err := ToHTML(input)
	if err != nil {
		t.Fatalf("ToHTML() error = %v", err)
	}

	output := string(got)
	for _, unsafe := range []string{
		"<script",
		"alert(\"xss\")",
		"<iframe",
		"javascript:",
		"onclick",
		"onerror",
	} {
		if strings.Contains(output, unsafe) {
			t.Fatalf("ToHTML() output contains unsafe content %q: %s", unsafe, output)
		}
	}

	for _, allowed := range []string{
		`target="_blank"`,
		`rel="noopener noreferrer"`,
		`>bad link</a>`,
		`<img src="https://example.local/logo.png" alt="logo"`,
		`<table cellpadding="5"><tbody><tr><td colspan="2">allowed table</td></tr></tbody></table>`,
	} {
		if !strings.Contains(output, allowed) {
			t.Fatalf("ToHTML() output missing allowed content %q: %s", allowed, output)
		}
	}
}
