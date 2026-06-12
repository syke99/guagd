package css

import (
	"fmt"
	"html/template"
	"strings"
	"unicode"
)

func BuildTheme(theme map[string]map[string]string) template.CSS {
	var sb strings.Builder
	if global, ok := theme["global"]; ok && len(global) > 0 {
		sb.WriteString(":root {\n")
		for prop, val := range global {
			if p := sanitizeProp(prop); p != "" {
				if v := sanitizeValue(val); v != "" {
					fmt.Fprintf(&sb, "  %s: %s;\n", p, v)
				}
			}
		}
		sb.WriteString("}\n")
	}
	for component, styles := range theme {
		if component == "global" || len(styles) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "#gs-%s {\n", component)
		for prop, val := range styles {
			if p := sanitizeProp(prop); p != "" {
				if v := sanitizeValue(val); v != "" {
					fmt.Fprintf(&sb, "  %s: %s;\n", p, v)
				}
			}
		}
		sb.WriteString("}\n")
	}
	return template.CSS(sb.String())
}

func sanitizeProp(p string) string {
	for _, ch := range p {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '-' {
			return ""
		}
	}
	return p
}

func sanitizeValue(v string) string {
	for _, dangerous := range []string{"(", ")", ";", "{", "}", "<", ">", `"`, "'", `\`, "\n", "\r"} {
		if strings.Contains(v, dangerous) {
			return ""
		}
	}
	return strings.TrimSpace(v)
}
