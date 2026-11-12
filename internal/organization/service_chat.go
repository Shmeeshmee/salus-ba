package organization

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The defaults of a clinic that has stored no chat settings, and the ranges of each (docs/plans/chat.md, SAL-29).
const (
	defaultResponseTimeHours = 24
	defaultEmailAfterMinutes = 60
	maxResponseTimeHours     = 168         // a week
	minEmailAfterMinutes     = 5           // a reader in the app has read it by then
	maxEmailAfterMinutes     = 7 * 24 * 60 // a week
	maxChatHours             = 21          // three stretches a day
)

// days are the FHIR days-of-week codes, in time.Weekday order.
var days = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

// Consents gives the state of a patient's consents with a clinic, in the clinic's scope (consent.Service).
type Consents interface {
	State(ctx context.Context, patientID, organizationID string) (access.Consent, error)
}

// ChatPorts are what Chat runs on.
type ChatPorts struct {
	Repo     ChatRepository
	Consents Consents
	Audit    Auditor
	Tx       Transactor
	Clock    clock.Clock
}

// Chat holds the clinic's chat settings: only its admin reads and changes them (a change is audited, also when
// refused); its staff and the patients with an active care consent with it read its hours.
type Chat struct{ ChatPorts }

// NewChat returns Chat on its ports.
func NewChat(p ChatPorts) *Chat { return &Chat{ChatPorts: p} }

// Settings returns the settings of the admin's clinic, the defaults when it has stored none. ErrForbidden for its
// other staff, ErrNotFound for anyone else.
func (s *Chat) Settings(ctx context.Context, c Caller) (ChatSettings, error) {
	if err := adminOf(c); err != nil {
		return ChatSettings{}, err
	}
	var out ChatSettings
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = s.settings(ctx, c.OrganizationID)
		return err
	})
	if err != nil {
		return ChatSettings{}, fmt.Errorf("organization: chat settings: %w", err)
	}
	return out, nil
}

// SetSettings replaces the settings of the admin's clinic with in, audited as an update of the clinic; refusals as
// Settings', audited too. ErrInvalidChatSettings for settings out of their ranges.
func (s *Chat) SetSettings(ctx context.Context, c Caller, in ChatSettings) (ChatSettings, error) {
	if err := adminOf(c); err != nil {
		if aerr := s.Tx.WithTx(ctx, func(ctx context.Context) error {
			return s.Audit.Record(ctx, chatEvent(c, auditevent.OutcomeMinorFailure))
		}); aerr != nil {
			return ChatSettings{}, fmt.Errorf("organization: chat settings: %w", aerr)
		}
		return ChatSettings{}, err
	}
	if err := validChatSettings(in); err != nil {
		return ChatSettings{}, err
	}
	if in.Hours == nil {
		in.Hours = []ChatHour{}
	}
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, c.OrganizationID); err != nil {
			return err
		}
		if err := s.Repo.PutChatSettings(ctx, c.OrganizationID, in, s.Clock.Now()); err != nil {
			return err
		}
		return s.Audit.Record(ctx, chatEvent(c, auditevent.OutcomeSuccess))
	})
	if err != nil {
		return ChatSettings{}, fmt.Errorf("organization: chat settings: %w", err)
	}
	return in, nil
}

// Hours returns the clinic id's chat hours to its staff and to a patient with an active care consent with it, and
// whether it is within them now in its time zone (always, when it states none). ErrNotFound to anyone else: the
// clinic's existence is not disclosed. Not audited: nothing in it is personal data.
func (s *Chat) Hours(ctx context.Context, c Caller, id string) (ChatHours, error) {
	staff := c.OrganizationID != "" && c.OrganizationID == id
	if !staff && (c.OrganizationID != "" || c.PatientID == "") {
		return ChatHours{}, ErrNotFound
	}
	var out ChatHours
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		// The clinic's scope: its consents decide a patient's access, then its row and settings are read.
		if err := s.Repo.SetOrganization(ctx, id); err != nil {
			return err
		}
		if !staff {
			state, err := s.Consents.State(ctx, c.PatientID, id)
			if err != nil {
				return err
			}
			if !state.Active {
				return ErrNotFound
			}
		}
		clinic, err := s.Repo.ByID(ctx, id)
		if err != nil {
			return err
		}
		set, err := s.settings(ctx, id)
		if err != nil {
			return err
		}
		loc, err := time.LoadLocation(clinic.TimeZone)
		if err != nil {
			return fmt.Errorf("the clinic's time zone: %w", err)
		}
		out = ChatHours{TimeZone: clinic.TimeZone, Hours: set.Hours, ResponseTimeHours: set.ResponseTimeHours, OpenNow: openAt(set.Hours, s.Clock.Now().In(loc))}
		return nil
	})
	switch {
	case errors.Is(err, ErrNotFound):
		return ChatHours{}, ErrNotFound
	case err != nil:
		return ChatHours{}, fmt.Errorf("organization: chat hours: %w", err)
	}
	return out, nil
}

// EmailAfter is how long a message of the clinic org stays unread before its reader gets an e-mail (the notifier's
// fallback), in the clinic's scope. Unaudited: no personal data.
func (s *Chat) EmailAfter(ctx context.Context, org string) (time.Duration, error) {
	var set ChatSettings
	err := s.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.Repo.SetOrganization(ctx, org); err != nil {
			return err
		}
		var err error
		set, err = s.settings(ctx, org)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("organization: e-mail delay: %w", err)
	}
	return time.Duration(set.EmailAfterMinutes) * time.Minute, nil
}

// settings reads the clinic's settings in the transaction in ctx, setting its scope, or the defaults.
func (s *Chat) settings(ctx context.Context, org string) (ChatSettings, error) {
	if err := s.Repo.SetOrganization(ctx, org); err != nil {
		return ChatSettings{}, err
	}
	set, ok, err := s.Repo.ChatSettings(ctx, org)
	switch {
	case err != nil:
		return ChatSettings{}, err
	case !ok:
		return ChatSettings{Hours: []ChatHour{}, ResponseTimeHours: defaultResponseTimeHours, EmailAfterMinutes: defaultEmailAfterMinutes}, nil
	case set.Hours == nil:
		set.Hours = []ChatHour{}
	}
	return set, nil
}

// adminOf is nil for the admin of a clinic, ErrForbidden for its other staff and ErrNotFound for anyone else.
func adminOf(c Caller) error {
	switch {
	case c.OrganizationID == "":
		return ErrNotFound
	case !c.Admin:
		return ErrForbidden
	}
	return nil
}

// chatEvent is the audit of a change of the chat settings by c of their clinic.
func chatEvent(c Caller, outcome auditevent.Outcome) auditevent.AuditEvent {
	id := c.OrganizationID
	if outcome != auditevent.OutcomeSuccess {
		id = ""
	}
	return event(auditevent.CodeUpdate, auditevent.ActionUpdate, c, id, outcome)
}

// validChatSettings is nil for settings within their ranges whose hours are valid, non-overlapping stretches.
func validChatSettings(in ChatSettings) error {
	if in.ResponseTimeHours < 1 || in.ResponseTimeHours > maxResponseTimeHours ||
		in.EmailAfterMinutes < minEmailAfterMinutes || in.EmailAfterMinutes > maxEmailAfterMinutes || len(in.Hours) > maxChatHours {
		return ErrInvalidChatSettings
	}
	type stretch struct{ day, start, end int }
	stretches := make([]stretch, 0, len(in.Hours))
	for _, h := range in.Hours {
		day := slices.Index(days, h.Day)
		start, okStart := minuteOf(h.Start)
		end, okEnd := minuteOf(h.End)
		if day < 0 || !okStart || !okEnd || start >= end {
			return ErrInvalidChatSettings
		}
		stretches = append(stretches, stretch{day, start, end})
	}
	slices.SortFunc(stretches, func(a, b stretch) int { return (a.day*1440 + a.start) - (b.day*1440 + b.start) })
	for i := 1; i < len(stretches); i++ {
		if stretches[i].day == stretches[i-1].day && stretches[i].start < stretches[i-1].end {
			return ErrInvalidChatSettings
		}
	}
	return nil
}

// minuteOf is the minute of the day of HH:MM, 00:00 to 24:00.
func minuteOf(s string) (int, bool) {
	if len(s) != 5 || s[2] != ':' {
		return 0, false
	}
	for _, i := range []int{0, 1, 3, 4} {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	h, m := int(s[0]-'0')*10+int(s[1]-'0'), int(s[3]-'0')*10+int(s[4]-'0')
	if m > 59 || h > 24 || (h == 24 && m != 0) {
		return 0, false
	}
	return h*60 + m, true
}

// openAt tells whether t, in the clinic's zone, falls within one of hours; with no hours, the clinic states none and
// is open.
func openAt(hours []ChatHour, t time.Time) bool {
	if len(hours) == 0 {
		return true
	}
	now := t.Hour()*60 + t.Minute()
	for _, h := range hours {
		start, _ := minuteOf(h.Start)
		end, _ := minuteOf(h.End)
		if h.Day == days[t.Weekday()] && start <= now && now < end {
			return true
		}
	}
	return false
}
