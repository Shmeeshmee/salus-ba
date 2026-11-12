import 'dart:async';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:salus_api/api.dart';

import 'booking.dart' show Places;
import 'call.dart' show Media;
import 'chat.dart';
import 'chat_bubbles.dart';
import 'chat_composer.dart';
import 'chat_invitation.dart';
import 'chat_list_screen.dart' show Names;
import 'file_prep.dart';
import 'files.dart';
import 'messages.dart';
import 'questionnaire.dart' show Item, ItemType;
import 'questionnaire_files.dart' show pickFile;
import 'ui.dart' as ui;
import 'uploads.dart' show pollDelay;

// The patient's chat view (SAL-31): the messages in a lazy list, newest at the bottom, older pages read when the
// patient scrolls up, live ones from the socket, each marked read; the composer with files from the camera or the
// files; the clinic's chat hours and the out-of-hours notice; the consent prompt for a doctor of another clinic and
// the revocation of every one granted in this conversation (listed by the API, so also after a restart). Messages
// are marked read only while the newest is in view and the app is in the foreground. A doctor-only thread or an AI summary never reaches the patient: the screen
// reads only this conversation's messages. Nothing here logs.

/// What a chat file may be: a photo (the camera or the library) or a PDF from the documents.
const _chatFiles = Item(linkId: 'chat', type: ItemType.attachment, accept: {FileKind.photo, FileKind.document});

class ChatScreen extends StatefulWidget {
  const ChatScreen({
    super.key,
    required this.chats,
    required this.conversation,
    required this.title,
    required this.places,
    required this.files,
    required this.media,
    required this.now,
  });

  final Chats chats;
  final Conversation conversation;

  /// The treating doctor's name, or the catalog's "Your doctor".
  final String title;
  final Names places;
  final Files files;
  final Media media;
  final DateTime Function() now;

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> with WidgetsBindingObserver {
  /// Newest first, as the reversed list shows them from the bottom.
  final _messages = <ChatMessage>[];
  final _seqs = <int>{};
  String? _older;
  var _loadingOlder = false;
  var _olderFailed = false;
  var _loading = true;
  String? _error;
  var _readSeq = 0;

  ChatHours? _hours;
  var _invitations = <ChatInvitation>[];

  /// What each invited doctor would read (`full` or `from`), by the invitation's id, which is the invited
  /// participant's: their `history_from_seq` in the conversation.
  var _histories = <String, String>{};

  /// The doctors of other clinics the patient granted access to this conversation, which they may revoke from the
  /// menu.
  var _granted = <ChatInvitation>[];
  Places _names = const Places({}, {});
  var _deciding = false;

  ChatEnd? _ended;
  void Function()? _stop;

  final _text = TextEditingController();
  final _scroll = ScrollController();

  /// Whether the newest message is in view (the list is reversed: its start is the bottom) and the app is in the
  /// foreground: only then is the newest message read.
  var _atBottom = true;
  var _foreground = true;

  /// The client id of the message being sent; kept after a failure so that the retry cannot send it twice.
  String? _clientId;
  var _sending = false;
  String? _sendError;
  final _pending = <PendingFile>[];

  final _thumbnails = <String, Future<Uint8List>>{};
  Timer? _scan;
  var _scans = 0;

  String get _id => widget.conversation.id;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _foreground = switch (WidgetsBinding.instance.lifecycleState) {
      null || AppLifecycleState.resumed => true,
      _ => false,
    };
    _scroll.addListener(_scrolled);
    _text.addListener(() => setState(() {}));
    _load();
    unawaited(_loadHours());
    unawaited(_loadInvitations());
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _scroll.dispose();
    _stop?.call();
    _scan?.cancel();
    _text.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final page = await widget.chats.messages(_id);
      if (!mounted) return;
      setState(() {
        _loading = false;
        _older = page.next;
        _addOlder(page.messages);
      });
      _stop?.call();
      _stop = widget.chats.follow(_id, page.messages.firstOrNull?.cursor ?? widget.conversation.cursor, _onFrame, _onEnd);
      _read();
      _watchScans();
    } on ChatException catch (e) {
      if (mounted) {
        setState(() {
          _loading = false;
          _error = e.code;
        });
      }
    }
  }

  void _addOlder(List<ChatMessage> page) {
    for (final m in page) {
      if (_seqs.add(m.seq)) _messages.add(m);
    }
  }

  void _insert(ChatMessage m) {
    if (!_seqs.add(m.seq)) return;
    final i = _messages.indexWhere((x) => x.seq < m.seq);
    i < 0 ? _messages.add(m) : _messages.insert(i, m);
  }

  Future<void> _loadOlder() async {
    if (_loadingOlder || _older == null || _olderFailed) return;
    _loadingOlder = true;
    try {
      final page = await widget.chats.messages(_id, before: _older);
      if (!mounted) return;
      setState(() {
        _addOlder(page.messages);
        _older = page.next;
      });
    } on ChatException {
      if (mounted) setState(() => _olderFailed = true);
    } finally {
      _loadingOlder = false;
    }
  }

  void _onFrame(ChatMessageFrame frame) {
    if (!mounted) return;
    final m = frame.payload;
    setState(() {
      if (frame.type == ChatMessageFrameTypeEnum.messageEdited) {
        final i = _messages.indexWhere((x) => x.id == m.id);
        if (i >= 0) _messages[i] = m;
      } else {
        _insert(m);
      }
    });
    if (m.system?.code case 'participant_invited' || 'participant_removed') unawaited(_loadInvitations());
    _read();
    _watchScans();
  }

  void _onEnd(ChatEnd end) {
    if (mounted) setState(() => _ended = end);
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    _read();
  }

  void _scrolled() {
    final bottom = _scroll.position.pixels <= 48;
    if (bottom == _atBottom) return;
    _atBottom = bottom;
    _read();
  }

  /// Moves the read position to the newest message, while it is in view and the app in the foreground; a failure
  /// is tried again with the next message.
  void _read() {
    if (!_atBottom || !_foreground) return;
    final newest = _messages.firstOrNull?.seq ?? 0;
    if (newest <= _readSeq) return;
    final before = _readSeq;
    _readSeq = newest;
    widget.chats.markRead(_id, newest).catchError((Object _) {
      if (_readSeq == newest) _readSeq = before;
    });
  }

  /// While a recent message has a file being checked, its page is read again (every 2 s at first, as for the
  /// questionnaire's files) until the scan has ended.
  void _watchScans() {
    final scanning = _messages.take(50).any((m) => m.attachments.any((a) => a.status == ChatAttachmentStatusEnum.scanning));
    if (!scanning) {
      _scans = 0;
      return;
    }
    _scan ??= Timer(pollDelay(_scans++), _rescan);
  }

  Future<void> _rescan() async {
    _scan = null;
    try {
      final page = await widget.chats.messages(_id);
      if (!mounted) return;
      final fresh = {for (final m in page.messages) m.id: m};
      setState(() {
        for (final m in _messages.take(50)) {
          final f = fresh[m.id];
          if (f != null) m.attachments = f.attachments;
        }
      });
    } on ChatException {
      // Tried again at the next delay.
    }
    if (mounted) _watchScans();
  }

  Future<void> _loadHours() async {
    try {
      final hours = await widget.chats.chatHours(widget.conversation.organizationId);
      if (mounted) setState(() => _hours = hours);
    } on ChatException {
      // Without the hours the chat works; the notice and the hours button stay away.
    }
  }

  Future<void> _loadInvitations() async {
    try {
      final mine = [for (final i in await widget.chats.invitations()) if (i.conversationId == _id) i];
      final granted = [for (final i in await widget.chats.invitations(granted: true)) if (i.conversationId == _id) i];
      final names = await _namesOf([...mine, ...granted]);
      var histories = _histories;
      if (mine.isNotEmpty) {
        try {
          final participants = (await widget.chats.get(_id)).participants;
          histories = {for (final p in participants) p.id: p.historyFromSeq == 0 ? 'full' : 'from'};
        } on ChatException {
          // The prompt says "the conversation" without it.
        }
      }
      if (mounted) {
        setState(() {
          _invitations = mine;
          _granted = granted;
          _names = names;
          _histories = histories;
        });
      }
    } on ChatException {
      // Asked again when the chat is next opened or the next invitation arrives.
    }
  }

  /// The names of the invited doctors and their clinics; those known so far when they cannot be read.
  Future<Places> _namesOf(List<ChatInvitation> invitations) async {
    if (invitations.isEmpty) return _names;
    try {
      return await widget.places({for (final i in invitations) i.organizationId});
    } on Exception {
      return _names; // The prompt waits for them (InvitationCard) and the menu names no doctor.
    }
  }

  Future<void> _retryNames() async {
    final names = await _namesOf([..._invitations, ..._granted]);
    if (mounted) setState(() => _names = names);
  }

  String _failure(ChatException e) => e.code == 'unreachable' ? context.t.unreachable : context.t.somethingWrong;

  void _say(String text) => ScaffoldMessenger.of(context)
    ..hideCurrentSnackBar()
    ..showSnackBar(SnackBar(content: Text(text)));

  Future<void> _decide(ChatInvitation invitation, bool grant) async {
    setState(() => _deciding = true);
    try {
      await widget.chats.decide(invitation.id, grant: grant);
      if (!mounted) return;
      setState(() {
        _invitations = [for (final i in _invitations) if (i.id != invitation.id) i];
        if (grant) _granted.add(invitation);
      });
      _say(grant ? context.t.chatGranted : context.t.chatDeclined);
    } on ChatException catch (e) {
      if (mounted) _say(_failure(e));
    } finally {
      if (mounted) setState(() => _deciding = false);
    }
  }

  Future<void> _revoke(ChatInvitation invitation) async {
    if (!await confirmRevoke(context) || !mounted) return;
    try {
      await widget.chats.revoke(invitation.id);
      if (!mounted) return;
      setState(() => _granted.remove(invitation));
      _say(context.t.chatRevoked);
    } on ChatException catch (e) {
      if (mounted) _say(_failure(e));
    }
  }

  Future<void> _attach() async {
    final t = context.t;
    final r = await pickFile(context, _chatFiles, media: widget.media, files: widget.files, now: widget.now);
    if (r == null || !mounted) return;
    final picked = r.picked;
    if (picked == null) return _say(r.error ?? t.cannotPrepareFile);
    final entry = PendingFile(picked.title ?? picked.name);
    setState(() => _pending.add(entry));
    Prepared? file;
    try {
      file = await widget.files.prepare(picked);
      if (picked.title case final title?) {
        file = Prepared(name: title, size: file.size, kind: file.kind, read: file.read, discard: file.discard);
      }
      final name = file.name;
      if (mounted) setState(() => entry.name = name);
      final a = await widget.chats.upload(_id, file);
      if (mounted) setState(() => entry.attachment = a);
    } on TooLarge catch (e) {
      _drop(entry, t.fileTooLarge(e.kind.maxMegabytes));
    } on CannotPrepare {
      _drop(entry, t.cannotPrepareFile);
    } on ChatException catch (e) {
      _drop(entry, e.code == 'too_many_files' ? t.chatTooManyFiles : t.fileNotUploaded);
    } finally {
      await file?.discard();
    }
  }

  void _drop(PendingFile entry, String why) {
    if (!mounted) return;
    setState(() => _pending.remove(entry));
    _say(why);
  }

  bool get _uploading => _pending.any((p) => p.attachment == null);

  bool get _canSend =>
      !_sending && !_uploading && (_text.text.trim().isNotEmpty || _pending.isNotEmpty);

  Future<void> _send() async {
    final t = context.t;
    final files = [for (final p in _pending) p.attachment!.id];
    final id = _clientId ??= uuidV4();
    setState(() {
      _sending = true;
      _sendError = null;
    });
    try {
      final m = await widget.chats.send(_id, id, _text.text.trim(), attachmentIds: files);
      if (!mounted) return;
      setState(() {
        _insert(m);
        _text.clear();
        _pending.clear();
        _clientId = null;
      });
      _read();
      _watchScans();
    } on ChatException catch (e) {
      if (mounted) setState(() => _sendError = e.code == 'unreachable' ? t.chatSendFailed : t.somethingWrong);
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  Future<void> _open(ChatAttachment a) async {
    final t = context.t;
    try {
      final bytes = await widget.chats.content(_id, a.id);
      if (!mounted) return;
      if (a.contentType == ChatAttachmentContentTypeEnum.applicationSlashPdf) return await widget.files.sharePdf(a.filename, bytes);
      await showDialog<void>(
        context: context,
        builder: (context) => Dialog(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Flexible(child: InteractiveViewer(child: Image.memory(bytes, semanticLabel: a.filename))),
              TextButton(onPressed: () => Navigator.of(context).pop(), child: Text(t.termClose)),
            ],
          ),
        ),
      );
    } on ChatException {
      if (mounted) _say(t.chatOpenFailed);
    }
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    final hours = _hours;
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.title),
        actions: [
          if (hours != null)
            IconButton(tooltip: t.chatHours, icon: const Icon(Icons.schedule), onPressed: () => showChatHours(context, hours)),
          if (_granted.isNotEmpty)
            PopupMenuButton<ChatInvitation>(
              onSelected: _revoke,
              itemBuilder: (_) => [
                for (final g in _granted)
                  PopupMenuItem(
                    value: g,
                    child: Text(switch (_names.roles[g.practitionerRoleId]) {
                      (final clinic, final role) => t.chatRevokeWho(role.name, clinic.name),
                      null => t.chatRevoke,
                    }),
                  ),
              ],
            ),
        ],
      ),
      body: SafeArea(
        child: Column(
          children: [
            if (_invitations.isNotEmpty)
              ConstrainedBox(
                constraints: BoxConstraints(maxHeight: MediaQuery.sizeOf(context).height * 0.5),
                child: SingleChildScrollView(
                  padding: const EdgeInsets.all(12),
                  child: Column(
                    children: [
                      for (final i in _invitations)
                        InvitationCard(
                          invitation: i,
                          history: _histories[i.id] ?? 'unknown',
                          names: _names,
                          busy: _deciding,
                          onDecide: (grant) => _decide(i, grant),
                          onRetry: _retryNames,
                        ),
                    ],
                  ),
                ),
              ),
            Expanded(child: _list()),
            if (_ended != null)
              Padding(
                padding: const EdgeInsets.all(16),
                child: ui.Message(_ended == ChatEnd.unauthenticated ? t.chatEndedSession : t.chatEnded),
              )
            else ...[
              if (hours != null && !hours.openNow)
                Container(
                  width: double.infinity,
                  color: Theme.of(context).colorScheme.primaryContainer.withValues(alpha: 0.5),
                  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                  child: ui.Message(t.chatOutOfHours(hours.responseTimeHours)),
                ),
              if (_sendError != null)
                Padding(padding: const EdgeInsets.fromLTRB(16, 8, 16, 0), child: ui.Message(_sendError!, error: true)),
              ChatComposer(
                text: _text,
                pending: _pending,
                onAttach: _attach,
                onRemove: (p) => setState(() => _pending.remove(p)),
                onSend: _canSend ? _send : null,
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _list() {
    if (_loading) return const ui.Loading();
    if (_error != null) {
      return Padding(
        padding: const EdgeInsets.all(24),
        child: ui.LoadFailed(_error == 'unreachable' ? context.t.unreachable : context.t.somethingWrong, onRetry: _load),
      );
    }
    final more = _older != null;
    return ListView.builder(
      key: const Key('chat-messages'),
      controller: _scroll,
      reverse: true,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      itemCount: _messages.length + (more ? 1 : 0),
      itemBuilder: (context, i) {
        if (more && !_olderFailed && i >= _messages.length - 10) {
          WidgetsBinding.instance.addPostFrameCallback((_) => _loadOlder());
        }
        if (i == _messages.length) {
          return _olderFailed
              ? Center(
                  child: TextButton(
                    onPressed: () {
                      setState(() => _olderFailed = false);
                      _loadOlder();
                    },
                    child: Text(context.t.tryAgain),
                  ),
                )
              : const SizedBox(height: 48);
        }
        final m = _messages[i];
        return ChatBubble(
          key: ValueKey(m.id),
          message: m,
          mine: m.senderRole == ChatMessageSenderRoleEnum.patient,
          onOpen: _open,
          thumbnail: (a) => _thumbnails.putIfAbsent(a.id, () => widget.chats.thumbnail(_id, a.id)),
        );
      },
    );
  }
}
