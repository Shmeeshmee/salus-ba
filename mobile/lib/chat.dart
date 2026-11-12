// The chat's contract for the patient's screens (docs/plans/chat.md, SAL-27): the calls of the API's chat tag and
// the conversation's socket, as one interface (Chats; the generated ChatApi is the raw HTTP client) on the generated
// models. test/fake_chat.dart implements it in memory, and lib/chat_client.dart on the API. Nothing here logs, and
// no message text leaves the app but to the API.
//
// The wave's calls (docs/plans/chat.md "Contract of wave 1": SAL-28's invitations, SAL-29's chat hours, SAL-30's
// attachments) are not in api/openapi.yaml yet: their types are written here by hand, field names as the contract
// has them on the wire, and move to the generated types once SAL-28 to SAL-30 are merged.
import 'dart:math';
import 'dart:typed_data';

import 'package:salus_api/api.dart';

import 'file_prep.dart' show Prepared;

/// A random (version 4) UUID, for a message's client id: the app has no other UUID to make, so no package for it.
String uuidV4([Random? random]) {
  final r = random ?? Random.secure();
  final b = List<int>.generate(16, (_) => r.nextInt(256));
  b[6] = (b[6] & 0x0f) | 0x40;
  b[8] = (b[8] & 0x3f) | 0x80;
  final h = [for (final x in b) x.toRadixString(16).padLeft(2, '0')].join();
  return '${h.substring(0, 8)}-${h.substring(8, 12)}-${h.substring(12, 16)}-${h.substring(16, 20)}-${h.substring(20)}';
}

/// Why a conversation's socket ended for good: the session ended (close code 4401) or access ended (4403, 4404).
enum ChatEnd { unauthenticated, consentRequired, forbidden, notFound }

/// A refused call, with the API's error code; `unreachable` when the API could not be reached.
class ChatException implements Exception {
  const ChatException(this.code);

  final String code;

  @override
  String toString() => 'ChatException($code)';
}

/// A page of the caller's conversations and the cursor of the next one (null at the end).
class ConversationsPage {
  const ConversationsPage(this.conversations, this.next);

  final List<Conversation> conversations;
  final String? next;
}

/// A pending invitation of a doctor from another clinic to one of the patient's conversations (SAL-28,
/// `GET /me/chat-invitations`). [history] (`full` or `from`) is read when the API sends it; the contract does not
/// promise it, so the screen says "the conversation" without it.
class ChatInvitation {
  const ChatInvitation({
    required this.id,
    required this.conversationId,
    required this.organizationId,
    required this.practitionerRoleId,
    required this.reason,
    required this.invitedAt,
    this.history,
  });

  final String id;
  final String conversationId;
  final String organizationId;
  final String practitionerRoleId;

  /// `second_opinion`, `specialist_review` or `handover`.
  final String reason;
  final DateTime invitedAt;
  final String? history;

  static ChatInvitation fromJson(Map<String, dynamic> json) => ChatInvitation(
    id: json['id'] as String,
    conversationId: json['conversation_id'] as String,
    organizationId: json['organization_id'] as String,
    practitionerRoleId: json['practitioner_role_id'] as String,
    reason: json['reason'] as String,
    invitedAt: DateTime.parse(json['invited_at'] as String),
    history: json['history'] as String?,
  );
}

/// One weekday's chat hours in the clinic's zone (SAL-29): [weekday] ISO (Monday 1), [start] and [end] `HH:MM`.
class ChatHoursDay {
  const ChatHoursDay(this.weekday, this.start, this.end);

  final int weekday;
  final String start;
  final String end;
}

/// The clinic's chat hours, its expected response time and whether it is within its hours now (SAL-29,
/// `GET /organizations/{id}/chat-hours`). The contract names `hours` per weekday with `start` and `end`; the shape
/// `[{weekday, start, end}]` is assumed until SAL-29 is merged.
class ChatHours {
  const ChatHours({required this.hours, required this.responseTimeHours, required this.openNow});

  final List<ChatHoursDay> hours;
  final int responseTimeHours;
  final bool openNow;

  static ChatHours fromJson(Map<String, dynamic> json) => ChatHours(
    hours: [
      for (final d in (json['hours'] as List? ?? const []).cast<Map<String, dynamic>>())
        ChatHoursDay(d['weekday'] as int, d['start'] as String, d['end'] as String),
    ],
    responseTimeHours: json['response_time_hours'] as int,
    openNow: json['open_now'] as bool,
  );
}

/// Where an attachment's scan stands (SAL-30).
enum AttachmentStatus { scanning, clean, rejected }

/// A file sent with a message (SAL-30 `ChatAttachment`). [thumbnail] is whether `…/thumbnail` has an image: the
/// contract names the field without its type, so a `true` or a non-empty string both count.
class ChatAttachment {
  ChatAttachment({
    required this.id,
    required this.contentType,
    required this.size,
    required this.filename,
    required this.status,
    this.thumbnail = false,
  });

  final String id;
  final String contentType;
  final int size;
  final String filename;
  AttachmentStatus status;
  bool thumbnail;

  bool get image => contentType.startsWith('image/');

  static ChatAttachment fromJson(Map<String, dynamic> json) {
    final thumbnail = json['thumbnail'];
    return ChatAttachment(
      id: json['id'] as String,
      contentType: json['content_type'] as String,
      size: json['size'] as int,
      filename: json['filename'] as String,
      status: AttachmentStatus.values.firstWhere(
        (s) => s.name == json['status'],
        orElse: () => AttachmentStatus.scanning,
      ),
      thumbnail: thumbnail == true || (thumbnail is String && thumbnail.isNotEmpty),
    );
  }
}

final _attachments = Expando<List<ChatAttachment>>('attachments');

/// SAL-30's `ChatMessage.attachments`, kept beside the generated model until the generated client has the field;
/// then this extension is deleted and the screens read the generated field of the same name.
extension ChatMessageAttachments on ChatMessage {
  List<ChatAttachment> get attachments => _attachments[this] ?? const [];
  set attachments(List<ChatAttachment> list) => _attachments[this] = list;
}

abstract interface class Chats {
  /// A page of the caller's conversations, newest activity first.
  Future<ConversationsPage> list({String? before});

  Future<Conversation> open(OpenConversationRequest request);

  Future<Conversation> get(String id);

  /// A page newest first, before the cursor; its next is the older page's cursor, null at the start.
  Future<MessagePage> messages(String id, {String? before, int limit = 50});

  /// Idempotent by clientMessageId: a retry with the same id gives the first message. With [attachmentIds] (the
  /// caller's uploads), [body] may be empty.
  Future<ChatMessage> send(String id, String clientMessageId, String body, {List<String> attachmentIds = const []});

  Future<ChatMessage> edit(String id, String messageId, String body, int version);

  Future<List<MessageRevision>> revisions(String id, String messageId);

  Future<void> markRead(String id, int seq);

  /// Every message after the cursor (or from now on) and then the live ones, each once, until the returned
  /// function is called or access ends.
  void Function() follow(String id, String? after, void Function(ChatMessageFrame frame) onFrame, void Function(ChatEnd end) onEnd);

  /// The patient's pending invitations of doctors from other clinics, every conversation's.
  Future<List<ChatInvitation>> invitations();

  /// The patient's answer to an invitation: a grant is a consent for that conversation and that doctor.
  Future<void> decide(String invitationId, {required bool grant});

  /// Revokes a granted invitation: the doctor loses access at once.
  Future<void> revoke(String invitationId);

  /// The chat hours of the clinic [organizationId].
  Future<ChatHours> chatHours(String organizationId);

  /// Uploads [file] to the conversation; answered `scanning` until the scan has run.
  Future<ChatAttachment> upload(String id, Prepared file);

  /// A clean attachment's bytes.
  Future<Uint8List> content(String id, String attachmentId);

  /// A clean image's thumbnail.
  Future<Uint8List> thumbnail(String id, String attachmentId);
}
