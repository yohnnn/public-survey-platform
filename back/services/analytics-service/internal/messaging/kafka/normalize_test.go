package kafka

import "testing"

func TestNormalizeGender(t *testing.T) {
	cases := map[string]string{
		"male":    "male",
		"Female":  "female",
		"Мужской": "male",
		"m":       "male",
		"other":   "",
		"другое":  "",
		"":        "",
	}
	for input, want := range cases {
		if got := normalizeGender(input); got != want {
			t.Fatalf("normalizeGender(%q)=%q want %q", input, got, want)
		}
	}
}
