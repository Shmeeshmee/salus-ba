import 'package:salus/chat.dart';
import 'package:salus_api/api.dart';

String _uuid(int n, String prefix) => '${prefix.padRight(8, '0')}-0000-4000-8000-${n.toRadixString(16).padLeft(12, '0')}';

String _cursor(int seq) => 'fake-$seq';

int _seqOf(String cursor) => int.parse(cursor.substring('fake-'.length));

/// The chat (Chats) in memory, for tests and for building the screens on synthetic data. It keeps the API's rules the
/// screens see: pages newest first by cursor, idempotent sends, edits by the sender with the current version, the
/// read position never going back, each frame once. The caller is the conversation's patient.
class FakeChat implements Chats {
  FakeChat({int history = 0, DateTime? start}) : _clock = start ?? DateTime.utc(2026, 11, 12, 8) {
    final at = _clock;
    conversation = Conversation(
      id: _uuid(1, 'c'),
      organizationId: _uuid(1, 'a'),
      patientId: _uuid(1, 'b'),
      practitionerRoleId: _uuid(1, 'e'),
      createdAt: at,
      lastSeq: 0,
      cursor: _cursor(0),
      unread: 0,
      staffOnly: false,
      participants: [
        ConversationParticipant(id: _me, role: ConversationParticipantRoleEnum.patient, patientId: _uuid(1, 'b'), addedAt: at, historyFromSeq: 0),
        ConversationParticipant(id: _doctor, role: ConversationParticipantRoleEnum.practitioner, practitionerRoleId: _uuid(1, 'e'), addedAt: at, historyFromSeq: 0),
      ],
    );
    for (var i = 0; i < history; i++) {
      _write(i.isOdd ? _doctor : _me, 'Synthetic placeholder message ${i + 1}', _uuid(i + 1, 'cc'));
    }
  }

  static final _me = _uuid(1, 'd');
  static final _doctor = _uuid(2, 'd');

  late final Conversation conversation;
  final _messages = <ChatMessage>[];
  final _revisions = <String, List<MessageRevision>>{};
  final _clientIds = <String, ChatMessage>{};
  final _followers = <void Function(ChatMessageFrame)>{};
  final _enders = <void Function(ChatEnd)>{};
  DateTime _clock;
  int _ids = 0;

  /// The patient's read position: the API never shows it, only the unread count.
  int _readSeq = 0;

  ChatMessage _write(String from, String body, String clientId) {
    _clock = _clock.add(const Duration(seconds: 1));
    final seq = _messages.length + 1;
    final m = ChatMessage(
      id: _uuid(++_ids, 'f'),
      conversationId: conversation.id,
      seq: seq,
      cursor: _cursor(seq),
      senderParticipantId: from,
      senderRole: from == _me ? ChatMessageSenderRoleEnum.patient : ChatMessageSenderRoleEnum.practitioner,
      kind: ChatMessageKindEnum.text,
      body: body,
      clientMessageId: clientId,
      sentAt: _clock,
      versionId: 1,
    );
    _messages.add(m);
    conversation
      ..lastSeq = seq
      ..lastSentAt = m.sentAt
      ..cursor = m.cursor;
    if (from == _me) _readSeq = seq;
    conversation.unread = conversation.lastSeq - _readSeq;
    for (final f in _followers.toList()) {
      f(ChatMessageFrame(type: ChatMessageFrameTypeEnum.message, version: ChatMessageFrameVersionEnum.number1, payload: m));
    }
    return m;
  }

  void _known(String id) {
    if (id != conversation.id) throw const ChatException('not_found');
  }

  static String _body(String text) {
    final t = text.trim();
    if (t.isEmpty || t.length > 4000) throw const ChatException('invalid_request');
    return t;
  }

  /// Writes a message as the practitioner, as if it came over the socket.
  ChatMessage deliver(String body) => _write(_doctor, _body(body), _uuid(++_ids, 'dd'));

  /// Ends every socket of the conversation with the reason.
  void end(ChatEnd reason) {
    _followers.clear();
    for (final e in _enders.toList()) {
      e(reason);
    }
    _enders.clear();
  }

  @override
  Future<ConversationsPage> list({String? before}) async => ConversationsPage([conversation], null);

  @override
  Future<Conversation> open(OpenConversationRequest request) async => conversation;

  @override
  Future<Conversation> get(String id) async {
    _known(id);
    return conversation;
  }

  @override
  Future<MessagePage> messages(String id, {String? before, int limit = 50}) async {
    _known(id);
    if (limit < 1 || limit > 100) throw const ChatException('invalid_request');
    final end = before == null ? _messages.length : _seqOf(before) - 1;
    final start = end - limit < 0 ? 0 : end - limit;
    final page = _messages.sublist(start, end).reversed.toList();
    final oldest = page.isEmpty ? null : page.last;
    return MessagePage(messages: page, next: oldest != null && oldest.seq > 1 ? oldest.cursor : null);
  }

  @override
  Future<ChatMessage> send(String id, String clientMessageId, String body) async {
    _known(id);
    return _clientIds[clientMessageId] ??= _write(_me, _body(body), clientMessageId);
  }

  @override
  Future<ChatMessage> edit(String id, String messageId, String body, int version) async {
    _known(id);
    final m = _messages.where((m) => m.id == messageId).firstOrNull;
    if (m == null) throw const ChatException('not_found');
    if (m.senderParticipantId != _me) throw const ChatException('forbidden');
    if (m.versionId != version) throw const ChatException('version_conflict');
    _revisions.putIfAbsent(m.id, () => []).add(MessageRevision(editedAt: m.editedAt ?? m.sentAt, body: m.body ?? '', versionId: m.versionId));
    _clock = _clock.add(const Duration(seconds: 1));
    m
      ..body = _body(body)
      ..versionId = m.versionId + 1
      ..editedAt = _clock;
    for (final f in _followers.toList()) {
      f(ChatMessageFrame(type: ChatMessageFrameTypeEnum.messageEdited, version: ChatMessageFrameVersionEnum.number1, payload: m));
    }
    return m;
  }

  @override
  Future<List<MessageRevision>> revisions(String id, String messageId) async {
    _known(id);
    return List.of(_revisions[messageId] ?? const []);
  }

  @override
  Future<void> markRead(String id, int seq) async {
    _known(id);
    if (seq < 0 || seq > conversation.lastSeq) throw const ChatException('invalid_request');
    if (seq > _readSeq) _readSeq = seq;
    conversation.unread = conversation.lastSeq - _readSeq;
  }

  @override
  void Function() follow(String id, String? after, void Function(ChatMessageFrame frame) onFrame, void Function(ChatEnd end) onEnd) {
    _known(id);
    var last = after == null ? _messages.length : _seqOf(after);
    void forward(ChatMessageFrame frame) {
      if (frame.type == ChatMessageFrameTypeEnum.message) {
        if (frame.payload.seq <= last) return;
        last = frame.payload.seq;
      }
      onFrame(frame);
    }

    for (final m in _messages.sublist(last)) {
      forward(ChatMessageFrame(type: ChatMessageFrameTypeEnum.message, version: ChatMessageFrameVersionEnum.number1, payload: m));
    }
    _followers.add(forward);
    _enders.add(onEnd);
    return () {
      _followers.remove(forward);
      _enders.remove(onEnd);
    };
  }
}
