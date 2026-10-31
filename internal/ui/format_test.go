package ui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{10 * 1024, "10.0 KiB"},
		{100 * 1024, "100 KiB"},
		{1 << 30, "1.0 GiB"},
		{42 << 30, "42.0 GiB"},
		{150 << 30, "150 GiB"},
		{1 << 50, "1.0 PiB"},
	}
	for _, tc := range tests {
		if got := HumanBytes(tc.in); got != tc.want {
			t.Errorf("HumanBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if got := HumanBytes(-1024); got != "-1.0 KiB" {
		t.Errorf("negative HumanBytes = %q", got)
	}
}

func TestFormatCount(t *testing.T) {
	tests := map[int64]string{
		0: "0", 999: "999", 1000: "1,000", 1234567: "1,234,567", -1234: "-1,234",
	}
	for in, want := range tests {
		if got := FormatCount(in); got != want {
			t.Errorf("FormatCount(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestPercent(t *testing.T) {
	if got := Percent(50, 200); got != 25 {
		t.Errorf("Percent = %v, want 25", got)
	}
	if got := Percent(300, 200); got != 100 {
		t.Errorf("Percent should clamp at 100, got %v", got)
	}
	if got := Percent(10, 0); got != 0 {
		t.Errorf("Percent with zero total = %v", got)
	}
}

func TestBar(t *testing.T) {
	if got := Bar(0.5, 10, false); got != "█████░░░░░" {
		t.Errorf("Bar = %q", got)
	}
	if got := Bar(0.5, 10, true); got != "#####-----" {
		t.Errorf("ASCII Bar = %q", got)
	}
	if got := Bar(0.001, 10, false); got != "█░░░░░░░░░" {
		t.Errorf("tiny Bar = %q", got)
	}
	if got := Bar(2, 4, false); got != "████" {
		t.Errorf("clamped Bar = %q", got)
	}
	if got := Bar(0.5, 0, false); got != "" {
		t.Errorf("zero width Bar = %q", got)
	}
}

func TestTruncateAndPad(t *testing.T) {
	if got := Truncate("abcdef", 4, true); got != "abc~" {
		t.Errorf("Truncate ASCII = %q", got)
	}
	if got := Truncate("abcdef", 4, false); got != "abc…" {
		t.Errorf("Truncate = %q", got)
	}
	if got := Truncate("abc", 4, false); got != "abc" {
		t.Errorf("short Truncate = %q", got)
	}
	if got := TruncateLeft("/a/b/c/d", 5, false); got != "…/c/d" {
		t.Errorf("TruncateLeft = %q (width %d)", got, ansi.StringWidth(got))
	}
	if got := PadRight("ab", 4, false); got != "ab  " {
		t.Errorf("PadRight = %q", got)
	}
	if got := PadLeft("ab", 4, false); got != "  ab" {
		t.Errorf("PadLeft = %q", got)
	}
	if got := PadRight("abcdef", 4, false); got != "abc…" {
		t.Errorf("PadRight truncation = %q", got)
	}
}

func TestFileModeShort(t *testing.T) {
	if got := FileModeShort(0o755); got != "rwxr-xr-x" {
		t.Errorf("FileModeShort(0755) = %q", got)
	}
	if got := FileModeShort(0o600); got != "rw-------" {
		t.Errorf("FileModeShort(0600) = %q", got)
	}
	if got := FileModeShort(os.ModeDir | 0o644); got != "rw-r--r--" {
		t.Errorf("FileModeShort(dir) = %q", got)
	}
}

func TestCountNounAndPercentFormatting(t *testing.T) {
	if got := CountNoun(0, "file", "files"); got != "0 files" {
		t.Errorf("CountNoun(0) = %q", got)
	}
	if got := CountNoun(1, "file", "files"); got != "1 file" {
		t.Errorf("CountNoun(1) = %q", got)
	}
	if got := CountNoun(1200, "file", "files"); got != "1,200 files" {
		t.Errorf("CountNoun(1200) = %q", got)
	}
	tests := map[float64]string{0: "0%", 0.04: "<0.1%", 0.4: "0.4%", 9.9: "9.9%", 28: "28%", 100: "100%"}
	for p, want := range tests {
		if got := FormatPercent(p); got != want {
			t.Errorf("FormatPercent(%v) = %q, want %q", p, got, want)
		}
	}
}

func TestFormatTime(t *testing.T) {
	if got := FormatTime(time.Time{}); got != "-" {
		t.Errorf("zero time = %q", got)
	}
	now := time.Now()
	got := FormatTime(now)
	if got == "-" || len(got) != len("Jan 02 15:04") {
		t.Errorf("FormatTime = %q", got)
	}
}

func TestWrapText(t *testing.T) {
	got := wrapText("one two three four five", 10)
	want := []string{"one two", "three four", "five"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("wrapText = %#v, want %#v", got, want)
	}
}
