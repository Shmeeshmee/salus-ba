import 'package:flutter/material.dart';

import 'package:salus_api/api.dart' show ChatAttachment;
import 'messages.dart';

/// A file in the composer: prepared and uploading, then uploaded (its [attachment]) until the message is sent.
class PendingFile {
  PendingFile(this.name);

  String name;
  ChatAttachment? attachment;
}

/// The chat view's composer: the files to send, "Add a file", the text and "Send" ([onSend] null while it may not
/// be pressed).
class ChatComposer extends StatelessWidget {
  const ChatComposer({
    super.key,
    required this.text,
    required this.pending,
    required this.onAttach,
    required this.onRemove,
    required this.onSend,
  });

  final TextEditingController text;
  final List<PendingFile> pending;
  final VoidCallback onAttach;
  final void Function(PendingFile p) onRemove;
  final VoidCallback? onSend;

  @override
  Widget build(BuildContext context) {
    final words = context.t;
    return Material(
      color: Theme.of(context).colorScheme.surface,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(4, 8, 4, 8),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (pending.isNotEmpty)
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 8),
                child: Wrap(
                  spacing: 8,
                  children: [
                    for (final p in pending)
                      InputChip(
                        avatar: Icon(p.attachment == null ? Icons.upload_outlined : Icons.attach_file, size: 18),
                        label: Text(p.name),
                        deleteButtonTooltipMessage: words.chatFileRemove,
                        onDeleted: () => onRemove(p),
                      ),
                  ],
                ),
              ),
            Row(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                IconButton(tooltip: words.chatAttach, icon: const Icon(Icons.attach_file), onPressed: onAttach),
                Expanded(
                  child: TextField(
                    controller: text,
                    minLines: 1,
                    maxLines: 5,
                    maxLength: 4000,
                    textCapitalization: TextCapitalization.sentences,
                    buildCounter: (_, {required currentLength, required isFocused, maxLength}) => null,
                    decoration: InputDecoration(hintText: words.chatMessageHint, isDense: true),
                  ),
                ),
                IconButton(tooltip: words.chatSend, icon: const Icon(Icons.send), onPressed: onSend),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
