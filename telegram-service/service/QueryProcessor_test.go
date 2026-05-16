package service

import (
	"testing"
)

func TestTypeTagRe_MatchesImageTag(t *testing.T) {
	query := "some query -type:image extra"
	match := typeTagRe.FindStringSubmatch(query)
	if match == nil {
		t.Fatal("expected match for -type:image, got nil")
	}
	if match[1] != "image" {
		t.Errorf("expected capture 'image', got %q", match[1])
	}
	cleaned := typeTagRe.ReplaceAllString(query, " ")
	if typeTagRe.MatchString(cleaned) {
		t.Errorf("expected tag stripped from %q, but still present", cleaned)
	}
}

func TestTypeTagRe_MatchesVideoTag(t *testing.T) {
	query := "-type:video"
	match := typeTagRe.FindStringSubmatch(query)
	if match == nil {
		t.Fatal("expected match for -type:video, got nil")
	}
	if match[1] != "video" {
		t.Errorf("expected capture 'video', got %q", match[1])
	}
}

func TestTypeTagRe_CaseInsensitive(t *testing.T) {
	for _, q := range []string{"-type:IMAGE", "-type:Video", "-type:IMAGE"} {
		if typeTagRe.FindStringSubmatch(q) == nil {
			t.Errorf("expected match for %q", q)
		}
	}
}

func TestTypeTagRe_NoMatchOnUnknownType(t *testing.T) {
	for _, q := range []string{"-type:gif", "-type:", "type:image"} {
		if typeTagRe.FindStringSubmatch(q) != nil {
			t.Errorf("expected no match for %q, but got one", q)
		}
	}
}

func TestTypeTagRe_QueryCleanedAfterStrip(t *testing.T) {
	cases := []struct {
		input    string
		wantType string
		wantRest string
	}{
		{"-type:image cat meme", "image", "cat meme"},
		{"cat meme -type:video", "video", "cat meme"},
		{"-type:image", "image", ""},
	}
	for _, tc := range cases {
		match := typeTagRe.FindStringSubmatch(tc.input)
		if match == nil {
			t.Errorf("input %q: expected match", tc.input)
			continue
		}
		got := typeTagRe.ReplaceAllString(tc.input, " ")
		gotTrimmed := got
		for len(gotTrimmed) > 0 && (gotTrimmed[0] == ' ' || gotTrimmed[len(gotTrimmed)-1] == ' ') {
			gotTrimmed = gotTrimmed[:len(gotTrimmed)-1]
			if len(gotTrimmed) > 0 && gotTrimmed[0] == ' ' {
				gotTrimmed = gotTrimmed[1:]
			}
		}
		if match[1] != tc.wantType {
			t.Errorf("input %q: type = %q, want %q", tc.input, match[1], tc.wantType)
		}
	}
}
