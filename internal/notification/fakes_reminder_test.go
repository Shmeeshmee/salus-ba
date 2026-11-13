package notification_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/notification"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/patient"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/push"
	"github.com/Shmeeshmee/salus/pkg/push/pushtest"
	"github.com/Shmeeshmee/salus/pkg/sms/smstest"
)

// Synthetic ids of the reminders.
const (
	reminderID  = "e1000000-0000-4000-8000-0000000000e1"
	remindAppt  = "e2000000-0000-4000-8000-0000000000e2"
	remindClin  = "c2000000-0000-4000-8000-00000000002c"
	remindPhone = "+38761000001"
)

// fakeDeliveries holds deliveries, settings and SMS usage.
type fakeDeliveries struct {
	err        error
	prefs      map[string]notification.Preferences
	usage      map[string]notification.UsageMonth // by clinic and month
	repo       *fakeChatRepo
	deliveries []notification.Delivery
	usageScope []string // the clinic set for each usage write
}

func newFakeDeliveries(repo *fakeChatRepo) *fakeDeliveries {
	return &fakeDeliveries{prefs: map[string]notification.Preferences{}, usage: map[string]notification.UsageMonth{}, repo: repo}
}

func (f *fakeDeliveries) AddDelivery(ctx context.Context, d notification.Delivery) (notification.Delivery, bool, error) {
	if ctx.Value(chatTxKey{}) == nil {
		return notification.Delivery{}, false, errors.New("outside a transaction")
	}
	for _, e := range f.deliveries {
		if e.Kind == d.Kind && e.SourceID == d.SourceID && e.Channel == d.Channel && e.DeviceID == d.DeviceID {
			return notification.Delivery{}, false, f.err
		}
	}
	d.ID, d.State = fmt.Sprintf("d%07d-0000-4000-8000-000000000000", len(f.deliveries)+1), notification.DeliveryPending
	f.deliveries = append(f.deliveries, d)
	return d, true, f.err
}

func (f *fakeDeliveries) LockDelivery(_ context.Context, id string) (notification.Delivery, error) {
	for _, d := range f.deliveries {
		if d.ID == id {
			return d, f.err
		}
	}
	return notification.Delivery{}, notification.ErrNotFound
}

func (f *fakeDeliveries) SetDeliveryState(_ context.Context, id string, state notification.DeliveryState, at time.Time, segments int) error {
	for i, d := range f.deliveries {
		if d.ID != id {
			continue
		}
		if d.State != notification.DeliveryPending {
			return notification.ErrDeliveryDone
		}
		f.deliveries[i].State, f.deliveries[i].Segments = state, segments
		if state == notification.DeliverySent {
			f.deliveries[i].SentAt = at
		}
		return f.err
	}
	return notification.ErrNotFound
}

func (f *fakeDeliveries) DeleteDeliveriesBefore(context.Context, time.Time) (int64, error) {
	return 0, f.err
}

func (f *fakeDeliveries) Preferences(_ context.Context, account string) (notification.Preferences, bool, error) {
	p, ok := f.prefs[account]
	return p, ok, f.err
}

func (f *fakeDeliveries) SetPreferences(_ context.Context, account string, p notification.Preferences, _ time.Time) error {
	f.prefs[account] = p
	return f.err
}

func (f *fakeDeliveries) AddSMSUsage(_ context.Context, org string, month time.Time, segments int) error {
	f.usageScope = append(f.usageScope, f.repo.clinic)
	key := org + month.Format("2006-01")
	u := f.usage[key]
	u.Month, u.Messages, u.Segments = month.Format("2006-01"), u.Messages+1, u.Segments+segments
	f.usage[key] = u
	return f.err
}

func (f *fakeDeliveries) ClinicScope(context.Context) (string, error) { return f.repo.clinic, f.err }

func (f *fakeDeliveries) SMSUsage(_ context.Context, org string, from, to time.Time) ([]notification.UsageMonth, error) {
	var out []notification.UsageMonth
	for m := from; !m.After(to); m = m.AddDate(0, 1, 0) {
		if u, ok := f.usage[org+m.Format("2006-01")]; ok {
			out = append(out, u)
		}
	}
	return out, f.err
}

// fakeDeliveryJobs records "<channel> <delivery> at <time>".
type fakeDeliveryJobs struct{ enqueued []string }

func (j *fakeDeliveryJobs) EnqueueDelivery(ctx context.Context, channel notification.Channel, id string, at time.Time) error {
	if ctx.Value(chatTxKey{}) == nil {
		return errors.New("enqueued outside a transaction")
	}
	j.enqueued = append(j.enqueued, fmt.Sprintf("%s %s at %s", channel, id, at.UTC().Format(time.RFC3339)))
	return nil
}

// fakeReminders is one reminder of remindAppt at start for patientID, in remindClin.
type fakeReminders struct {
	takeErr, noticeErr, linkErr error
	start                       time.Time
	links                       []appointment.Link
	taken, dispatched           int
}

func (f *fakeReminders) notice() appointment.ReminderNotice {
	return appointment.ReminderNotice{Start: f.start, ReminderID: reminderID, AppointmentID: remindAppt, OrganizationID: remindClin, PatientID: patientID, AccountID: patientAccount}
}

func (f *fakeReminders) TakeReminder(_ context.Context, _, org, id string) (appointment.ReminderNotice, error) {
	f.taken++
	if org != remindClin || id != reminderID {
		return appointment.ReminderNotice{}, appointment.ErrNotFound
	}
	return f.notice(), f.takeErr
}

func (f *fakeReminders) ReminderDispatched(ctx context.Context, _, _ string) error {
	if ctx.Value(chatTxKey{}) == nil {
		return errors.New("outside a transaction")
	}
	f.dispatched++
	return nil
}

func (f *fakeReminders) ReminderNotice(_ context.Context, _, _, _ string) (appointment.ReminderNotice, error) {
	return f.notice(), f.noticeErr
}

func (f *fakeReminders) IssueLink(_ context.Context, _ string, l appointment.Link) (string, error) {
	f.links = append(f.links, l)
	return fakeLink(len(f.links)), f.linkErr
}

// fakeLink is the nth link the fake issues, as long as a real one (43 characters of token).
func fakeLink(n int) string { return fmt.Sprintf("https://api.salus.invalid/a/%043d", n) }

func clockAt(t time.Time) clock.Clock { return clock.Fixed{T: t} }

// fakeRules gives the clinic's SMS option (off when off is set) and, as organization.Appointments.For does, sets the
// clinic's scope on repo.
type fakeRules struct {
	repo *fakeChatRepo
	off  *bool
	sms  bool
}

func (r fakeRules) For(_ context.Context, org string) (organization.AppointmentSettings, error) {
	if r.repo != nil {
		r.repo.clinic = org
	}
	s := organization.DefaultAppointmentSettings()
	s.SMSReminders = r.sms && (r.off == nil || !*r.off)
	return s, nil
}

// sarajevo is the reminders' clinic, in Europe/Sarajevo.
type sarajevo struct{}

func (sarajevo) Read(_ context.Context, _ organization.Caller, id string) (organization.Organization, error) {
	return organization.Organization{ID: id, Name: "Synthetic clinic", TimeZone: "Europe/Sarajevo"}, nil
}

type fakePhones struct{ phone string }

func (p fakePhones) NoticePhone(context.Context, string, string, string) (string, error) {
	return p.phone, nil
}

type fakeProfiles struct {
	err   error
	phone string
}

func (p fakeProfiles) Me(context.Context, patient.Caller) (patient.Patient, error) {
	return patient.Patient{ID: patientID, Phone: p.phone}, p.err
}

// reminderFixture is the notifier's reminders on fakes at now.
type reminderFixture struct {
	repo       *fakeChatRepo
	deliveries *fakeDeliveries
	jobs       *fakeDeliveryJobs
	reminders  *fakeReminders
	audit      *fakeChatAudit
	sender     *fakeSender
	pusher     *pushtest.Sender
	sms        *smstest.Sender
	q          *fakeQuestionnaires
	svc        *notification.Service
}

func newReminderFixture(now, start time.Time, rules fakeRules) *reminderFixture {
	f := &reminderFixture{
		repo: newFakeChatRepo(), jobs: &fakeDeliveryJobs{}, reminders: &fakeReminders{start: start}, audit: &fakeChatAudit{},
		sender: &fakeSender{}, pusher: &pushtest.Sender{}, sms: &smstest.Sender{}, q: &fakeQuestionnaires{},
	}
	f.deliveries = newFakeDeliveries(f.repo)
	rules.repo = f.repo
	f.repo.devices["c3000000-0000-4000-8000-0000000000c3"] = notification.Device{ID: "c3000000-0000-4000-8000-0000000000c3", AccountID: patientAccount, Token: "synthetic-token-r", Platform: push.Android}
	f.svc = notification.NewService(notification.Ports{
		Accounts: &fakeChatAccounts{language: map[string]string{patientAccount: "en"}}, Clinics: sarajevo{}, Questionnaires: f.q, Sender: f.sender, PatientAppURL: "https://app.salus.invalid",
		Deliveries: f.deliveries, DeliveryJobs: f.jobs, Devices: f.repo, Reminders: f.reminders, Rules: rules,
		Languages: &fakeChatAccounts{}, Phones: fakePhones{phone: remindPhone}, Profiles: fakeProfiles{phone: remindPhone},
		Pusher: f.pusher, SMS: f.sms, Audit: f.audit, Tx: fakeChatTx{}, Clock: clock.Fixed{T: now},
	})
	return f
}
