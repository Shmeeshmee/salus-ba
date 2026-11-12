import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:http/http.dart' as http;
import 'package:salus_api/api.dart';

import 'booking.dart' show contentDisposition;
import 'chat.dart';
import 'file_prep.dart' show Prepared;

// The patient's chat (Chats) on the API: the routes of SAL-27, SAL-28 and SAL-30 through the generated ChatApi and
// models, SAL-29's chat hours too, except where the generated client cannot stream (an upload's bytes, a file's
// bytes): those are plain requests on the same client and bearer token, read into the generated models. The socket carries the bearer token and no Origin. Nothing here
// logs.

/// A conversation's open socket, as [ChatClient] reads it; tests fake it.
abstract interface class ChatLine {
  /// The frames, as text; done when the socket closed.
  Stream<dynamic> get incoming;
  int? get closeCode;
  String? get closeReason;
  Future<void> close();
}

/// Opens a socket to [url] with [headers].
typedef Dial = Future<ChatLine> Function(Uri url, Map<String, String> headers);

/// A dart:io WebSocket: it sends no Origin unless told to, as the API asks of the app.
Future<ChatLine> dialWebSocket(Uri url, Map<String, String> headers) async =>
    _WebSocketLine(await WebSocket.connect(url.toString(), headers: headers));

class _WebSocketLine implements ChatLine {
  _WebSocketLine(this._socket);

  final WebSocket _socket;

  @override
  Stream<dynamic> get incoming => _socket;

  @override
  int? get closeCode => _socket.closeCode;

  @override
  String? get closeReason => _socket.closeReason;

  @override
  Future<void> close() => _socket.close();
}

/// The wait before reconnect [attempt] (from 0): 1, 2, 4, 8, 16, then every 30 seconds.
Duration chatBackoff(int attempt) => Duration(seconds: attempt >= 5 ? 30 : 1 << attempt);

class ChatClient implements Chats {
  ChatClient(this._client, {this._dial = dialWebSocket, Future<void> Function(Duration)? wait})
    : _api = ChatApi(_client),
      _wait = wait ?? Future<void>.delayed;

  final ApiClient _client;
  final ChatApi _api;
  final Dial _dial;
  final Future<void> Function(Duration) _wait;

  static String _seg(String id) => Uri.encodeComponent(id);

  /// Runs a call of the generated client; a refusal becomes the API's code.
  static Future<T> _call<T>(Future<T> Function() call) async {
    try {
      return await call();
    } on ApiException catch (e) {
      if (e.innerException != null) throw const ChatException('unreachable');
      throw ChatException(_code(e.message));
    }
  }

  static String _code(String? body) {
    try {
      final json = jsonDecode(body ?? '');
      if (json is Map && json['error'] is String) return json['error'] as String;
    } on FormatException {
      // Not the API's JSON.
    }
    return 'unknown';
  }

  static T _present<T>(T? value) => value ?? (throw const ChatException('unknown'));

  /// The answer of a call, refused at 400 and above.
  static http.Response _ok(http.Response r) {
    if (r.statusCode >= HttpStatus.badRequest) throw ChatException(_code(r.body));
    return r;
  }

  Future<http.Response> _plain(String method, String path, {Object? body, Map<String, String>? headers}) => _call(
    () => _client.invokeAPI(path, method, [], body, {...?headers}, {}, body == null ? null : 'application/json'),
  ).then(_ok);

  @override
  Future<ConversationsPage> list({String? before}) async {
    final l = _present(await _call(() => _api.listConversations(before: before)));
    return ConversationsPage(l.conversations, l.next);
  }

  @override
  Future<Conversation> open(OpenConversationRequest request) async =>
      _present(await _call<Conversation?>(() => _api.openConversation(request)));

  @override
  Future<Conversation> get(String id) async => _present(await _call<Conversation?>(() => _api.getConversation(id)));

  @override
  Future<MessagePage> messages(String id, {String? before, int limit = 50}) async =>
      _present(await _call<MessagePage?>(() => _api.listMessages(id, before: before, limit: limit)));

  @override
  Future<ChatMessage> send(
    String id,
    String clientMessageId,
    String body, {
    List<String> attachmentIds = const [],
  }) async => _present(
    await _call<ChatMessage?>(
      () => _api.sendMessage(
        id,
        SendMessageRequest(clientMessageId: clientMessageId, body: body, attachmentIds: attachmentIds),
      ),
    ),
  );

  @override
  Future<ChatMessage> edit(String id, String messageId, String body, int version) async => _present(
    await _call<ChatMessage?>(() => _api.editMessage(id, messageId, 'W/"$version"', EditMessageRequest(body: body))),
  );

  @override
  Future<List<MessageRevision>> revisions(String id, String messageId) async =>
      _present(await _call(() => _api.listMessageRevisions(id, messageId))).revisions;

  @override
  Future<void> markRead(String id, int seq) => _call(() => _api.markConversationRead(id, ReadPosition(seq: seq)));

  @override
  Future<List<ChatInvitation>> invitations({bool granted = false}) async =>
      _present(await _call(() => _api.listChatInvitations(status: granted ? 'active' : 'pending_consent'))).invitations;

  @override
  Future<void> decide(String invitationId, {required bool grant}) => _call(
    () => _api.decideChatInvitation(
      invitationId,
      ChatInvitationDecision(
        decision: grant ? ChatInvitationDecisionDecisionEnum.grant : ChatInvitationDecisionDecisionEnum.decline,
      ),
    ),
  );

  @override
  Future<void> revoke(String invitationId) => _call(() => _api.revokeChatInvitation(invitationId));

  @override
  Future<ChatHours> chatHours(String organizationId) async =>
      _present(await _call<ChatHours?>(() => _api.getChatHours(organizationId)));

  /// The bytes streamed as the body with their length and the name in `Content-Disposition`, as the questionnaire's
  /// uploads are (Booking.upload).
  @override
  Future<ChatAttachment> upload(String id, Prepared file) async {
    final headers = <String, String>{};
    await _client.authentication?.applyToParams(<QueryParam>[], headers);
    final request = http.StreamedRequest('POST', Uri.parse('${_client.basePath}/conversations/${_seg(id)}/attachments'))
      ..contentLength = file.size
      ..headers.addAll({
        ...headers,
        'Content-Type': 'application/octet-stream',
        'Content-Disposition': contentDisposition(file.name),
      });
    final http.Response r;
    try {
      final sending = _client.client.send(request);
      await file.read().pipe(request.sink);
      r = await http.Response.fromStream(await sending);
    } on Exception {
      throw const ChatException('unreachable');
    }
    return _present(ChatAttachment.fromJson(jsonDecode(_ok(r).body)));
  }

  Future<Uint8List> _bytes(String id, String attachmentId, String part) async => (await _plain(
    'GET',
    '/conversations/${_seg(id)}/attachments/${_seg(attachmentId)}/$part',
    headers: {'Accept': '*/*'},
  )).bodyBytes;

  @override
  Future<Uint8List> content(String id, String attachmentId) => _bytes(id, attachmentId, 'content');

  @override
  Future<Uint8List> thumbnail(String id, String attachmentId) => _bytes(id, attachmentId, 'thumbnail');

  Uri _events(String id, String? after) {
    final base = Uri.parse(_client.basePath);
    return base.replace(
      scheme: base.scheme == 'https' ? 'wss' : 'ws',
      path: '${base.path}/conversations/${_seg(id)}/events',
      queryParameters: after == null ? null : {'after': after},
    );
  }

  /// Why the socket ended for good, or null when it is to be opened again.
  static ChatEnd? _ended(int? code, String? reason) => switch (code) {
    4401 => ChatEnd.unauthenticated,
    4403 => reason == 'consent_required' ? ChatEnd.consentRequired : ChatEnd.forbidden,
    4404 => ChatEnd.notFound,
    _ => null,
  };

  /// Why the conversation is refused, as its socket would have closed; null while it still answers or the API
  /// cannot be reached.
  Future<ChatEnd?> _refused(String id) async {
    try {
      await get(id);
      return null;
    } on ChatException catch (e) {
      return switch (e.code) {
        'unauthenticated' => ChatEnd.unauthenticated,
        'consent_required' => ChatEnd.consentRequired,
        'forbidden' => ChatEnd.forbidden,
        'not_found' => ChatEnd.notFound,
        _ => null,
      };
    }
  }

  /// A `message` or `message_edited` frame; any other (`send_ack`, `receipt`, `typing`, …) is null.
  static ChatMessageFrame? _frame(Object? data) {
    try {
      final json = jsonDecode(data as String) as Map<String, dynamic>;
      final type = ChatMessageFrameTypeEnum.fromJson(json['type']);
      if (type == null) return null;
      return ChatMessageFrame(
        type: type,
        version: ChatMessageFrameVersionEnum.number1,
        payload: _present(ChatMessage.fromJson(json['payload'])),
      );
    } on Object {
      return null; // A frame the app does not know is ignored.
    }
  }

  /// Reconnects with [chatBackoff] after a drop, resuming after the last message forwarded; a `seq` already
  /// forwarded is dropped. Stops for good on 4401, 4403 and 4404 with [onEnd], and when a handshake fails and the
  /// conversation itself is refused (a refused upgrade gives no status to read).
  @override
  void Function() follow(
    String id,
    String? after,
    void Function(ChatMessageFrame frame) onFrame,
    void Function(ChatEnd end) onEnd,
  ) {
    var stopped = false;
    var cursor = after;
    int? last;
    ChatLine? line;
    var attempt = 0;
    var refused = 0; // handshakes failed in a row

    void forward(ChatMessageFrame frame) {
      if (frame.type == ChatMessageFrameTypeEnum.message) {
        if (last != null && frame.payload.seq <= last!) return;
        last = frame.payload.seq;
        cursor = frame.payload.cursor;
      }
      onFrame(frame);
    }

    Future<void> run() async {
      while (!stopped) {
        final headers = <String, String>{};
        await _client.authentication?.applyToParams(<QueryParam>[], headers);
        final ChatLine open;
        try {
          open = line = await _dial(_events(id, cursor), headers);
          refused = 0;
        } on Exception {
          // Each check is an audited read: on the first failed handshake in a row, then on every third.
          refused++;
          final end = refused == 1 || refused % 3 == 0 ? await _refused(id) : null;
          if (stopped) return;
          if (end != null) return onEnd(end);
          await _wait(chatBackoff(attempt++));
          continue;
        }
        try {
          if (stopped) return unawaited(open.close());
          await for (final data in open.incoming) {
            attempt = 0;
            final frame = _frame(data);
            if (frame != null && !stopped) forward(frame);
          }
          final end = _ended(open.closeCode, open.closeReason);
          if (end != null) {
            if (!stopped) onEnd(end);
            return;
          }
        } on Exception {
          // A socket that broke is opened again.
        }
        if (stopped) return;
        await _wait(chatBackoff(attempt++));
      }
    }

    unawaited(run());
    return () {
      stopped = true;
      unawaited(line?.close());
    };
  }
}
