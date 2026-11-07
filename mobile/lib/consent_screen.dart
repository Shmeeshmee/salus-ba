import 'package:flutter/material.dart';

import 'messages.dart';
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
    final failed = context.t.privacyNoticeFailed;
    final opened = await notice.open(notice.url);
    if (!mounted) return;
    setState(() => _error = opened ? null : failed);
  }

  @override
  Widget build(BuildContext context) {
    final clinicName = widget.clinicName;
    final t = context.t;
    return ui.Page(
      title: t.consentTitle(clinicName),
      children: [
        Text(t.consentIntro(clinicName)),
        ui.Heading(t.whatIsShared),
        Text(t.whatIsSharedBody),
        ui.Heading(t.whoSeesIt),
        Text(t.whoSeesItBody(clinicName)),
        ui.Heading(t.whatFor),
        Text(t.whatForBody),
        ui.Heading(t.whoIsResponsible),
        Text(t.whoIsResponsibleBody(clinicName)),
        Align(
          alignment: AlignmentDirectional.centerStart,
          child: TextButton(onPressed: _openNotice, child: Text(t.readPrivacyNotice)),
        ),
        if (_error != null) ui.Message(_error!, error: true),
        ui.Heading(t.howToRevoke),
        Text(t.howToRevokeBody),
        FilledButton(onPressed: () => Navigator.of(context).pop(true), child: Text(t.agreeAndBook)),
        TextButton(onPressed: () => Navigator.of(context).pop(false), child: Text(t.notNow)),
      ],
    );
  }
}
