import 'package:flutter/material.dart';

import 'privacy_notice.dart';
import 'ui.dart' as ui;

/// The consent asked for on the first booking with a clinic (docs/booking/SPEC.md "Consent shape"): what is
/// shared, with whom, for what, who is responsible and how to revoke it, with a link to the privacy notice.
/// Pops true when the patient agrees. The text needs legal review before production (SPEC.md).
class ConsentScreen extends StatefulWidget {
  const ConsentScreen({super.key, required this.clinicName});

  final String clinicName;

  @override
  State<ConsentScreen> createState() => _ConsentScreenState();
}

class _ConsentScreenState extends State<ConsentScreen> {
  String? _error;

  Future<void> _openNotice() async {
    final notice = PrivacyNotice.of(context);
    final opened = await notice.open(notice.url);
    if (!mounted) return;
    setState(() => _error = opened ? null : 'The privacy notice could not be opened.');
  }

  @override
  Widget build(BuildContext context) {
    final clinicName = widget.clinicName;
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
        const ui.Heading('Who is responsible'),
        Text("$clinicName is responsible for your data. Salus processes it on the clinic's behalf."),
        Align(
          alignment: AlignmentDirectional.centerStart,
          child: TextButton(onPressed: _openNotice, child: const Text('Read the privacy notice')),
        ),
        if (_error != null) ui.Message(_error!, error: true),
        const ui.Heading('How to revoke'),
        const Text(
          'You can revoke this consent at any time under My consents. The clinic then no longer sees your profile. '
          'It keeps your name and the details of appointments that have already taken place, as part of its care '
          'records. Your future appointments there are cancelled.',
        ),
        FilledButton(onPressed: () => Navigator.of(context).pop(true), child: const Text('Agree and book')),
        TextButton(onPressed: () => Navigator.of(context).pop(false), child: const Text('Not now')),
      ],
    );
  }
}
