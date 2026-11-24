import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' show DataRequest, DataRequestKind;

import 'booking.dart';
import 'l10n/app_localizations.dart';
import 'loading.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;

/// The kind of a data request in plain words (GDPR Art. 15 to 21).
String kindName(AppLocalizations t, DataRequestKind kind) => switch (kind) {
  DataRequestKind.portability => t.dataRequestPortability,
  DataRequestKind.rectification => t.dataRequestRectification,
  DataRequestKind.restriction => t.dataRequestRestriction,
  DataRequestKind.objection => t.dataRequestObjection,
  DataRequestKind.erasure => t.dataRequestErasure,
  DataRequestKind.access => t.dataRequestAccess,
};

String _explanation(AppLocalizations t, DataRequestKind kind) => switch (kind) {
  DataRequestKind.portability => t.explainPortability,
  DataRequestKind.rectification => t.explainRectification,
  DataRequestKind.restriction => t.explainRestriction,
  DataRequestKind.objection => t.explainObjection,
  DataRequestKind.erasure => t.explainErasure,
  DataRequestKind.access => t.explainAccess,
};

/// The account target in the target choice: not a clinic id.
const _account = '';

/// A new request (docs/clinic-cms/WAVE5.md, D6): its kind, each explained, and who answers it: the patient's Salus
/// account, or their record at one of the clinics they gave a consent to (S1). Pops with the filed request.
class DataRequestScreen extends StatefulWidget {
  const DataRequestScreen({super.key, required this.booking});

  final Booking booking;

  @override
  State<DataRequestScreen> createState() => _DataRequestScreenState();
}

class _DataRequestScreenState extends LoadingState<DataRequestScreen, List<(String, String?)>> {
  DataRequestKind? _kind;
  String _target = _account;
  bool _busy = false;
  String? _error;

  /// The patient's clinics by id with their names; null for one no longer listed.
  @override
  Future<List<(String, String?)>> load() async {
    final ids = {for (final c in await widget.booking.consents()) c.organizationId};
    if (ids.isEmpty) return const [];
    final names = {for (final c in await widget.booking.fhir.listedClinics()) c.id: c.name};
    return [for (final id in ids) (id, names[id])];
  }

  Future<void> _send() async {
    final kind = _kind;
    if (kind == null) {
      setState(() => _error = context.t.chooseDataRequestKind);
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final filed = await widget.booking.privacy.file(kind, clinicId: _target == _account ? null : _target);
      if (mounted) Navigator.of(context).pop<DataRequest>(filed);
    } on ApiFailure catch (failure) {
      if (!mounted) return;
      final t = context.t;
      setState(() {
        _busy = false;
        _error = switch (failure.code) {
          'not_found' => t.noRecordAtClinic,
          'export_in_progress' => t.exportInProgress,
          _ => bookingMessage(t, failure),
        };
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return ui.Page(
      title: t.newDataRequest,
      children: [
        ...body(
          (clinics) => [
            ui.Heading(t.dataRequestKind),
            RadioGroup<DataRequestKind>(
              groupValue: _kind,
              onChanged: (k) => setState(() => _kind = k),
              child: Column(
                children: [
                  for (final kind in DataRequestKind.values)
                    RadioListTile<DataRequestKind>(
                      value: kind,
                      title: Text(kindName(t, kind)),
                      subtitle: Text(_explanation(t, kind)),
                    ),
                ],
              ),
            ),
            ui.Heading(t.dataRequestTarget),
            Text(t.dataRequestTargetHint),
            RadioGroup<String>(
              groupValue: _target,
              onChanged: (id) => setState(() => _target = id ?? _account),
              child: Column(
                children: [
                  RadioListTile<String>(value: _account, title: Text(t.dataRequestToSalusOption)),
                  for (final (id, name) in clinics)
                    RadioListTile<String>(value: id, title: Text(name ?? t.unlistedClinic)),
                ],
              ),
            ),
            if (_error != null) ui.Message(_error!, error: true),
            ui.SubmitButton(t.sendDataRequest, busy: _busy, onPressed: _send),
          ],
        ),
      ],
    );
  }
}
