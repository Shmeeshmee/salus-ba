import 'package:flutter/material.dart';
import 'package:salus_api/api.dart';

import 'allergies_screen.dart';
import 'booking.dart';
import 'medications_screen.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;

/// The patient's profile (docs/booking/SPEC.md "Profile"): created on the first save, then updated at the version
/// it was read. A saved profile also opens the patient's allergies and current medications (SAL-41).
class ProfileScreen extends StatefulWidget {
  const ProfileScreen({
    super.key,
    required this.booking,
    required this.now,
    required this.onSaved,
    this.profile,
    this.onLater,
  });

  final Booking booking;
  final DateTime Function() now;

  /// The profile to edit; null to create it.
  final Patient? profile;
  final ValueChanged<Patient> onSaved;

  /// When set, the patient may leave the profile for later.
  final VoidCallback? onLater;

  @override
  State<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends State<ProfileScreen> {
  final _form = GlobalKey<FormState>();
  final _given = TextEditingController();
  final _family = TextEditingController();
  final _birth = TextEditingController();
  final _phone = TextEditingController();
  late int? _version = widget.profile?.versionId;
  String? _error;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    final p = widget.profile;
    if (p != null) _show(p);
  }

  @override
  void dispose() {
    _given.dispose();
    _family.dispose();
    _birth.dispose();
    _phone.dispose();
    super.dispose();
  }

  void _show(Patient p) {
    _given.text = p.givenName;
    _family.text = p.familyName;
    _birth.text = p.birthDate.toIso8601String().substring(0, 10);
    _phone.text = p.phone ?? '';
    _version = p.versionId;
  }

  Future<void> _save() async {
    if (!_form.currentState!.validate()) return;
    final phone = normalisePhone(_phone.text);
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final saved = await widget.booking.saveMe(
        givenName: _given.text.trim(),
        familyName: _family.text.trim(),
        birthDate: parseDate(_birth.text)!,
        phone: phone.isEmpty ? null : phone,
        versionId: _version,
      );
      if (mounted) widget.onSaved(saved);
    } on ApiFailure catch (failure) {
      if (!mounted) return;
      setState(() => _error = bookingMessage(context.t, failure));
      if (failure.code == 'version_conflict') await _reload();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  /// Reads the profile again after a conflict, so the next save is at its current version.
  Future<void> _reload() async {
    try {
      final current = await widget.booking.me();
      if (current != null && mounted) setState(() => _show(current));
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _error = bookingMessage(context.t, failure));
    }
  }

  @override
  Widget build(BuildContext context) {
    final creating = widget.profile == null;
    final t = context.t;
    return ui.Page(
      title: creating ? t.completeProfile : t.myProfile,
      children: [
        if (creating) Text(t.profileIntro),
        Form(
          key: _form,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              TextFormField(
                controller: _given,
                decoration: InputDecoration(labelText: t.givenName),
                autofillHints: const [AutofillHints.givenName],
                textCapitalization: TextCapitalization.words,
                validator: (v) => validateGivenName(t, v),
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _family,
                decoration: InputDecoration(labelText: t.familyName),
                autofillHints: const [AutofillHints.familyName],
                textCapitalization: TextCapitalization.words,
                validator: (v) => validateFamilyName(t, v),
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _birth,
                decoration: InputDecoration(labelText: t.dateOfBirth, helperText: t.dateFormatHint),
                keyboardType: TextInputType.datetime,
                // Digits and dashes read left to right in every language (docs/i18n/SPEC.md §3 "Right to left").
                textDirection: TextDirection.ltr,
                validator: (v) => validateBirthDate(t, v, widget.now()),
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _phone,
                decoration: InputDecoration(labelText: t.phoneOptional, helperText: t.phoneHint(phoneExample(t))),
                keyboardType: TextInputType.phone,
                textDirection: TextDirection.ltr,
                autofillHints: const [AutofillHints.telephoneNumber],
                validator: (v) => validatePhone(t, v),
              ),
              const SizedBox(height: 24),
              ui.SubmitButton(t.save, busy: _busy, onPressed: _save),
              if (widget.onLater != null) ...[
                const SizedBox(height: 8),
                TextButton(onPressed: _busy ? null : widget.onLater, child: Text(t.notNow)),
              ],
            ],
          ),
        ),
        if (_error != null) ui.Message(_error!, error: true),
        if (widget.profile case final profile?) ...[
          OutlinedButton(
            onPressed: () => _open(AllergiesScreen(booking: widget.booking, patientId: profile.id)),
            child: Text(t.allergies),
          ),
          OutlinedButton(
            onPressed: () => _open(MedicationsScreen(booking: widget.booking, patientId: profile.id)),
            child: Text(t.currentMedications),
          ),
        ],
      ],
    );
  }

  void _open(Widget screen) => Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => screen));
}
