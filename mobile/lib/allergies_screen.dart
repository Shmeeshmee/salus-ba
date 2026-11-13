import 'package:flutter/material.dart';
import 'package:salus_api/api.dart'
    show AllergyCategory, AllergyClinicalStatus, AllergyIntoleranceInput, AllergySeverity;

import 'booking.dart';
import 'health_profile.dart';
import 'l10n/app_localizations.dart';
import 'messages.dart';
import 'profile_entries.dart';
import 'questionnaire_language.dart' show appLanguage;
import 'ui.dart' as ui;

/// The allergies in the patient's profile and those clinics recorded in their visits (SAL-41).
class AllergiesScreen extends StatefulWidget {
  const AllergiesScreen({super.key, required this.booking, required this.patientId});

  final Booking booking;
  final String patientId;

  @override
  State<AllergiesScreen> createState() => _AllergiesScreenState();
}

class _AllergiesScreenState extends EntriesState<AllergiesScreen, Allergy> {
  HealthProfile get _health => widget.booking.health;

  @override
  Future<List<Allergy>> load() => widget.booking.fhir.allergies(widget.patientId);

  Future<void> _open([Allergy? allergy]) =>
      openForm(() => AllergyForm(health: widget.booking.health, allergy: allergy));

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return ui.Page(
      title: t.allergies,
      children: [
        Text(t.allergiesIntro),
        ...outcome,
        ...body(
          (all) => [
            if (all.isEmpty) Text(t.noAllergies),
            for (final a in all)
              entryCard(
                title: a.text,
                lines: _lines(t, a),
                confirmed: a.confirmed,
                fromClinic: a.fromClinic,
                onChange: () => _open(a),
                onRemove: () => _health.removeAllergy(a.id, a.version),
                onAddToProfile: () => _health.allergyToProfile(a.id, a.version),
              ),
            ui.SubmitButton(t.addAllergy, busy: false, onPressed: _open),
          ],
        ),
      ],
    );
  }

  static List<String> _lines(AppLocalizations t, Allergy a) {
    final category = a.category;
    final reaction = a.reaction;
    final severity = a.severity;
    return [
      if (category != null) categoryName(t, category),
      if (reaction != null)
        severity == null ? t.reactionLine(reaction) : t.reactionWithSeverity(reaction, severityName(t, severity)),
      clinicalStatusName(t, a.clinicalStatus),
    ];
  }
}

String categoryName(AppLocalizations t, AllergyCategory c) => switch (c) {
  AllergyCategory.food => t.categoryFood,
  AllergyCategory.medication => t.categoryMedication,
  AllergyCategory.environment => t.categoryEnvironment,
  AllergyCategory.biologic => t.categoryBiologic,
};

String severityName(AppLocalizations t, AllergySeverity s) => switch (s) {
  AllergySeverity.mild => t.severityMild,
  AllergySeverity.moderate => t.severityModerate,
  AllergySeverity.severe => t.severitySevere,
};

String clinicalStatusName(AppLocalizations t, AllergyClinicalStatus s) => switch (s) {
  AllergyClinicalStatus.active => t.allergyActive,
  AllergyClinicalStatus.inactive => t.allergyInactive,
  AllergyClinicalStatus.resolved => t.allergyResolved,
};

/// Adds an allergy ([allergy] null) or changes the patient's own. A changed entry keeps its term when its text is
/// unchanged: the resource does not carry the term's id, so none is sent unless a term is picked again.
class AllergyForm extends StatefulWidget {
  const AllergyForm({super.key, required this.health, this.allergy});

  final HealthProfile health;
  final Allergy? allergy;

  @override
  State<AllergyForm> createState() => _AllergyFormState();
}

class _AllergyFormState extends EntryFormState<AllergyForm> {
  late final _text = TextEditingController(text: widget.allergy?.text);
  late final _reaction = TextEditingController(text: widget.allergy?.reaction);
  late AllergyCategory? _category = widget.allergy?.category;
  late AllergySeverity? _severity = widget.allergy?.severity;
  late AllergyClinicalStatus _status = widget.allergy?.clinicalStatus ?? AllergyClinicalStatus.active;
  String? _termId;

  @override
  void initState() {
    super.initState();
    _reaction.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _text.dispose();
    _reaction.dispose();
    super.dispose();
  }

  Future<void> _save() {
    final reaction = _reaction.text.trim();
    final input = AllergyIntoleranceInput(
      termId: _termId,
      text: _text.text.trim(),
      category: _category,
      reaction: reaction.isEmpty ? null : reaction,
      severity: reaction.isEmpty ? null : _severity,
      clinicalStatus: _status,
    );
    final a = widget.allergy;
    return save(
      () => a == null ? widget.health.addAllergy(input) : widget.health.updateAllergy(a.id, a.version, input),
    );
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    final described = _reaction.text.trim().isNotEmpty;
    return ui.Page(
      title: widget.allergy == null ? t.addAllergy : t.changeAllergy,
      children: [
        if (widget.allergy?.confirmed ?? false) Text(t.confirmedChangeNote),
        Form(
          key: form,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              TermField(
                controller: _text,
                label: t.allergySubstance,
                search: (q) => widget.health.search(q, appLanguage(context)),
                onPicked: (id) => _termId = id,
              ),
              const SizedBox(height: 16),
              DropdownButtonFormField<AllergyCategory?>(
                isExpanded: true,
                initialValue: _category,
                decoration: InputDecoration(labelText: t.allergyCategory),
                items: [
                  DropdownMenuItem(child: Text(t.notSpecified)),
                  for (final c in AllergyCategory.values) DropdownMenuItem(value: c, child: Text(categoryName(t, c))),
                ],
                onChanged: (c) => setState(() => _category = c),
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _reaction,
                decoration: InputDecoration(labelText: t.reactionOptional),
                validator: (v) => (v ?? '').trim().runes.length > 200 ? t.upToCharacters(200) : null,
              ),
              const SizedBox(height: 16),
              DropdownButtonFormField<AllergySeverity?>(
                isExpanded: true,
                initialValue: _severity,
                decoration: InputDecoration(labelText: t.severity),
                items: [
                  DropdownMenuItem(child: Text(t.notSpecified)),
                  for (final s in AllergySeverity.values) DropdownMenuItem(value: s, child: Text(severityName(t, s))),
                ],
                onChanged: described ? (s) => setState(() => _severity = s) : null,
              ),
              const SizedBox(height: 16),
              DropdownButtonFormField<AllergyClinicalStatus>(
                isExpanded: true,
                initialValue: _status,
                decoration: InputDecoration(labelText: t.entryStatus),
                items: [
                  for (final s in AllergyClinicalStatus.values)
                    DropdownMenuItem(value: s, child: Text(clinicalStatusName(t, s))),
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
