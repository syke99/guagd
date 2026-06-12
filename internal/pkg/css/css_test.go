package css

import (
	"strings"
	"testing"
)

func TestSanitizeProp(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"color", "color"},
		{"background-color", "background-color"},
		{"--accent", "--accent"},
		{"font-size", "font-size"},
		{"color: red", ""},    // colon not allowed
		{"color;drop", ""},    // semicolon not allowed
		{"color{}", ""},       // braces not allowed
		{"color()", ""},       // parens not allowed
		{"", ""},
	}
	for _, c := range cases {
		got := sanitizeProp(c.in)
		if got != c.want {
			t.Errorf("sanitizeProp(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSanitizeValue(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"#e85d04", "#e85d04"},
		{"  #e85d04  ", "#e85d04"},  // trimmed
		{"16px", "16px"},
		{"red", "red"},
		{"url(bad)", ""},            // parens rejected
		{"red; color: blue", ""},   // semicolon rejected
		{"{background: red}", ""},  // braces rejected
		{"<script>", ""},           // angle brackets rejected
		{`"value"`, ""},            // double quote rejected
		{"it's", ""},               // single quote rejected
		{`back\slash`, ""},         // backslash rejected
		{"line\nbreak", ""},        // newline rejected
		{"line\rbreak", ""},        // carriage return rejected
		{"", ""},
	}
	for _, c := range cases {
		got := sanitizeValue(c.in)
		if got != c.want {
			t.Errorf("sanitizeValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildTheme(t *testing.T) {
	t.Run("empty theme produces empty output", func(t *testing.T) {
		got := BuildTheme(nil)
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
		got = BuildTheme(map[string]map[string]string{})
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("global vars written to :root block", func(t *testing.T) {
		theme := map[string]map[string]string{
			"global": {"--accent": "#e85d04"},
		}
		got := string(BuildTheme(theme))
		if !strings.Contains(got, ":root {") {
			t.Errorf("expected :root block, got %q", got)
		}
		if !strings.Contains(got, "--accent: #e85d04;") {
			t.Errorf("expected --accent declaration, got %q", got)
		}
	})

	t.Run("component styles written to #gs-<name> block", func(t *testing.T) {
		theme := map[string]map[string]string{
			"car-list": {"background": "#111"},
		}
		got := string(BuildTheme(theme))
		if !strings.Contains(got, "#gs-car-list {") {
			t.Errorf("expected #gs-car-list block, got %q", got)
		}
		if !strings.Contains(got, "background: #111;") {
			t.Errorf("expected background declaration, got %q", got)
		}
	})

	t.Run("dangerous prop is stripped", func(t *testing.T) {
		theme := map[string]map[string]string{
			"global": {"color: injected": "red"},
		}
		got := string(BuildTheme(theme))
		if strings.Contains(got, "injected") {
			t.Errorf("expected dangerous prop to be stripped, got %q", got)
		}
	})

	t.Run("dangerous value is stripped", func(t *testing.T) {
		theme := map[string]map[string]string{
			"global": {"color": "url(javascript:evil)"},
		}
		got := string(BuildTheme(theme))
		if strings.Contains(got, "javascript") {
			t.Errorf("expected dangerous value to be stripped, got %q", got)
		}
	})

	t.Run("global block omitted when empty after sanitize", func(t *testing.T) {
		theme := map[string]map[string]string{
			"global": {},
		}
		got := string(BuildTheme(theme))
		if strings.Contains(got, ":root") {
			t.Errorf("expected no :root block for empty global, got %q", got)
		}
	})
}
