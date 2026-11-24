import 'package:flutter/material.dart';
import 'package:salus_api/api.dart'
    show
        DataRequest,
        DataRequestAnswerCode,
        DataRequestExport,
        DataRequestExportStatus,
        DataRequestExtensionReason,
        DataRequestKind,
        DataRequestStatus;

import 'booking.dart';
import 'close_account_screen.dart';
import 'data_request_screen.dart';
import 'fhir.dart';
import 'files.dart';
import 'l10n/app_localizations.dart';
import 'loading.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

typedef _Requests = ({
  List<DataRequest> all,
  Map<String, Organization> clinics,
  Map<String, DataRequestExport> exports,
});

/// Where a request to the Salus account is answered, and a clinic without a zone.
const _salusZone = 'Europe/Sarajevo';

/// The patient's privacy (docs/clinic-cms/WAVE5.md, D6): their data requests with their state and due date, the
/// export of an access or portability request to download while it is ready, a new request, and closing the account.
class PrivacyScreen extends StatefulWidget {
  const PrivacyScreen({super.key, required this.booking, required this.files, required this.onAccountClosed});

  final Booking booking;
  final Files files;

  /// The account is closed: the session ends on the device.
  final VoidCallback onAccountClosed;

  @override
  State<PrivacyScreen> createState() => _PrivacyScreenState();
}

class _PrivacyScreenState extends LoadingState<PrivacyScreen, _Requests> {
  String? _message;
  String? _error;
  String? _busy;

  @override
  Future<_Requests> load() async {
    final privacy = widget.booking.privacy;
    final all = await privacy.mine();
    final withExport = all.where(_hasExport).toList();
    final (clinics, exports) = await (
      all.any((r) => r.organizationId != null) ? widget.booking.fhir.listedClinics() : Future.value(<Organization>[]),
      Future.wait(withExport.map((r) => privacy.export(r.id))),
    ).wait;
    return (
      all: all,
      clinics: {for (final c in clinics) c.id: c},
      exports: {for (var i = 0; i < withExport.length; i++) withExport[i].id: ?exports[i]},
    );
  }

  /// An access or portability request that may have an export: one not withdrawn or refused.
  static bool _hasExport(DataRequest r) =>
      (r.kind == DataRequestKind.access || r.kind == DataRequestKind.portability) &&
      r.status != DataRequestStatus.withdrawn &&
      r.status != DataRequestStatus.refused;

  static bool _open(DataRequest r) =>
      r.status == DataRequestStatus.received || r.status == DataRequestStatus.inProgress;

  String _date(DataRequest r, DateTime instant) {
    final zone = data?.clinics[r.organizationId]?.timeZone ?? '';
    return Zone(zone.isEmpty ? _salusZone : zone).date(instant, context.t.localeName);
  }

  void _clear() => setState(() {
    _message = null;
    _error = null;
  });

  Future<void> _new() async {
    _clear();
    final filed = await Navigator.of(context)
        .push(MaterialPageRoute<DataRequest>(builder: (_) => DataRequestScreen(booking: widget.booking)));
    if (filed == null || !mounted) return;
    await reload();
    if (mounted) setState(() => _message = context.t.dataRequestSent(_date(filed, filed.dueAt)));
  }

  Future<void> _withdraw(DataRequest request) async {
    final t = context.t;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(t.withdrawDataRequestTitle),
        content: Text(t.withdrawDataRequestBody),
        actions: [
          TextButton(onPressed: () => Navigator.of(context).pop(false), child: Text(t.keepDataRequest)),
          FilledButton(onPressed: () => Navigator.of(context).pop(true), child: Text(t.withdrawDataRequest)),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    _clear();
    try {
      await widget.booking.privacy.withdraw(request);
      if (!mounted) return;
      await reload();
      if (mounted) setState(() => _message = context.t.dataRequestWithdrawn);
    } on ApiFailure catch (failure) {
      if (!mounted) return;
      final changed = failure.code == 'version_conflict';
      if (changed) await reload();
      if (!mounted) return;
      final t = context.t;
      setState(
        () => _error = switch (failure.code) {
          'data_request_closed' => t.dataRequestClosed,
          'version_conflict' => t.dataRequestChanged,
          _ => bookingMessage(t, failure),
        },
      );
    }
  }

  Future<void> _download(DataRequest request) async {
    _clear();
    setState(() => _busy = request.id);
    try {
      final bytes = await widget.booking.privacy.archive(request.id);
      await widget.files.shareFile('salus-export-${request.id}.zip', bytes, 'application/zip');
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _error = bookingMessage(context.t, failure));
    } on Exception {
      // The share sheet or the temporary copy failed on the device.
      if (mounted) setState(() => _error = context.t.somethingWrong);
    } finally {
      if (mounted) setState(() => _busy = null);
    }
  }

  void _closeAccount() {
    _clear();
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => CloseAccountScreen(privacy: widget.booking.privacy, onClosed: widget.onAccountClosed),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return ui.Page(
      title: t.privacy,
      children: [
        Text(t.privacyIntro),
        if (_message != null) ui.Message(_message!),
        if (_error != null) ui.Message(_error!, error: true),
        ...body(
          (data) => [
            FilledButton(onPressed: _new, child: Text(t.newDataRequest)),
            if (data.all.isEmpty) Text(t.noDataRequests),
            for (final request in data.all) _card(t, request, data),
            OutlinedButton(onPressed: _closeAccount, child: Text(t.closeMyAccount)),
          ],
        ),
      ],
    );
  }

  Widget _card(AppLocalizations t, DataRequest r, _Requests data) {
    final clinic = r.organizationId;
    final answer = r.answerCode;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ui.Heading(kindName(t, r.kind)),
            Text(clinic == null ? t.requestToSalus : t.requestToClinic(data.clinics[clinic]?.name ?? t.unlistedClinic)),
            Text(_status(t, r.status)),
            Text(t.dataRequestReceivedOn(_date(r, r.receivedAt))),
            if (_open(r)) Text(_deadline(t, r)),
            if (answer != null) Text(_answer(t, answer)),
            if (r.status == DataRequestStatus.refused) Text(t.answerComplaint),
            if (data.exports[r.id] case final export?) ..._export(t, r, export),
            if (_open(r) && r.filedByPatient) ...[
              const SizedBox(height: 8),
              OutlinedButton(onPressed: () => _withdraw(r), child: Text(t.withdrawDataRequest)),
            ],
          ],
        ),
      ),
    );
  }

  String _deadline(AppLocalizations t, DataRequest r) {
    final until = r.extendedUntil;
    if (until == null) return t.dataRequestDueBy(_date(r, r.dueAt));
    return r.extensionReason == DataRequestExtensionReason.manyRequests
        ? t.dataRequestExtendedMany(_date(r, until))
        : t.dataRequestExtendedComplex(_date(r, until));
  }

  List<Widget> _export(AppLocalizations t, DataRequest r, DataRequestExport export) => switch (export.status) {
    DataRequestExportStatus.queued => [Text(t.exportPreparing)],
    DataRequestExportStatus.failed => [Text(t.exportFailed)],
    DataRequestExportStatus.expired => [Text(t.exportExpired)],
    DataRequestExportStatus.ready => [
      if (export.expiresAt case final until?) Text(t.exportReadyUntil(_date(r, until))),
      const SizedBox(height: 8),
      FilledButton(onPressed: _busy == null ? () => _download(r) : null, child: Text(t.downloadExport)),
    ],
  };
}

String _status(AppLocalizations t, DataRequestStatus status) => switch (status) {
  DataRequestStatus.received => t.dataRequestReceived,
  DataRequestStatus.inProgress => t.dataRequestInProgress,
  DataRequestStatus.fulfilled => t.dataRequestFulfilled,
  DataRequestStatus.refused => t.dataRequestRefused,
  DataRequestStatus.withdrawn => t.dataRequestWithdrawnStatus,
};

String _answer(AppLocalizations t, DataRequestAnswerCode code) => switch (code) {
  DataRequestAnswerCode.done => t.answerDone,
  DataRequestAnswerCode.donePartlyKept => t.answerDonePartlyKept,
  DataRequestAnswerCode.identityUnverified => t.answerIdentityUnverified,
  DataRequestAnswerCode.manifestlyUnfounded => t.answerManifestlyUnfounded,
  DataRequestAnswerCode.excessive => t.answerExcessive,
  DataRequestAnswerCode.retentionLaw => t.answerRetentionLaw,
  DataRequestAnswerCode.noData => t.answerNoData,
};
