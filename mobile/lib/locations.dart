import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' as api;

import 'fhir.dart';
import 'messages.dart';

// Where an in-person visit is (docs/clinic-cms/WAVE6.md U1, U2): one of the clinic's locations, picked before the
// time and shown on the visit with its address.

/// The location's street and postal code as one line; null when it has neither.
String? locationAddress(api.Location location) {
  final parts = [location.addressLine, location.postalCode].whereType<String>().where((s) => s.isNotEmpty);
  return parts.isEmpty ? null : parts.join(', ');
}

/// The locations of [all] (the clinic's, in its order) that some of [slots] are at.
List<api.Location> locationsOf(List<Slot> slots, List<api.Location> all) {
  final at = {for (final s in slots) s.locationId};
  return [
    for (final l in all)
      if (at.contains(l.id)) l,
  ];
}

/// A location's name and, below it, its address.
class LocationLines extends StatelessWidget {
  const LocationLines(this.location, {super.key});

  final api.Location location;

  @override
  Widget build(BuildContext context) {
    final address = locationAddress(location);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [Text(location.name), if (address != null) Text(address)],
    );
  }
}

/// The choice between the clinic's locations of the day, made before the time is picked; the picked one's address
/// below it.
class LocationPicker extends StatelessWidget {
  const LocationPicker({super.key, required this.locations, required this.selected, required this.onChanged});

  final List<api.Location> locations;
  final api.Location selected;
  final ValueChanged<api.Location> onChanged;

  @override
  Widget build(BuildContext context) {
    final address = locationAddress(selected);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(context.t.locationQuestion),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (final l in locations)
              ChoiceChip(label: Text(l.name), selected: l.id == selected.id, onSelected: (_) => onChanged(l)),
          ],
        ),
        if (address != null) ...[const SizedBox(height: 8), Text(address)],
      ],
    );
  }
}
