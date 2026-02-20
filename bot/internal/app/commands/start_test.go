package commands

import (
	"strings"
	"testing"
)

func TestBuildStartInfo(t *testing.T) {
	t.Parallel()

	text := buildStartInfo("operator.user")

	mustContain := []string{
		"ERPNext ga ulandi: operator.user",
		"/batch",
		"/log",
		"/epc",
		"/calibrate",
		"Nechta draft EPC bilan ketganini ko'rish",
		"Fayl captionida umumiy son chiqadi",
	}
	for _, part := range mustContain {
		if !strings.Contains(text, part) {
			t.Fatalf("start info missing %q\ntext:\n%s", part, text)
		}
	}
}
