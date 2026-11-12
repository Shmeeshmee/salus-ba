// The chat's contract for the patient's screens (docs/plans/chat.md, SAL-27): the calls of the API's chat tag and
// the conversation's socket, as one interface (Chats; the generated ChatApi is the raw HTTP client) on the generated models. test/fake_chat.dart implements it in memory,
// so SAL-31's screens are built and tested before the backend of SAL-28 to SAL-30 is merged; the real client
// (SAL-31) calls the API behind the same interface. Nothing here logs, and no message text leaves the app but to
// the API.
import 'package:salus_api/api.dart';

/// Why a conversation's socket ended for good: access ended (close codes 4403 and 4404).
enum ChatEnd { consentRequired, forbidden, notFound }

/// A refused call, with the API's error code.
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

abstract interface class Chats {
  /// A page of the caller's conversations, newest activity first.
  Future<ConversationsPage> list({String? before});

  Future<Conversation> open(OpenConversationRequest request);

  Future<Conversation> get(String id);

  /// A page newest first, before the cursor; its next is the older page's cursor, null at the start.
  Future<MessagePage> messages(String id, {String? before, int limit = 50});

  /// Idempotent by clientMessageId: a retry with the same id gives the first message.
  Future<ChatMessage> send(String id, String clientMessageId, String body);

  Future<ChatMessage> edit(String id, String messageId, String body, int version);

  Future<List<MessageRevision>> revisions(String id, String messageId);

  Future<void> markRead(String id, int seq);

  /// Every message after the cursor (or from now on) and then the live ones, each once, until the returned
  /// function is called or access ends.
  void Function() follow(String id, String? after, void Function(ChatMessageFrame frame) onFrame, void Function(ChatEnd end) onEnd);
}
