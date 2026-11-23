import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' as api;

import 'booking.dart';
import 'language_picker.dart';
import 'messages.dart';
import 'questionnaire_language.dart';
import 'service_price.dart';
import 'ui.dart' as ui;

/// A listed clinic's profile for patients abroad (docs/clinic-cms, package X2), above its practitioners: its
/// description in the app's language where the clinic translated it (else as written), whether it welcomes patients
/// from abroad, the languages its staff speak, and its services with their prices (in person and video). Nothing is
/// charged in the app. Shows nothing when the profile cannot be read; the practitioners stay.
class ClinicProfileCard extends StatefulWidget {
  const ClinicProfileCard({super.key, required this.booking, required this.clinicId});

  final Booking booking;
  final String clinicId;

  @override
  State<ClinicProfileCard> createState() => _ClinicProfileCardState();
}

class _ClinicProfileCardState extends State<ClinicProfileCard> {
  String? _language;
  api.ClinicProfile? _profile;
  List<api.OfferedService> _services = const [];

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final language = appLanguage(context);
    if (language == _language) return;
    _language = language;
    _load(language).catchError((Object _) {}); // the practitioners go on without it
  }

  /// Reads the profile, then the services named in [language]; nothing shows when the profile cannot be read.
  Future<void> _load(String language) async {
    final profile = await widget.booking.clinicProfile(widget.clinicId);
    if (!mounted || _language != language) return;
    setState(() => _profile = profile);
    final services = await _servicesIn(language);
    if (mounted && _language == language) setState(() => _services = services);
  }

  /// The services for either visit mode, each once.
  Future<List<api.OfferedService>> _servicesIn(String language) async {
    final all = [
      for (final mode in [api.VisitMode.inPerson, api.VisitMode.video])
        ...await widget.booking.offeredServices(widget.clinicId, mode, language),
    ];
    final seen = <String>{};
    return [
      for (final s in all)
        if (seen.add(s.id)) s,
    ];
  }

  @override
  Widget build(BuildContext context) {
    final p = _profile;
    if (p == null) return const SizedBox.shrink();
    final t = context.t;
    final description = descriptionIn(p, _language ?? '');
    final spoken = [
      for (final tag in languageNames.keys)
        if (p.languages.any((l) => l.toJson() == tag)) languageNames[tag]!,
    ];
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          spacing: 8,
          children: [
            if (description.isNotEmpty) Text(description),
            if (p.welcomesInternational) Text(t.welcomesInternational, style: Theme.of(context).textTheme.titleSmall),
            if (spoken.isNotEmpty) Text(t.languagesSpoken(spoken.join(', '))),
            if (_services.isNotEmpty) ui.Heading(t.clinicServices),
            for (final s in _services)
              Row(
                children: [
                  Expanded(child: Text(s.name)),
                  Text(priceText(s.priceMinor, s.currency, t.localeName)),
                ],
              ),
          ],
        ),
      ),
    );
  }
}

/// The profile's description in [language] where the clinic translated it, else as written.
String descriptionIn(api.ClinicProfile p, String language) {
  for (final x in p.translations) {
    if (x.language.toJson() == language) return x.description;
  }
  return p.description;
}
