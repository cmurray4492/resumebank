package render

import (
	"fmt"
	"html/template"
	"time"
)

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"formatDate": func(t time.Time) string {
			return t.Format("January 2, 2006")
		},
		"derefInt": func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		},
		"matchPercent": func(similarity float64) string {
			pct := similarity * 100
			if pct < 0 {
				pct = 0
			}
			if pct > 100 {
				pct = 100
			}
			return fmt.Sprintf("%.0f%%", pct)
		},
		"formatSalary": func(min, max *int) string {
			switch {
			case min != nil && max != nil:
				return fmt.Sprintf("$%s – $%s", commaInt(*min), commaInt(*max))
			case min != nil:
				return fmt.Sprintf("$%s+", commaInt(*min))
			case max != nil:
				return fmt.Sprintf("Up to $%s", commaInt(*max))
			default:
				return "Not specified"
			}
		},
	}
}

func commaInt(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	rem := len(s) % 3
	if rem > 0 {
		out = append(out, s[:rem]...)
		if len(s) > rem {
			out = append(out, ',')
		}
	}
	for i := rem; i < len(s); i += 3 {
		out = append(out, s[i:i+3]...)
		if i+3 < len(s) {
			out = append(out, ',')
		}
	}
	return string(out)
}
