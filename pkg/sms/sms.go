// Package sms is the port of text messages to a mobile number (docs/plans/appointments.md, ADR 0015): a Message to
// one E.164 number, sent by a provider's adapter. smstest has the in-memory fake, the default of every binary until
// the owner chooses an EU/EEA provider with a DPA (docs/privacy/processors.md) and adds its credentials; smshttp is a
// generic HTTPS adapter, off and not wired. It knows nothing of appointments or patients: the caller writes the body,
// which never holds anything clinical. Nothing here logs; errors never carry a number or a body.
package sms

import (
	"context"
	"errors"
	"strings"
)

// Message is one text message: the number in E.164 (a plus and 8 to 15 digits), the body, and the caller's reference
// (an opaque id: the provider's client reference, so a retried send can be recognised where the provider supports it).
type Message struct {
	To        string
	Body      string
	Reference string
}

// Sender delivers a Message and returns the provider's id of it.
type Sender interface {
	Send(ctx context.Context, m Message) (string, error)
}

var (
	// ErrInvalidMessage is returned for a number that is not E.164, an empty body, or a body longer than MaxSegments.
	ErrInvalidMessage = errors.New("sms: invalid message")
	// ErrRejected is returned when the provider refuses the message for good (a number it cannot reach, a body it
	// does not take): a retry fails the same way.
	ErrRejected = errors.New("sms: the provider rejected the message")
)

// MaxSegments is the most parts a message may take: a reminder fits in one or two.
const MaxSegments = 3

// The lengths of a part: a single GSM-7 message holds 160 characters, each part of a longer one 153 (the rest is the
// concatenation header); with any character outside GSM-7 the message is UCS-2, 70 and 67.
const (
	gsmSingle  = 160
	gsmPart    = 153
	ucs2Single = 70
	ucs2Part   = 67
)

// gsmBasic is the GSM 03.38 basic character set; gsmExtended the characters of its extension table, which take two.
const (
	gsmBasic    = "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !\"#¤%&'()*+,-./0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà"
	gsmExtended = "^{}\\[~]|€\f"
)

// Segments is how many parts body takes, as the networks count them: GSM-7 when every character is in its sets (an
// extension character counting two), UCS-2 otherwise (a character outside the Basic Multilingual Plane counting two).
// Diacritics such as č, ć and đ are not GSM-7. Zero for an empty body.
func Segments(body string) int {
	if body == "" {
		return 0
	}
	gsm, units := true, 0
	for _, r := range body {
		switch {
		case strings.ContainsRune(gsmBasic, r):
			units++
		case strings.ContainsRune(gsmExtended, r):
			units += 2
		default:
			gsm = false
		}
	}
	single, part := gsmSingle, gsmPart
	if !gsm {
		single, part, units = ucs2Single, ucs2Part, 0
		for _, r := range body {
			units++
			if r > 0xFFFF {
				units++
			}
		}
	}
	if units <= single {
		return 1
	}
	return (units + part - 1) / part
}

// ValidNumber tells whether to is an E.164 number: a plus, then 8 to 15 digits, the first not zero.
func ValidNumber(to string) bool {
	digits, ok := strings.CutPrefix(to, "+")
	if !ok || len(digits) < 8 || len(digits) > 15 || digits[0] == '0' {
		return false
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// Check is nil for a message a sender may hand on: an E.164 number and a body of 1 to MaxSegments parts.
func Check(m Message) error {
	if n := Segments(m.Body); !ValidNumber(m.To) || n == 0 || n > MaxSegments {
		return ErrInvalidMessage
	}
	return nil
}
