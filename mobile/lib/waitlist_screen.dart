import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' as api;

import 'booking.dart';
import 'fhir.dart';
import 'l10n/app_localizations.dart';
import 'loading.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

/// How far ahead a patient may wait: the API takes 90 days at most; the app offers these.
const waitlistSpans = [7, 14, 30, 90];

/// The times of day a patient may wait for, in the clinic's zone: any, the morning, the afternoon.
enum DayPart { any, morning, afternoon }

/// The waitlist's calls (SAL-35), every failure an [ApiFailure].
class Waitlists {
  Waitlists(this._api);

  final api.WaitlistApi _api;

  /// The patient's entries, newest first, each with its open offer.
  Future<List<api.WaitlistEntry>> mine() async => (await apiCall(_api.listMyWaitlist))?.entries ?? const [];

  /// Joins the waitlist of [clinic] for [role] from [today] (the clinic's date) for [days], at [part] of the day.
  Future<void> join(String clinic, String role, DateTime today, int days, DayPart part) => apiCall(
    () => _api.joinWaitlist(
      api.NewWaitlistEntry(
        organizationId: clinic,
        practitionerRoleId: role,
        earliest: today,
        latest: DateTime.utc(today.year, today.month, today.day + days),
        times: switch (part) {
          DayPart.any => null,
          DayPart.morning => api.WaitlistTimes(start: '08:00', end: '12:00'),
          DayPart.afternoon => api.WaitlistTimes(start: '12:00', end: '17:00'),
        },
      ),
    ),
  );

  Future<void> withdraw(String id) => apiCall(() => _api.removeWaitlistEntry(id));
  Future<void> accept(String offer) => apiCall(() => _api.acceptWaitlistOffer(offer));
  Future<void> decline(String offer) => apiCall(() => _api.declineWaitlistOffer(offer));
}

/// The waitlist's refusals, each with its fixed sentence; [bookingMessage] for the rest.
String waitlistMessage(AppLocalizations t, ApiFailure failure) => switch (failure.code) {
  'waitlist_off' => t.waitlistOff,
  'too_many_entries' => t.waitlistTooMany,
  'offer_expired' => t.waitlistOfferExpired,
  _ => bookingMessage(t, failure),
};

/// Joining a practitioner's waitlist at a clinic: how far ahead and at what time of day. The care consent is needed
/// first, as for a booking; then the patient waits, and a freed time is offered to them in turn.
class JoinWaitlistScreen extends StatefulWidget {
  const JoinWaitlistScreen({
    super.key,
    required this.booking,
    required this.now,
    required this.clinic,
    required this.role,
  });

  final Booking booking;
  final DateTime Function() now;
  final Organization clinic;
  final PractitionerRole role;

  @override
  State<JoinWaitlistScreen> createState() => _JoinWaitlistScreenState();
}

class _JoinWaitlistScreenState extends State<JoinWaitlistScreen> {
  int _days = waitlistSpans[1];
  DayPart _part = DayPart.any;
  bool _busy = false;
  String? _error;

  Future<void> _join() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final today = Zone(widget.clinic.timeZone).day(widget.now());
      await widget.booking.waitlists.join(widget.clinic.id, widget.role.id, today, _days, _part);
      if (!mounted) return;
      final navigator = Navigator.of(context);
      final joined = context.t.waitlistJoined;
      navigator.pop();
      await navigator.push(
        MaterialPageRoute<void>(
          builder: (_) => WaitlistScreen(booking: widget.booking, now: widget.now, notice: joined),
        ),
      );
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _error = waitlistMessage(context.t, failure));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return ui.Page(
      title: t.waitlistJoinTitle,
      children: [
        Text(specialtyAtClinic(t, specialtyName(t, widget.role), widget.clinic.name)),
        Text(widget.role.name),
        Text(t.waitlistExplained),
        ui.Heading(t.waitlistHowLong),
        Wrap(
          spacing: 8,
          children: [
            for (final days in waitlistSpans)
              ChoiceChip(label: Text(t.waitlistDays(days)), selected: _days == days, onSelected: (_) => setState(() => _days = days)),
          ],
        ),
        ui.Heading(t.waitlistWhen),
        Wrap(
          spacing: 8,
          children: [
            for (final part in DayPart.values)
              ChoiceChip(label: Text(_partName(t, part)), selected: _part == part, onSelected: (_) => setState(() => _part = part)),
          ],
        ),
        Text(t.timesIn(widget.clinic.timeZone)),
        ui.SubmitButton(t.waitlistJoin, busy: _busy, onPressed: _join),
        if (_error != null) ui.Message(_error!, error: true),
      ],
    );
  }
}

String _partName(AppLocalizations t, DayPart part) => switch (part) {
  DayPart.any => t.waitlistAnyTime,
  DayPart.morning => t.waitlistMornings,
  DayPart.afternoon => t.waitlistAfternoons,
};

typedef _Waitlist = ({List<api.WaitlistEntry> entries, Places places});

/// The patient's waitlist: what they wait for and until when, and a freed time offered to them, which they accept
/// (it books the time) or decline before its hold ends; an entry may be withdrawn.
class WaitlistScreen extends StatefulWidget {
  const WaitlistScreen({super.key, required this.booking, required this.now, this.notice});

  final Booking booking;
  final DateTime Function() now;

  /// What the previous screen did, such as joining.
  final String? notice;

  @override
  State<WaitlistScreen> createState() => _WaitlistScreenState();
}

class _WaitlistScreenState extends LoadingState<WaitlistScreen, _Waitlist> {
  late String? _message = widget.notice;
  String? _error;
  bool _busy = false;

  @override
  Future<_Waitlist> load() async {
    final entries = await widget.booking.waitlists.mine();
    return (entries: entries, places: await widget.booking.places(entries.map((e) => e.organizationId)));
  }

  Future<void> _act(Future<void> Function() call, String done) async {
    setState(() {
      _busy = true;
      _error = null;
      _message = null;
    });
    try {
      await call();
      if (mounted) setState(() => _message = done);
      await reload();
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _error = waitlistMessage(context.t, failure));
      await reload();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return ui.Page(
      title: t.waitlistTitle,
      children: [
        if (_message != null) ui.Message(_message!),
        if (_error != null) ui.Message(_error!, error: true),
        ...body((data) {
          final active = [
            for (final e in data.entries)
              if (e.status == api.WaitlistStatus.active) e,
          ];
          if (active.isEmpty) return [Text(t.waitlistEmpty)];
          return [for (final e in active) _entry(t, e, data.places)];
        }),
      ],
    );
  }

  Widget _entry(AppLocalizations t, api.WaitlistEntry e, Places places) {
    final clinic = places.clinics[e.organizationId];
    final role = places.roles[e.practitionerRoleId]?.$2;
    final zone = Zone(clinic?.timeZone ?? '');
    final offer = e.offer;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(role?.name ?? t.waitlistAPractitioner, style: Theme.of(context).textTheme.titleMedium),
            if (clinic != null) Text(clinic.name),
            Text(t.waitlistUntil(Zone.longDate(e.latest, t.localeName))),
            if (offer != null) ...[
              const SizedBox(height: 12),
              ui.Heading(t.waitlistOfferTitle),
              Text(zone.date(offer.slotStart, t.localeName)),
              Text(t.timeRange(zone.time(offer.slotStart), zone.time(offer.slotEnd), zone.name)),
              Text(t.waitlistHeldUntil(zone.time(offer.expiresAt))),
              Text(t.waitlistAcceptEffect),
              const SizedBox(height: 8),
              ui.SubmitButton(
                t.waitlistAccept,
                busy: _busy,
                onPressed: () => _act(() => widget.booking.waitlists.accept(offer.id), t.waitlistBooked),
              ),
              OutlinedButton(
                onPressed: _busy ? null : () => _act(() => widget.booking.waitlists.decline(offer.id), t.waitlistDeclined),
                child: Text(t.waitlistDecline),
              ),
            ],
            TextButton(
              onPressed: _busy ? null : () => _act(() => widget.booking.waitlists.withdraw(e.id), t.waitlistWithdrawn),
              child: Text(t.waitlistWithdraw),
            ),
          ],
        ),
      ),
    );
  }
}
