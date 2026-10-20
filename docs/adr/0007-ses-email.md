# 7. AWS SES for e-mail, Mailpit locally

Status: accepted, 2026-10-19

## Context

The platform sends appointment confirmations, reminders and account e-mails. Running our own mail server is
not worth the deliverability work. Developers need to see sent mail without sending real mail.

## Decision

E-mail goes through a port in the `notification` domain, sent by `cmd/notifier`. Production uses AWS SES in an
EU region; local and CI use Mailpit over SMTP. SMS and push come later behind the same domain.

## Consequences

- SES is a processor: a DPA and an entry in `docs/privacy/processors.md` are required before it is wired in.
- E-mail content is minimal: no clinical details, only what the recipient needs and a link into the app.
- The port has two implementations (SES and SMTP), which justifies the interface.
- Delivery is a River job, so a failed send is retried and never lost with its triggering write.
