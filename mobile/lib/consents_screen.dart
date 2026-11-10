import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' show Consent, ConsentKind, ConsentStatusEnum;

import 'booking.dart';
import 'fhir.dart';
import 'loading.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

typedef _Consents = ({List<Consent> all, Map<String, Organization> clinics});

/// The patient's consents, of both kinds, each revocable while active: revoking the care consent cancels the future
/// appointments with that clinic (docs/booking/SPEC.md "Revocation"); withdrawing the transcription consent stops
/// the transcription, translation and AI assistance there and changes nothing else (SAL-12).
class ConsentsScreen extends StatefulWidget {
  const ConsentsScreen({super.key, required this.booking});

  final Booking booking;

  @override
  State<ConsentsScreen> createState() => _ConsentsScreenState();
}

class _ConsentsScreenState extends LoadingState<ConsentsScreen, _Consents> {
  String? _message;
  String? _error;

  @override
  Future<_Consents> load() async {
    final all = await widget.booking.consents();
    if (all.isEmpty) return (all: all, clinics: const <String, Organization>{});
    return (all: all, clinics: {for (final c in await widget.booking.fhir.listedClinics()) c.id: c});
  }

  Future<void> _revoke(Consent consent, String clinic) async {
    final t = context.t;
    final transcription = consent.kind == ConsentKind.transcription;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(transcription ? t.stopTranscriptionTitle : t.revokeTitle),
        content: Text(transcription ? t.stopTranscriptionBody(clinic) : t.revokeBody(clinic)),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: Text(transcription ? t.keepTranscription : t.keepConsent),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: Text(transcription ? t.stopTranscription : t.revokeConsent),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    setState(() {
      _message = null;
      _error = null;
    });
    try {
      await widget.booking.revoke(consent.id);
      if (!mounted) return;
      setState(() => _message = transcription ? context.t.transcriptionStopped : context.t.consentRevoked);
      await reload();
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _error = bookingMessage(context.t, failure));
    }
  }

  @override
  Widget build(BuildContext context) => ui.Page(
    title: context.t.myConsents,
    children: [
      if (_message != null) ui.Message(_message!),
      if (_error != null) ui.Message(_error!, error: true),
      ...body(
        (data) => [
          Text(data.all.isEmpty ? context.t.noConsents : context.t.consentsIntro),
          for (final consent in data.all) _card(consent, data.clinics[consent.organizationId]),
        ],
      ),
    ],
  );

  Widget _card(Consent consent, Organization? clinic) {
    final zone = Zone(clinic?.timeZone ?? 'UTC');
    final t = context.t;
    final name = clinic?.name ?? t.unlistedClinic;
    final revokedAt = consent.revokedAt;
    final active = consent.status == ConsentStatusEnum.active;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ui.Heading(name),
            Text(consent.kind == ConsentKind.transcription ? t.consentForTranscription : t.consentForCare),
            Text(t.givenOn(zone.date(consent.dateTime, t.localeName))),
            Text(active || revokedAt == null ? t.consentActive : t.revokedOn(zone.date(revokedAt, t.localeName))),
            if (active) ...[
              const SizedBox(height: 8),
              OutlinedButton(onPressed: () => _revoke(consent, name), child: Text(t.revokeConsent)),
            ],
          ],
        ),
      ),
    );
  }
}
