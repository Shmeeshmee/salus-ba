import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' show Conversation;

import 'booking.dart' show Places;
import 'call.dart' show Media;
import 'chat.dart';
import 'chat_screen.dart';
import 'files.dart';
import 'messages.dart';
import 'ui.dart' as ui;

// The patient's conversations (SAL-31): the doctor and the clinic of each, newest activity first, with the unread
// count. A doctor-only side thread is never the patient's; the API does not list one, and neither does this screen.

/// Names the doctors of [clinicIds]' clinics (Booking.places).
typedef Names = Future<Places> Function(Iterable<String> clinicIds);

class ChatListScreen extends StatefulWidget {
  const ChatListScreen({
    super.key,
    required this.chats,
    required this.places,
    required this.files,
    required this.media,
    required this.now,
  });

  final Chats chats;
  final Names places;
  final Files files;
  final Media media;
  final DateTime Function() now;

  @override
  State<ChatListScreen> createState() => _ChatListScreenState();
}

class _ChatListScreenState extends State<ChatListScreen> {
  List<Conversation>? _conversations;
  Places _names = const Places({}, {});
  String? _next;
  var _loadingMore = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _error = null);
    try {
      final page = await widget.chats.list();
      final mine = [for (final c in page.conversations) if (!c.staffOnly) c];
      final names = await _resolve(mine);
      if (!mounted) return;
      setState(() {
        _conversations = mine;
        _next = page.next;
        _names = names;
      });
    } on ChatException catch (e) {
      if (mounted) setState(() => _error = e.code);
    }
  }

  Future<Places> _resolve(List<Conversation> list) async {
    try {
      return await widget.places({for (final c in list) c.organizationId});
    } on Exception {
      return _names; // Without the names the list still works: "Your doctor".
    }
  }

  Future<void> _more() async {
    if (_loadingMore || _next == null) return;
    _loadingMore = true;
    try {
      final page = await widget.chats.list(before: _next);
      final mine = [for (final c in page.conversations) if (!c.staffOnly) c];
      final names = await _resolve([...?_conversations, ...mine]);
      if (mounted) {
        setState(() {
          _conversations = [...?_conversations, ...mine];
          _next = page.next;
          _names = names;
        });
      }
    } on ChatException {
      // The end of the list stays where it is; scrolling there again tries again.
    } finally {
      _loadingMore = false;
    }
  }

  Future<void> _openChat(Conversation c, String title) async {
    await Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => ChatScreen(
          chats: widget.chats,
          conversation: c,
          title: title,
          places: widget.places,
          files: widget.files,
          media: widget.media,
          now: widget.now,
        ),
      ),
    );
    if (mounted) await _load();
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    final list = _conversations;
    return Scaffold(
      appBar: AppBar(title: Text(t.chatsTitle)),
      body: SafeArea(
        child: switch ((list, _error)) {
          (_, final String code) => Padding(
            padding: const EdgeInsets.all(24),
            child: ui.LoadFailed(code == 'unreachable' ? t.unreachable : t.somethingWrong, onRetry: _load),
          ),
          (null, _) => const ui.Loading(),
          (final List<Conversation> l, _) when l.isEmpty => Padding(
            padding: const EdgeInsets.all(24),
            child: ui.Message(t.chatsEmpty),
          ),
          (final List<Conversation> l, _) => ListView.separated(
            padding: const EdgeInsets.all(16),
            itemCount: l.length,
            separatorBuilder: (_, _) => const SizedBox(height: 8),
            itemBuilder: (context, i) {
              if (i == l.length - 1) WidgetsBinding.instance.addPostFrameCallback((_) => _more());
              return _Row(conversation: l[i], names: _names, onTap: _openChat);
            },
          ),
        },
      ),
    );
  }
}

class _Row extends StatelessWidget {
  const _Row({required this.conversation, required this.names, required this.onTap});

  final Conversation conversation;
  final Places names;
  final void Function(Conversation c, String title) onTap;

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    final named = names.roles[conversation.practitionerRoleId];
    final title = named?.$2.name ?? t.chatDoctorUnknown;
    final clinic = named?.$1.name ?? names.clinics[conversation.organizationId]?.name;
    final unread = conversation.unread;
    return Card(
      child: ListTile(
        title: Text(title),
        subtitle: clinic == null ? null : Text(clinic),
        trailing: unread == 0
            ? null
            : Semantics(
                label: t.chatUnread(unread),
                excludeSemantics: true,
                child: Badge.count(count: unread, backgroundColor: Theme.of(context).colorScheme.primary),
              ),
        onTap: () => onTap(conversation, title),
      ),
    );
  }
}
