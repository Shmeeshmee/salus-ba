import 'dart:convert';
import 'dart:typed_data';

import 'package:salus/chat.dart';
import 'package:salus/file_prep.dart';
import 'package:salus_api/api.dart';

String _uuid(int n, String prefix) => '${prefix.padRight(8, '0')}-0000-4000-8000-${n.toRadixString(16).padLeft(12, '0')}';

String _cursor(int seq) => 'fake-$seq';

int _seqOf(String cursor) => int.parse(cursor.substring('fake-'.length));

/// A 1 × 1 PNG: the thumbnail of every clean image.
final syntheticThumbnail = base64Decode(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
);

/// The chat (Chats) in memory, for tests and for building the screens on synthetic data. It keeps the API's rules the
/// screens see: pages newest first by cursor, idempotent sends, edits by the sender with the current version, the
/// read position never going back, each frame once. The caller is the conversation's patient. It also has SAL-28's
/// invitations of doctors from other clinics (each a pending participant of the conversation until granted) with
/// their system messages, SAL-30's attachments whose scan a test moves on (at most 10 unsent), and the clinic's chat
/// hours as the plan's contract shapes them for SAL-29.
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
        ConversationParticipant(id: _me, role: ConversationParticipantRoleEnum.patient, status: ConversationParticipantStatusEnum.active, patientId: _uuid(1, 'b'), addedAt: at, historyFromSeq: 0),
        ConversationParticipant(id: _doctor, role: ConversationParticipantRoleEnum.practitioner, status: ConversationParticipantStatusEnum.active, practitionerRoleId: _uuid(1, 'e'), organizationId: _uuid(1, 'a'), addedAt: at, historyFromSeq: 0),
      ],
    );
    for (var i = 0; i < history; i++) {
      _write(i.isOdd ? _doctor : _me, body: 'Synthetic placeholder message ${i + 1}', clientId: _uuid(i + 1, 'cc'));
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

  /// The clinic's chat hours: Monday to Friday, 08:00 to 16:00, an answer within a day; a test sets [openNow].
  var openNow = true;
  var responseTimeHours = 24;
  final _invitations = <ChatInvitation>[];
  final _granted = <ChatInvitation>[];

  /// The invitations the patient revoked, in order.
  final revoked = <String>[];
  final _uploads = <String, (ChatAttachment, List<int>)>{};

  /// The patient's read position: the API never shows it, only the unread count.
  int _readSeq = 0;

  /// How many uploads the patient may have waiting to be sent (SAL-30).
  static const unsentMaximum = 10;
  final _unsent = <String>{};

  ChatMessage _write(
    String? from, {
    String? body,
    SystemMessage? system,
    List<ChatAttachment> attachments = const [],
    String? clientId,
  }) {
    _clock = _clock.add(const Duration(seconds: 1));
    final seq = _messages.length + 1;
    final m = ChatMessage(
      id: _uuid(++_ids, 'f'),
      conversationId: conversation.id,
      seq: seq,
      cursor: _cursor(seq),
      senderParticipantId: from,
      senderRole: switch (from) {
        null => ChatMessageSenderRoleEnum.system,
        _ when from == _me => ChatMessageSenderRoleEnum.patient,
        _ => ChatMessageSenderRoleEnum.practitioner,
      },
      kind: system == null ? ChatMessageKindEnum.text : ChatMessageKindEnum.system,
      body: body,
      system: system,
      clientMessageId: clientId ?? _uuid(++_ids, 'dd'),
      sentAt: _clock,
      versionId: 1,
      attachments: attachments,
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
  ChatMessage deliver(String body) => _write(_doctor, body: _body(body));

  /// Sends a file as the practitioner, with its scan at [status].
  ChatMessage deliverFile(String filename, ChatAttachmentContentTypeEnum contentType, ChatAttachmentStatusEnum status) {
    final a = _attachment(filename, contentType, syntheticThumbnail);
    scanned(a.id, status);
    return _write(_doctor, attachments: [a]);
  }

  /// Writes a system message with its opaque arguments.
  ChatMessage system(String code, Map<String, String> args) => _write(null, system: SystemMessage(code: code, args: args));

  /// The treating doctor invites a doctor from another clinic: a pending participant until the patient decides,
  /// reading the whole history or, with [fromSeq], the messages after it.
  ChatInvitation invite({ChatInvitationReasonEnum reason = ChatInvitationReasonEnum.secondOpinion, int fromSeq = 0}) {
    final n = _invitations.length + _granted.length + 2;
    final i = ChatInvitation(
      id: _uuid(n, '1a'),
      conversationId: conversation.id,
      organizationId: _uuid(n, 'a'),
      practitionerRoleId: _uuid(n, 'e'),
      reason: reason,
      invitedAt: _clock,
      status: ChatInvitationStatusEnum.pendingConsent,
    );
    _invitations.add(i);
    conversation.participants.add(
      ConversationParticipant(
        id: i.id,
        role: ConversationParticipantRoleEnum.practitioner,
        status: ConversationParticipantStatusEnum.pendingConsent,
        practitionerRoleId: i.practitionerRoleId,
        organizationId: i.organizationId,
        addedAt: _clock,
        historyFromSeq: fromSeq,
      ),
    );
    system('participant_invited', {
      'practitioner_role_id': i.practitionerRoleId,
      'organization_id': i.organizationId,
      'reason': reason.toString(),
    });
    return i;
  }

  /// The scan of an upload ends: a clean image has a thumbnail.
  void scanned(String attachmentId, ChatAttachmentStatusEnum status) {
    final a = _uploads[attachmentId]!.$1..status = status;
    a.thumbnail = status == ChatAttachmentStatusEnum.clean && a.contentType != ChatAttachmentContentTypeEnum.applicationSlashPdf;
  }

  ChatAttachment _attachment(String filename, ChatAttachmentContentTypeEnum contentType, List<int> bytes) {
    final a = ChatAttachment(
      id: _uuid(++_ids, 'ab'),
      contentType: contentType,
      size: bytes.length,
      filename: filename,
      status: ChatAttachmentStatusEnum.scanning,
      thumbnail: false,
    );
    _uploads[a.id] = (a, bytes);
    return a;
  }

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
  Future<ChatMessage> send(String id, String clientMessageId, String body, {List<String> attachmentIds = const []}) async {
    _known(id);
    final first = _clientIds[clientMessageId];
    if (first != null) return first;
    final files = [
      for (final a in attachmentIds)
        switch (_uploads[a]?.$1) {
          final f? when f.status != ChatAttachmentStatusEnum.rejected && _unsent.contains(a) => f,
          _ => throw const ChatException('invalid_request'),
        },
    ];
    final text = files.isNotEmpty && body.trim().isEmpty ? null : _body(body);
    final m = _clientIds[clientMessageId] = _write(_me, body: text, attachments: files, clientId: clientMessageId);
    _unsent.removeAll(attachmentIds);
    return m;
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

  @override
  Future<List<ChatInvitation>> invitations({bool granted = false}) async => List.of(granted ? _granted : _invitations);

  @override
  Future<void> decide(String invitationId, {required bool grant}) async {
    final i = _invitations.where((i) => i.id == invitationId).firstOrNull;
    if (i == null) throw const ChatException('not_found');
    _invitations.remove(i);
    final role = {'practitioner_role_id': i.practitionerRoleId};
    if (!grant) {
      conversation.participants.removeWhere((p) => p.id == i.id);
      system('consent_declined', role);
      return;
    }
    _granted.add(i..status = ChatInvitationStatusEnum.active);
    for (final p in conversation.participants.where((p) => p.id == i.id)) {
      p.status = ConversationParticipantStatusEnum.active;
    }
    system('consent_granted', role);
    system('participant_added', {...role, 'organization_id': i.organizationId, 'reason': i.reason.toString()});
  }

  @override
  Future<void> revoke(String invitationId) async {
    final i = _granted.where((i) => i.id == invitationId).firstOrNull;
    if (i == null) throw const ChatException('not_found');
    _granted.remove(i);
    revoked.add(i.id);
    conversation.participants.removeWhere((p) => p.id == i.id);
    system('participant_removed', {'practitioner_role_id': i.practitionerRoleId});
  }

  @override
  Future<ChatHours> chatHours(String organizationId) async {
    if (organizationId != conversation.organizationId) throw const ChatException('not_found');
    return ChatHours(
      hours: [for (var d = 1; d <= 5; d++) ChatHoursDay(d, '08:00', '16:00')],
      responseTimeHours: responseTimeHours,
      openNow: openNow,
    );
  }

  @override
  Future<ChatAttachment> upload(String id, Prepared file) async {
    _known(id);
    final bytes = [await for (final chunk in file.read()) ...chunk];
    final type = switch (file.kind) {
      FileKind.photo => ChatAttachmentContentTypeEnum.imageSlashJpeg,
      FileKind.document => ChatAttachmentContentTypeEnum.applicationSlashPdf,
      FileKind.video => throw const ChatException('unsupported_type'),
    };
    if (_unsent.length >= unsentMaximum) throw const ChatException('too_many_files');
    final a = _attachment(file.name, type, bytes);
    _unsent.add(a.id);
    return a;
  }

  ChatAttachment _clean(String id, String attachmentId) {
    _known(id);
    final a = _uploads[attachmentId]?.$1;
    if (a == null || a.status != ChatAttachmentStatusEnum.clean) throw const ChatException('not_found');
    return a;
  }

  @override
  Future<Uint8List> content(String id, String attachmentId) async =>
      Uint8List.fromList(_uploads[_clean(id, attachmentId).id]!.$2);

  @override
  Future<Uint8List> thumbnail(String id, String attachmentId) async {
    if (!_clean(id, attachmentId).thumbnail) throw const ChatException('not_found');
    return syntheticThumbnail;
  }
}
