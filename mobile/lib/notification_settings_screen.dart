import 'dart:io' show HttpStatus;

import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' as api;

import 'booking.dart';
import 'l10n/app_localizations.dart';
import 'loading.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

/// The quiet hours the screen offers when the patient turns them on (docs/plans/appointments.md "Quiet hours").
const _defaultQuiet = (start: '22:00', end: '08:00');

/// The device's IANA zone when Dart's [DateTime.timeZoneName] is one; on most phones it is an abbreviation (CET),
/// which says nothing certain about the zone, so there is none.
String? deviceTimeZone() {
  final name = DateTime.now().timeZoneName;
  return Zone.isIana(name) ? name : null;
}

/// How the patient's clinics remind them of appointments (SAL-33): push, e-mail and SMS, the quiet hours (HH:MM in
/// the patient's zone, which may wrap midnight), and the device's zone, sent with every save.
class NotificationSettingsScreen extends StatefulWidget {
  const NotificationSettingsScreen({super.key, required this.booking, this.deviceZone = deviceTimeZone});

  final Booking booking;

  /// The device's IANA zone, or null when the device gives none: the zone stored before is kept then.
  final String? Function() deviceZone;

  @override
  State<NotificationSettingsScreen> createState() => _NotificationSettingsScreenState();
}

class _NotificationSettingsScreenState extends LoadingState<NotificationSettingsScreen, api.NotificationSettings> {
  var _push = true;
  var _email = true;
  var _sms = false;
  var _quiet = true;
  var _start = _defaultQuiet.start;
  var _end = _defaultQuiet.end;
  var _busy = false;
  String? _message;
  String? _error;

  @override
  Future<api.NotificationSettings> load() async {
    final settings = await apiCall(widget.booking.notifications.getNotificationSettings);
    if (settings == null) throw const ApiFailure(status: HttpStatus.badGateway);
    _show(settings);
    return settings;
  }

  void _show(api.NotificationSettings s) {
    _push = s.channels.push;
    _email = s.channels.email;
    _sms = s.channels.sms;
    _quiet = s.quietHours != null;
    _start = s.quietHours?.start ?? _defaultQuiet.start;
    _end = s.quietHours?.end ?? _defaultQuiet.end;
  }

  void _change(VoidCallback change) => setState(() {
    change();
    _message = null;
    _error = null;
  });

  Future<void> _pick(String current, ValueChanged<String> picked) async {
    final parts = current.split(':');
    final time = await showTimePicker(
      context: context,
      initialTime: TimeOfDay(hour: int.parse(parts[0]), minute: int.parse(parts[1])),
      initialEntryMode: TimePickerEntryMode.input,
      builder: (context, child) =>
          MediaQuery(data: MediaQuery.of(context).copyWith(alwaysUse24HourFormat: true), child: child!),
    );
    if (time == null || !mounted) return;
    _change(() => picked('${_two(time.hour)}:${_two(time.minute)}'));
  }

  Future<void> _save() async {
    final t = context.t;
    setState(() {
      _busy = true;
      _message = null;
      _error = null;
    });
    final settings = api.NotificationSettings(
      channels: api.NotificationChannels(push: _push, email: _email, sms: _sms),
      quietHours: _quiet ? api.QuietHours(start: _start, end: _end) : null,
      timeZone: widget.deviceZone() ?? data?.timeZone,
    );
    try {
      final saved = await apiCall(() => widget.booking.notifications.setNotificationSettings(settings));
      if (!mounted) return;
      setState(() {
        data = saved ?? settings;
        _message = t.remindersSaved;
      });
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _error = _failure(t, failure));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  static String _failure(AppLocalizations t, ApiFailure failure) {
    if (failure.unreachable) return t.unreachable;
    if (failure.code == 'phone_required') return t.remindersPhoneRequired;
    if (failure.status == 400) return t.remindersInvalid;
    return t.somethingWrong;
  }

  static String _two(int n) => n.toString().padLeft(2, '0');

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return ui.Page(
      title: t.reminders,
      children: body(
        (_) => [
          Text(t.remindersIntro),
          Card(
            child: Column(
              children: [
                SwitchListTile(value: _push, onChanged: (v) => _change(() => _push = v), title: Text(t.remindersPush)),
                SwitchListTile(
                  value: _email,
                  onChanged: (v) => _change(() => _email = v),
                  title: Text(t.remindersEmail),
                ),
                SwitchListTile(
                  value: _sms,
                  onChanged: (v) => _change(() => _sms = v),
                  title: Text(t.remindersSms),
                  subtitle: Text(t.remindersSmsHint),
                ),
              ],
            ),
          ),
          Card(
            child: Column(
              children: [
                SwitchListTile(
                  value: _quiet,
                  onChanged: (v) => _change(() => _quiet = v),
                  title: Text(t.quietHours),
                  subtitle: Text(t.quietHoursHint),
                ),
                if (_quiet) ...[
                  ListTile(
                    title: Text(t.quietHoursFrom),
                    trailing: Text(_start),
                    onTap: () => _pick(_start, (v) => _start = v),
                  ),
                  ListTile(
                    title: Text(t.quietHoursUntil),
                    trailing: Text(_end),
                    onTap: () => _pick(_end, (v) => _end = v),
                  ),
                ],
              ],
            ),
          ),
          if (_message != null) ui.Message(_message!),
          if (_error != null) ui.Message(_error!, error: true),
          ui.SubmitButton(t.save, busy: _busy, onPressed: _save),
        ],
      ),
    );
  }
}
