import 'package:flutter/material.dart';
import 'package:intl/intl.dart' show NumberFormat;
import 'package:salus_api/api.dart' show Dosage, DoseUnit, MedicationStatementInput, MedicationStatus, PeriodUnit;

import 'booking.dart';
import 'health_profile.dart';
import 'l10n/app_localizations.dart';
import 'messages.dart';
import 'profile_entries.dart';
import 'questionnaire_language.dart' show appLanguage;
import 'ui.dart' as ui;

/// The medications in the patient's profile and those clinics recorded in their visits (SAL-41).
class MedicationsScreen extends StatefulWidget {
  const MedicationsScreen({super.key, required this.booking, required this.patientId});

  final Booking booking;
  final String patientId;

  @override
  State<MedicationsScreen> createState() => _MedicationsScreenState();
}

class _MedicationsScreenState extends EntriesState<MedicationsScreen, Medication> {
  HealthProfile get _health => widget.booking.health;

  @override
  Future<List<Medication>> load() => widget.booking.fhir.medications(widget.patientId);

  Future<void> _open([Medication? medication]) =>
      openForm(() => MedicationForm(health: widget.booking.health, medication: medication));

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return ui.Page(
      title: t.currentMedications,
      children: [
        Text(t.medicationsIntro),
        ...outcome,
        ...body(
          (all) => [
            if (all.isEmpty) Text(t.noMedications),
            for (final m in all)
              entryCard(
                title: m.text,
                lines: _lines(t, m),
                confirmed: m.confirmed,
                fromClinic: m.fromClinic,
                onChange: () => _open(m),
                onRemove: () => _health.removeMedication(m.id, m.version),
                onAddToProfile: () => _health.medicationToProfile(m.id, m.version),
              ),
            ui.SubmitButton(t.addMedication, busy: false, onPressed: _open),
          ],
        ),
      ],
    );
  }

  static List<String> _lines(AppLocalizations t, Medication m) {
    final d = m.dosage;
    final (value, unit) = (d.doseValue, d.doseUnit);
    final (frequency, period, periodUnit) = (d.frequency, d.period, d.periodUnit);
    return [
      if (value != null && unit != null) t.doseLine(doseText(t, value, unit)),
      if (frequency != null && period != null && periodUnit != null) howOftenText(t, frequency, period, periodUnit),
      if (d.asNeeded ?? false) t.asNeeded,
      statusName(t, m.status),
    ];
  }
}

String _number(AppLocalizations t, num n) => NumberFormat.decimalPattern(t.localeName).format(n);

/// A dose in the app's language, as "500 mg" or "2 tablets".
String doseText(AppLocalizations t, num value, DoseUnit unit) {
  final v = _number(t, value);
  return switch (unit) {
    DoseUnit.mg => t.doseMg(v),
    DoseUnit.g => t.doseG(v),
    DoseUnit.ug => t.doseUg(v),
    DoseUnit.mL => t.doseMl(v),
    DoseUnit.leftSquareBracketIURightSquareBracket => t.doseIu(v),
    DoseUnit.leftCurlyBracketTabletRightCurlyBracket => t.doseTablet(value, v),
    DoseUnit.leftCurlyBracketCapsuleRightCurlyBracket => t.doseCapsule(value, v),
    DoseUnit.leftCurlyBracketDropRightCurlyBracket => t.doseDrop(value, v),
    DoseUnit.leftCurlyBracketPuffRightCurlyBracket => t.dosePuff(value, v),
    DoseUnit.leftCurlyBracketPatchRightCurlyBracket => t.dosePatch(value, v),
  };
}

/// How often, as "2× a day" or "1× every 2 weeks".
String howOftenText(AppLocalizations t, int frequency, num period, PeriodUnit unit) {
  if (period == 1) {
    return switch (unit) {
      PeriodUnit.h => t.howOftenPerHour(frequency),
      PeriodUnit.d => t.howOftenPerDay(frequency),
      PeriodUnit.wk => t.howOftenPerWeek(frequency),
      PeriodUnit.mo => t.howOftenPerMonth(frequency),
    };
  }
  final every = _number(t, period);
  return switch (unit) {
    PeriodUnit.h => t.howOftenEveryHours(period, frequency, every),
    PeriodUnit.d => t.howOftenEveryDays(period, frequency, every),
    PeriodUnit.wk => t.howOftenEveryWeeks(period, frequency, every),
    PeriodUnit.mo => t.howOftenEveryMonths(period, frequency, every),
  };
}

/// A unit in the form's choice, by its readable name; the UCUM code is what is sent.
String unitName(AppLocalizations t, DoseUnit unit) => switch (unit) {
  DoseUnit.mg => t.unitMg,
  DoseUnit.g => t.unitG,
  DoseUnit.ug => t.unitUg,
  DoseUnit.mL => t.unitMl,
  DoseUnit.leftSquareBracketIURightSquareBracket => t.unitIu,
  DoseUnit.leftCurlyBracketTabletRightCurlyBracket => t.unitTablet,
  DoseUnit.leftCurlyBracketCapsuleRightCurlyBracket => t.unitCapsule,
  DoseUnit.leftCurlyBracketDropRightCurlyBracket => t.unitDrop,
  DoseUnit.leftCurlyBracketPuffRightCurlyBracket => t.unitPuff,
  DoseUnit.leftCurlyBracketPatchRightCurlyBracket => t.unitPatch,
};

String periodName(AppLocalizations t, PeriodUnit unit) => switch (unit) {
  PeriodUnit.h => t.periodHours,
  PeriodUnit.d => t.periodDays,
  PeriodUnit.wk => t.periodWeeks,
  PeriodUnit.mo => t.periodMonths,
};

String statusName(AppLocalizations t, MedicationStatus s) => switch (s) {
  MedicationStatus.active => t.medicationActive,
  MedicationStatus.stopped => t.medicationStopped,
};

/// A number as the patient typed it, with a comma or a point; null when it is none.
num? parseNumber(String text) => num.tryParse(text.trim().replaceAll(',', '.'));

/// The form's text of a number read from the API: no ".0" on a whole number.
String _shown(num? n) => n == null ? '' : (n == n.truncate() ? n.truncate().toString() : n.toString());

/// Adds a medication ([medication] null) or changes the patient's own; as [AllergyForm] for the term.
class MedicationForm extends StatefulWidget {
  const MedicationForm({super.key, required this.health, this.medication});

  final HealthProfile health;
  final Medication? medication;

  @override
  State<MedicationForm> createState() => _MedicationFormState();
}

class _MedicationFormState extends EntryFormState<MedicationForm> {
  late final Dosage? _read = widget.medication?.dosage;
  late final _text = TextEditingController(text: widget.medication?.text);
  late final _amount = TextEditingController(text: _shown(_read?.doseValue));
  late final _times = TextEditingController(text: _shown(_read?.frequency));
  late final _every = TextEditingController(text: _shown(_read?.period));
  late DoseUnit? _unit = _read?.doseUnit;
  late PeriodUnit? _periodUnit = _read?.periodUnit;
  late bool _asNeeded = _read?.asNeeded ?? false;
  late MedicationStatus _status = widget.medication?.status ?? MedicationStatus.active;
  String? _termId;

  @override
  void dispose() {
    _text.dispose();
    _amount.dispose();
    _times.dispose();
    _every.dispose();
    super.dispose();
  }

  String? _checkAmount(AppLocalizations t, String? v) {
    final text = (v ?? '').trim();
    if (text.isEmpty) return _unit == null ? null : t.doseBoth;
    final n = parseNumber(text);
    if (n == null) return t.enterNumber;
    if (n <= 0) return t.numberAboveZero;
    if (n >= 100000) return t.numberTooLarge;
    return _unit == null ? t.doseBoth : null;
  }

  String? _checkTimes(AppLocalizations t, String? v) {
    final text = (v ?? '').trim();
    final filled = [text.isNotEmpty, _every.text.trim().isNotEmpty, _periodUnit != null];
    if (filled.contains(true) && filled.contains(false)) return t.howOftenAll;
    if (text.isEmpty) return null;
    final n = int.tryParse(text);
    if (n == null) return t.enterWholeNumber;
    if (n < 1) return t.numberAboveZero;
    return n > 100 ? t.numberTooLarge : null;
  }

  String? _checkEvery(AppLocalizations t, String? v) {
    final text = (v ?? '').trim();
    if (text.isEmpty) return null;
    final n = parseNumber(text);
    if (n == null) return t.enterNumber;
    if (n <= 0) return t.numberAboveZero;
    return n > 1000 ? t.numberTooLarge : null;
  }

  Future<void> _save() {
    final input = MedicationStatementInput(
      termId: _termId,
      text: _text.text.trim(),
      status: _status,
      dosage: Dosage(
        doseValue: parseNumber(_amount.text),
        doseUnit: _unit,
        frequency: int.tryParse(_times.text.trim()),
        period: parseNumber(_every.text),
        periodUnit: _periodUnit,
        asNeeded: _asNeeded ? true : null,
      ),
    );
    final m = widget.medication;
    return save(
      () => m == null ? widget.health.addMedication(input) : widget.health.updateMedication(m.id, m.version, input),
    );
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    const gap = SizedBox(height: 16);
    const number = TextInputType.numberWithOptions(decimal: true);
    return ui.Page(
      title: widget.medication == null ? t.addMedication : t.changeMedication,
      children: [
        if (widget.medication?.confirmed ?? false) Text(t.confirmedChangeNote),
        Form(
          key: form,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              TermField(
                controller: _text,
                label: t.medicationName,
                search: (q) => widget.health.search(q, appLanguage(context)),
                onPicked: (id) => _termId = id,
              ),
              const SizedBox(height: 24),
              ui.Heading(t.doseHeading),
              gap,
              TextFormField(
                controller: _amount,
                decoration: InputDecoration(labelText: t.doseAmount),
                keyboardType: number,
                validator: (v) => _checkAmount(t, v),
              ),
              gap,
              DropdownButtonFormField<DoseUnit?>(
                isExpanded: true,
                initialValue: _unit,
                decoration: InputDecoration(labelText: t.doseUnit),
                items: [
                  DropdownMenuItem(child: Text(t.notSpecified)),
                  for (final u in DoseUnit.values) DropdownMenuItem(value: u, child: Text(unitName(t, u))),
                ],
                onChanged: (u) => setState(() => _unit = u),
              ),
              const SizedBox(height: 24),
              ui.Heading(t.howOftenHeading),
              gap,
              TextFormField(
                controller: _times,
                decoration: InputDecoration(labelText: t.timesLabel),
                keyboardType: TextInputType.number,
                validator: (v) => _checkTimes(t, v),
              ),
              gap,
              TextFormField(
                controller: _every,
                decoration: InputDecoration(labelText: t.everyLabel),
                keyboardType: number,
                validator: (v) => _checkEvery(t, v),
              ),
              gap,
              DropdownButtonFormField<PeriodUnit?>(
                isExpanded: true,
                initialValue: _periodUnit,
                decoration: InputDecoration(labelText: t.periodUnitLabel),
                items: [
                  DropdownMenuItem(child: Text(t.notSpecified)),
                  for (final u in PeriodUnit.values) DropdownMenuItem(value: u, child: Text(periodName(t, u))),
                ],
                onChanged: (u) => setState(() => _periodUnit = u),
              ),
              gap,
              CheckboxListTile(
                value: _asNeeded,
                title: Text(t.asNeeded),
                contentPadding: EdgeInsets.zero,
                onChanged: (v) => setState(() => _asNeeded = v ?? false),
              ),
              gap,
              DropdownButtonFormField<MedicationStatus>(
                isExpanded: true,
                initialValue: _status,
                decoration: InputDecoration(labelText: t.entryStatus),
                items: [
                  for (final s in MedicationStatus.values) DropdownMenuItem(value: s, child: Text(statusName(t, s))),
                ],
                onChanged: (s) => setState(() => _status = s ?? _status),
              ),
              const SizedBox(height: 24),
              ui.SubmitButton(t.save, busy: busy, onPressed: _save),
            ],
          ),
        ),
        if (error != null) ui.Message(error!, error: true),
      ],
    );
  }
}
