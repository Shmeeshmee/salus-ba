import 'dart:async';
import 'dart:convert';
import 'dart:io' show WebSocketException;

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:salus/chat.dart';
import 'package:salus/chat_client.dart';
import 'package:salus/file_prep.dart';
import 'package:salus_api/api.dart';

// The real Chats against the API's exact JSON (api/openapi.yaml, tag chat; internal/communication's handler tests)
// for SAL-27's routes, and against docs/plans/chat.md "Contract of wave 1" for the wave's. Synthetic data only.

const conversationId = 'c0000000-0000-4000-8000-000000000001';
const orgId = 'a0000000-0000-4000-8000-00000000000a';
const token = 'synthetic-session-token';

Map<String, Object?> conversationJson({int unread = 2}) => {
  'id': conversationId,
  'organization_id': orgId,
  'patient_id': 'b0000000-0000-4000-8000-000000000001',
  'practitioner_role_id': 'e0000000-0000-4000-8000-000000000001',
  'created_at': '2026-11-12T08:00:00Z',
  'last_sent_at': '2026-11-12T08:05:00Z',
  'last_seq': 7,
  'cursor': 'cursor-7',
  'unread': unread,
  'staff_only': false,
};

Map<String, Object?> messageJson(
  int seq, {
  List<Map<String, Object?>>? attachments,
  String? body = 'Synthetic placeholder',
}) => {
  'id': 'f0000000-0000-4000-8000-${seq.toString().padLeft(12, '0')}',
  'conversation_id': conversationId,
  'seq': seq,
  'cursor': 'cursor-$seq',
  'sender_participant_id': 'd0000000-0000-4000-8000-000000000002',
  'sender_role': 'practitioner',
  'kind': 'text',
  'body': ?body,
  'client_message_id': 'cc000000-0000-4000-8000-${seq.toString().padLeft(12, '0')}',
  'sent_at': '2026-11-12T08:0$seq:00Z',
  'version_id': 1,
  'attachments': ?attachments,
};

Map<String, Object?> attachmentJson({String status = 'scanning', bool thumbnail = false}) => {
  'id': 'ab000000-0000-4000-8000-000000000001',
  'content_type': 'image/jpeg',
  'size': 3,
  'filename': 'synthetic-photo.jpg',
  'status': status,
  'thumbnail': thumbnail,
};

/// The API: one answer per "METHOD /path", every request recorded with its body.
class Api {
  final routes = <String, Future<http.Response> Function(http.Request)>{};
  final requests = <http.Request>[];

  void answer(String route, int status, [Object? body]) =>
      routes[route] = (_) async =>
          http.Response(body == null ? '' : jsonEncode(body), status, headers: {'content-type': 'application/json'});

  http.Request last(String route) => requests.lastWhere((r) => '${r.method} ${r.url.path}' == route);

  Future<http.Response> handle(http.Request r) {
    requests.add(r);
    final respond = routes['${r.method} ${r.url.path}'];
    return respond == null ? Future.value(http.Response('{"error":"not_found"}', 404)) : respond(r);
  }
}

/// A socket the test drives: frames in, then a close with its code.
class Line implements ChatLine {
  Line(this.url, this.headers);

  final Uri url;
  final Map<String, String> headers;
  final _frames = StreamController<dynamic>();
  var closed = false;

  @override
  int? closeCode;

  @override
  String? closeReason;

  @override
  Stream<dynamic> get incoming => _frames.stream;

  void frame(String type, Object payload) => _frames.add(jsonEncode({'type': type, 'version': 1, 'payload': payload}));

  void end(int code, [String? reason]) {
    closeCode = code;
    closeReason = reason;
    unawaited(_frames.close());
  }

  @override
  Future<void> close() async {
    closed = true;
    if (!_frames.isClosed) await _frames.close();
  }
}

/// The API and the socket the client talks to; the first [refused] handshakes fail as a refused upgrade does.
({Api api, ChatClient chats, List<Line> lines, List<Duration> waits, List<Uri> dials}) setup({int refused = 0}) {
  final api = Api();
  final lines = <Line>[];
  final waits = <Duration>[];
  final dials = <Uri>[];
  final bearer = HttpBearerAuth()..accessToken = token;
  final client = ApiClient(basePath: 'https://api.test', authentication: bearer)..client = MockClient(api.handle);
  final chats = ChatClient(
    client,
    dial: (url, headers) async {
      dials.add(url);
      if (dials.length <= refused) {
        throw const WebSocketException('Connection to the server was not upgraded to websocket');
      }
      final line = Line(url, headers);
      lines.add(line);
      return line;
    },
    wait: (d) async => waits.add(d),
  );
  return (api: api, chats: chats, lines: lines, waits: waits, dials: dials);
}

Future<void> settle() => Future<void>.delayed(Duration.zero);

void main() {
  test('lists the conversations and reads one with the bearer token', () async {
    final s = setup();
    s.api.answer('GET /conversations', 200, {
      'conversations': [conversationJson()],
      'next': 'page-2',
    });
    final page = await s.chats.list();
    expect(page.conversations.single.unread, 2);
    expect(page.next, 'page-2');
    expect(s.api.last('GET /conversations').headers['Authorization'], 'Bearer $token');
    expect(s.api.last('GET /conversations').headers.keys.map((k) => k.toLowerCase()), isNot(contains('x-salus-csrf')));

    s.api.answer('GET /conversations', 200, {'conversations': <Object>[]});
    await s.chats.list(before: 'page-2');
    expect(s.api.last('GET /conversations').url.queryParameters['before'], 'page-2');

    s.api.answer('GET /conversations/$conversationId', 200, conversationJson());
    expect((await s.chats.get(conversationId)).cursor, 'cursor-7');
  });

  test('reads a page of messages with their attachments', () async {
    final s = setup();
    s.api.answer('GET /conversations/$conversationId/messages', 200, {
      'messages': [
        messageJson(7, attachments: [attachmentJson(status: 'clean', thumbnail: true)], body: null),
        messageJson(6),
      ],
      'next': 'cursor-6',
    });
    final page = await s.chats.messages(conversationId, before: 'cursor-8', limit: 2);
    expect(s.api.last('GET /conversations/$conversationId/messages').url.queryParameters, {
      'before': 'cursor-8',
      'limit': '2',
    });
    expect(page.next, 'cursor-6');
    expect(page.messages.first.body, isNull);
    final a = page.messages.first.attachments.single;
    expect(
      (a.status, a.thumbnail, a.filename, a.contentType),
      (ChatAttachmentStatusEnum.clean, true, 'synthetic-photo.jpg', ChatAttachmentContentTypeEnum.imageSlashJpeg),
    );
    expect(page.messages.last.attachments, isEmpty);
  });

  test('sends with the client id and the attachment ids, empty for none', () async {
    final s = setup();
    s.api.answer('POST /conversations/$conversationId/messages', 201, messageJson(8));
    await s.chats.send(conversationId, 'cc000000-0000-4000-8000-000000000008', 'Synthetic placeholder');
    expect(jsonDecode(s.api.last('POST /conversations/$conversationId/messages').body), {
      'client_message_id': 'cc000000-0000-4000-8000-000000000008',
      'body': 'Synthetic placeholder',
      'attachment_ids': <String>[],
    });

    s.api.answer('POST /conversations/$conversationId/messages', 200, messageJson(8, attachments: [attachmentJson()]));
    final m = await s.chats.send(
      conversationId,
      'cc000000-0000-4000-8000-000000000009',
      '',
      attachmentIds: ['ab000000-0000-4000-8000-000000000001'],
    );
    expect(jsonDecode(s.api.last('POST /conversations/$conversationId/messages').body), {
      'client_message_id': 'cc000000-0000-4000-8000-000000000009',
      'body': '',
      'attachment_ids': ['ab000000-0000-4000-8000-000000000001'],
    });
    expect(m.attachments.single.status, ChatAttachmentStatusEnum.scanning);
  });

  test('marks read, edits with If-Match and reads the revisions', () async {
    final s = setup();
    s.api.answer('PUT /conversations/$conversationId/read', 204);
    await s.chats.markRead(conversationId, 7);
    expect(jsonDecode(s.api.last('PUT /conversations/$conversationId/read').body), {'seq': 7});

    final messageId = messageJson(7)['id'];
    s.api.answer('PATCH /conversations/$conversationId/messages/$messageId', 200, {...messageJson(7), 'version_id': 2});
    expect((await s.chats.edit(conversationId, '$messageId', 'Synthetic edit', 1)).versionId, 2);
    final edit = s.api.last('PATCH /conversations/$conversationId/messages/$messageId');
    expect(edit.headers['If-Match'], 'W/"1"');
    expect(jsonDecode(edit.body), {'body': 'Synthetic edit'});

    s.api.answer('GET /conversations/$conversationId/messages/$messageId/revisions', 200, {
      'revisions': [
        {'edited_at': '2026-11-12T08:07:00Z', 'body': 'Synthetic placeholder', 'version_id': 1},
      ],
    });
    expect((await s.chats.revisions(conversationId, '$messageId')).single.versionId, 1);
  });

  test("a refusal carries the API's code, and an unreachable API says so", () async {
    final s = setup();
    s.api.answer('POST /conversations/$conversationId/messages', 403, {'error': 'consent_required'});
    expect(
      () => s.chats.send(conversationId, 'cc000000-0000-4000-8000-000000000001', 'x'),
      throwsA(isA<ChatException>().having((e) => e.code, 'code', 'consent_required')),
    );
    s.api.routes['GET /conversations'] = (_) async => throw http.ClientException('network');
    expect(s.chats.list, throwsA(isA<ChatException>().having((e) => e.code, 'code', 'unreachable')));
    s.api.routes['GET /me/chat-invitations'] = (_) async => throw http.ClientException('network');
    expect(s.chats.invitations, throwsA(isA<ChatException>().having((e) => e.code, 'code', 'unreachable')));
  });

  test("the invitations, the patient's decision and the revocation", () async {
    final s = setup();
    // SAL-28's list is assumed to be {invitations: [...]}.
    s.api.answer('GET /me/chat-invitations', 200, {
      'invitations': [
        {
          'id': '1a000000-0000-4000-8000-000000000001',
          'conversation_id': conversationId,
          'organization_id': 'a0000000-0000-4000-8000-00000000000b',
          'practitioner_role_id': 'e0000000-0000-4000-8000-000000000002',
          'reason': 'specialist_review',
          'invited_at': '2026-11-12T08:06:00Z',
        },
      ],
    });
    final i = (await s.chats.invitations()).single;
    expect(s.api.last('GET /me/chat-invitations').url.queryParameters, {'status': 'pending_consent'});
    expect((i.reason, i.status, i.conversationId), (ChatInvitationReasonEnum.specialistReview, null, conversationId));

    s.api.answer('GET /me/chat-invitations', 200, {
      'invitations': [
        {
          'id': i.id,
          'conversation_id': conversationId,
          'organization_id': i.organizationId,
          'practitioner_role_id': i.practitionerRoleId,
          'reason': 'specialist_review',
          'invited_at': '2026-11-12T08:06:00Z',
          'status': 'active',
        },
      ],
    });
    final granted = (await s.chats.invitations(granted: true)).single;
    expect(s.api.last('GET /me/chat-invitations').url.queryParameters, {'status': 'active'});
    expect(granted.status, ChatInvitationStatusEnum.active);

    s.api.answer('POST /me/chat-invitations/${i.id}/decision', 204);
    await s.chats.decide(i.id, grant: true);
    expect(jsonDecode(s.api.last('POST /me/chat-invitations/${i.id}/decision').body), {'decision': 'grant'});
    await s.chats.decide(i.id, grant: false);
    expect(jsonDecode(s.api.last('POST /me/chat-invitations/${i.id}/decision').body), {'decision': 'decline'});
    expect(s.api.last('POST /me/chat-invitations/${i.id}/decision').headers['Authorization'], 'Bearer $token');

    s.api.answer('POST /me/chat-invitations/${i.id}/revoke', 204);
    await s.chats.revoke(i.id);
    expect(s.api.requests.last.url.path, '/me/chat-invitations/${i.id}/revoke');
  });

  test("the clinic's chat hours", () async {
    final s = setup();
    s.api.answer('GET /organizations/$orgId/chat-hours', 200, {
      'time_zone': 'Europe/Sarajevo',
      'hours': [
        {'day': 'mon', 'start': '08:00', 'end': '16:00'},
      ],
      'response_time_hours': 4,
      'open_now': false,
    });
    final h = await s.chats.chatHours(orgId);
    expect(s.api.last('GET /organizations/$orgId/chat-hours').headers['Authorization'], 'Bearer $token');
    expect(
      (h.openNow, h.responseTimeHours, h.timeZone, h.hours.single.day, h.hours.single.end),
      (false, 4, 'Europe/Sarajevo', ChatHourDayEnum.mon, '16:00'),
    );
  });

  test('uploads the bytes streamed with their length and name, and reads a clean file back', () async {
    final s = setup();
    s.api.answer('POST /conversations/$conversationId/attachments', 201, attachmentJson());
    final a = await s.chats.upload(
      conversationId,
      Prepared(
        name: 'synthetic photo.jpg',
        size: 3,
        kind: FileKind.photo,
        read: () => Stream.fromIterable([
          [1, 2],
          [3],
        ]),
        discard: () async {},
      ),
    );
    expect(a.status, ChatAttachmentStatusEnum.scanning);
    final r = s.api.last('POST /conversations/$conversationId/attachments');
    expect(r.bodyBytes, [1, 2, 3]);
    expect(r.headers['Authorization'], 'Bearer $token');
    expect(r.headers['Content-Type'], 'application/octet-stream');
    expect(r.headers['Content-Disposition'], "attachment; filename*=UTF-8''synthetic%20photo.jpg");

    s.api.routes['GET /conversations/$conversationId/attachments/${a.id}/content'] = (_) async =>
        http.Response.bytes([1, 2, 3], 200);
    s.api.routes['GET /conversations/$conversationId/attachments/${a.id}/thumbnail'] = (_) async =>
        http.Response.bytes([9], 200);
    expect(await s.chats.content(conversationId, a.id), [1, 2, 3]);
    expect(await s.chats.thumbnail(conversationId, a.id), [9]);
  });

  group('follow', () {
    test('opens the socket with the bearer token, after the cursor, and drops a seq already shown', () async {
      final s = setup();
      final seqs = <int>[];
      final edits = <int>[];
      final stop = s.chats.follow(conversationId, 'cursor-7', (f) {
        (f.type == ChatMessageFrameTypeEnum.message ? seqs : edits).add(f.payload.seq);
      }, (_) {});
      await settle();
      final line = s.lines.single;
      expect(line.url.toString(), 'wss://api.test/conversations/$conversationId/events?after=cursor-7');
      expect(line.headers, {'Authorization': 'Bearer $token'});
      expect(line.headers.keys.map((k) => k.toLowerCase()), isNot(contains('origin')));

      line
        ..frame('message', messageJson(8, attachments: [attachmentJson()]))
        ..frame('message', messageJson(8))
        ..frame('typing', {'participant_id': 'd0000000-0000-4000-8000-000000000002'})
        ..frame('message_edited', messageJson(8))
        ..frame('message', messageJson(9));
      await settle();
      expect(seqs, [8, 9]);
      expect(edits, [8]);
      stop();
      expect(line.closed, isTrue);
    });

    test('reconnects with backoff after a drop and resumes after the last message shown', () async {
      final s = setup();
      final seqs = <int>[];
      final stop = s.chats.follow(conversationId, null, (f) => seqs.add(f.payload.seq), (_) {});
      await settle();
      expect(s.lines.single.url.query, isEmpty);
      s.lines.single
        ..frame('message', messageJson(8))
        ..end(1011);
      await settle();
      await settle();
      expect(s.lines, hasLength(2));
      expect(s.lines.last.url.queryParameters['after'], 'cursor-8');
      s.lines.last
        ..frame('message', messageJson(8))
        ..frame('message', messageJson(9))
        ..end(1006);
      await settle();
      await settle();
      expect(seqs, [8, 9]);
      expect(s.waits, [const Duration(seconds: 1), const Duration(seconds: 1)]);
      s.lines.last.end(1006);
      await settle();
      await settle();
      expect(s.waits.last, const Duration(seconds: 2));
      stop();
    });

    for (final (code, reason, end) in [
      (4401, 'unauthenticated', ChatEnd.unauthenticated),
      (4403, 'consent_required', ChatEnd.consentRequired),
      (4403, 'forbidden', ChatEnd.forbidden),
      (4404, 'not_found', ChatEnd.notFound),
    ]) {
      test('stops for good on $code $reason', () async {
        final s = setup();
        final ends = <ChatEnd>[];
        s.chats.follow(conversationId, null, (_) {}, ends.add);
        await settle();
        s.lines.single.end(code, reason);
        await settle();
        await settle();
        expect(ends, [end]);
        expect(s.lines, hasLength(1));
      });
    }

    // A handshake the API refuses (401, 403 or 404 before the upgrade) comes back as an exception without the
    // status; the conversation's own answer says whether access ended.
    for (final (status, code, end) in [
      (401, 'unauthenticated', ChatEnd.unauthenticated),
      (403, 'consent_required', ChatEnd.consentRequired),
      (403, 'forbidden', ChatEnd.forbidden),
      (404, 'not_found', ChatEnd.notFound),
    ]) {
      test('ends for good when the handshake is refused and the conversation answers $status $code', () async {
        final s = setup(refused: 1);
        s.api.answer('GET /conversations/$conversationId', status, {'error': code});
        final ends = <ChatEnd>[];
        s.chats.follow(conversationId, 'cursor-7', (_) {}, ends.add);
        for (var i = 0; i < 5; i++) {
          await settle();
        }
        expect(ends, [end]);
        expect(s.dials, hasLength(1));
        expect(s.waits, isEmpty);
      });
    }

    // Each check is an audited read of the conversation: on the first refused handshake in a row, then every third.
    test('reads the conversation on the first refused handshake and then only every third', () async {
      final s = setup(refused: 7);
      s.api.answer('GET /conversations/$conversationId', 200, conversationJson());
      final stop = s.chats.follow(conversationId, 'cursor-7', (_) {}, (_) {});
      for (var i = 0; i < 40; i++) {
        await settle();
      }
      expect(s.dials, hasLength(8));
      expect(s.api.requests.where((r) => r.url.path == '/conversations/$conversationId'), hasLength(3));
      stop();
    });

    test('tries again after a refused handshake while the conversation still answers', () async {
      final s = setup(refused: 2);
      s.api.answer('GET /conversations/$conversationId', 200, conversationJson());
      final ends = <ChatEnd>[];
      final stop = s.chats.follow(conversationId, 'cursor-7', (_) {}, ends.add);
      for (var i = 0; i < 10; i++) {
        await settle();
      }
      expect(ends, isEmpty);
      expect((s.dials.length, s.lines.length), (3, 1));
      expect(s.waits, [const Duration(seconds: 1), const Duration(seconds: 2)]);
      stop();
    });
  });
}
