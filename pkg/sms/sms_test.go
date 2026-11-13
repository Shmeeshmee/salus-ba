package sms_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/sms"
)

func TestSegments_CountsGSM7AndUCS2PartsAsTheNetworksDo(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		body string
		want int
	}{
		{"empty", "", 0},
		{"one GSM-7 part", strings.Repeat("a", 160), 1},
		{"two GSM-7 parts", strings.Repeat("a", 161), 2},
		{"three GSM-7 parts", strings.Repeat("a", 307), 3},
		{"an extension character counts two", strings.Repeat("a", 159) + "€", 2},
		{"a diacritic makes it UCS-2", strings.Repeat("a", 69) + "č", 1},
		{"two UCS-2 parts", strings.Repeat("a", 70) + "đ", 2},
		{"outside the BMP counts two", strings.Repeat("a", 69) + "😀", 2},
		{"GSM-7 accents", "Termin: ÄÖÑÜ é è à", 1},
	} {
		if got := sms.Segments(tt.body); got != tt.want {
			t.Errorf("%s: Segments() = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestValidNumber_AcceptsE164Only(t *testing.T) {
	t.Parallel()
	for to, want := range map[string]bool{
		"+38760000001": true, "+12025550123": true, "38760000001": false, "+0387600001": false, "+3876": false,
		"+3876000000000001": false, "+387 60 000 001": false, "": false, "+3876000000a": false,
	} {
		if got := sms.ValidNumber(to); got != want {
			t.Errorf("ValidNumber(%q) = %t, want %t", to, got, want)
		}
	}
}

func TestCheck_RefusesWhatNoSenderShouldSend(t *testing.T) {
	t.Parallel()
	if err := sms.Check(sms.Message{To: "+38760000001", Body: "Reminder"}); err != nil {
		t.Errorf("a valid message: %v", err)
	}
	for name, m := range map[string]sms.Message{
		"no number":      {Body: "x"},
		"empty body":     {To: "+38760000001"},
		"four parts":     {To: "+38760000001", Body: strings.Repeat("a", 4*153)},
		"not E.164":      {To: "060000001", Body: "x"},
		"spaces in body": {To: "+38760000001 ", Body: "x"},
	} {
		if err := sms.Check(m); !errors.Is(err, sms.ErrInvalidMessage) {
			t.Errorf("%s: Check() = %v, want ErrInvalidMessage", name, err)
		}
	}
}
