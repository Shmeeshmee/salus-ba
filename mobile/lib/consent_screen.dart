import 'package:flutter/material.dart';

import 'ui.dart' as ui;

/// The consent asked for on the first booking with a clinic (docs/booking/SPEC.md "Consent shape"): what is
/// shared, with whom, for what and how to revoke it. Pops true when the patient agrees.
class ConsentScreen extends StatelessWidget {
  const ConsentScreen({super.key, required this.clinicName});

  final String clinicName;

  @override
  Widget build(BuildContext context) {
    return ui.Page(
      title: 'Share your details with $clinicName',
      children: [
        Text('Before your first booking with $clinicName, we need your consent to share your details with it.'),
        const ui.Heading('What is shared'),
        const Text(
          'Your profile (your name, date of birth and, if you gave one, your phone number) and your appointments '
          'with this clinic.',
        ),
        const ui.Heading('Who sees it'),
        Text(
          '$clinicName. The practitioner you book with sees your profile; the clinic\'s administrators see your '
          'name and appointment times.',
        ),
        const ui.Heading('What for'),
        const Text('Only for your care at this clinic: to prepare for and hold your appointments.'),
        const ui.Heading('How to revoke'),
        const Text(
          'You can revoke this consent at any time under My consents. The clinic then no longer sees your '
          'profile, and your future appointments there are cancelled. Appointments that have already taken place '
          'stay with the clinic as part of its care records.',
        ),
        FilledButton(onPressed: () => Navigator.of(context).pop(true), child: const Text('Agree and book')),
        TextButton(onPressed: () => Navigator.of(context).pop(false), child: const Text('Not now')),
      ],
    );
  }
}
