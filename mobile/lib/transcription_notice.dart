import 'package:flutter/material.dart';

import 'messages.dart';
import 'ui.dart' as ui;

/// The notice before the call while the patient's transcription consent with the clinic is not active (SAL-12,
/// docs/consultation/COPY.md M37 to M43): what is transcribed and translated, the AI assistance for the
/// practitioner, where the transcript is kept, what declining does and how to withdraw. [onAgree] gives the
/// consent; [onDecline] goes on to the call without the agent. The text waits for the owner's approval and the
/// lawyer's answer to Q16 (docs/legal/BRIEF.md).
class TranscriptionNotice extends StatelessWidget {
  const TranscriptionNotice({
    super.key,
    required this.clinicName,
    required this.busy,
    this.error,
    required this.onAgree,
    required this.onDecline,
  });

  final String clinicName;
  final bool busy;
  final String? error;
  final VoidCallback onAgree;
  final VoidCallback onDecline;

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return ui.Page(
      title: t.transcriptionTitle,
      children: [
        Text(t.transcriptionIntro(clinicName)),
        Text(t.transcriptionAssistance),
        Text(t.transcriptionKept(clinicName)),
        Text(t.transcriptionDecline),
        Text(t.transcriptionWithdraw),
        if (error != null) ui.Message(error!, error: true),
        ui.SubmitButton(t.transcriptionAgree, busy: busy, onPressed: onAgree),
        TextButton(onPressed: busy ? null : onDecline, child: Text(t.transcriptionWithout)),
      ],
    );
  }
}
