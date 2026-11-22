import 'package:flutter/material.dart';

import 'fhir.dart';
import 'l10n/app_localizations.dart';
import 'messages.dart';

/// How the patient attends, in the patient's words: at the clinic or by video (ADR 0016).
String visitModeName(AppLocalizations t, {required bool inPerson}) => inPerson ? t.visitModeInPerson : t.visitModeVideo;

/// Whether [slots] hold both visits at the clinic and by video: only then is there a mode to pick.
bool hasBothModes(List<Slot> slots) => slots.any((s) => s.inPerson) && slots.any((s) => !s.inPerson);

/// The slots of one mode, in their order.
List<Slot> slotsOfMode(List<Slot> slots, {required bool inPerson}) => [
  for (final s in slots)
    if (s.inPerson == inPerson) s,
];

/// The choice between a visit by video and one at the clinic, made before the time is picked.
class VisitModePicker extends StatelessWidget {
  const VisitModePicker({super.key, required this.inPerson, required this.onChanged});

  final bool inPerson;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(t.visitModeQuestion),
        const SizedBox(height: 8),
        SegmentedButton<bool>(
          showSelectedIcon: false,
          segments: [
            ButtonSegment(value: false, label: Text(t.visitModeVideo)),
            ButtonSegment(value: true, label: Text(t.visitModeInPerson)),
          ],
          selected: {inPerson},
          onSelectionChanged: (s) => onChanged(s.first),
        ),
      ],
    );
  }
}
