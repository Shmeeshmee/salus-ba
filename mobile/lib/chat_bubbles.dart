import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' show ChatMessage, SystemMessage;

import 'chat.dart';
import 'l10n/app_localizations.dart';
import 'messages.dart';
import 'zones.dart';

// One message of the chat view (SAL-31): a participant's text and files in a bubble, the patient's own on the
// reading side's end, or a system message as a line from the catalog. Nothing here reads a system message's
// arguments as text: they are opaque ids and codes.

/// A system message in the app's language, from its code (docs/plans/chat.md "Contract of wave 1").
String systemText(AppLocalizations t, SystemMessage s) => switch (s.code) {
  'participant_invited' => t.chatSystemInvited(s.args['reason'] ?? ''),
  'participant_added' => t.chatSystemAdded(s.args['reason'] ?? ''),
  'participant_removed' => t.chatSystemRemoved,
  'consent_granted' => t.chatSystemGranted,
  'consent_declined' => t.chatSystemDeclined,
  _ => t.chatSystemOther,
};

class ChatBubble extends StatelessWidget {
  const ChatBubble({
    super.key,
    required this.message,
    required this.mine,
    required this.onOpen,
    required this.thumbnail,
  });

  final ChatMessage message;

  /// The patient's own message.
  final bool mine;
  final void Function(ChatAttachment a) onOpen;
  final Future<Uint8List> Function(ChatAttachment a) thumbnail;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final system = message.system;
    if (system != null) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 24),
        child: Text(
          systemText(context.t, system),
          textAlign: TextAlign.center,
          style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant),
        ),
      );
    }
    final colors = theme.colorScheme;
    final body = message.body;
    return Align(
      alignment: mine ? AlignmentDirectional.centerEnd : AlignmentDirectional.centerStart,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 300),
        child: Container(
          margin: const EdgeInsets.symmetric(vertical: 4),
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 6),
          decoration: BoxDecoration(
            color: mine ? colors.primaryContainer : colors.surface,
            border: Border.all(color: colors.outlineVariant),
            borderRadius: BorderRadius.circular(12),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (body != null && body.isNotEmpty) Text(body),
              for (final a in message.attachments) _Attachment(a, onOpen: onOpen, thumbnail: thumbnail),
              const SizedBox(height: 2),
              Text(Zone.clock(message.sentAt.toLocal()), style: theme.textTheme.labelSmall),
            ],
          ),
        ),
      ),
    );
  }
}

/// A file of a message: checked, refused, or ready to open; a clean image shows its thumbnail.
class _Attachment extends StatelessWidget {
  const _Attachment(this.a, {required this.onOpen, required this.thumbnail});

  final ChatAttachment a;
  final void Function(ChatAttachment a) onOpen;
  final Future<Uint8List> Function(ChatAttachment a) thumbnail;

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    final colors = Theme.of(context).colorScheme;
    Widget line(IconData icon, List<Widget> texts) => Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 18),
          const SizedBox(width: 8),
          Flexible(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: texts)),
        ],
      ),
    );
    return switch (a.status) {
      AttachmentStatus.scanning => line(Icons.hourglass_empty, [Text(a.filename), Text(t.checkingFile)]),
      AttachmentStatus.rejected => line(Icons.block, [
        Text(a.filename),
        Text(t.chatFileRejected, style: TextStyle(color: colors.error)),
      ]),
      AttachmentStatus.clean when a.image && a.thumbnail => Padding(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: InkWell(
          onTap: () => onOpen(a),
          child: FutureBuilder<Uint8List>(
            future: thumbnail(a),
            builder: (context, snap) => ClipRRect(
              borderRadius: BorderRadius.circular(8),
              child: snap.hasData
                  ? Image.memory(snap.data!, height: 160, fit: BoxFit.cover, semanticLabel: a.filename)
                  : SizedBox(height: 160, width: 160, child: Center(child: Text(a.filename))),
            ),
          ),
        ),
      ),
      AttachmentStatus.clean => InkWell(
        onTap: () => onOpen(a),
        child: line(Icons.description_outlined, [Text(a.filename)]),
      ),
    };
  }
}
