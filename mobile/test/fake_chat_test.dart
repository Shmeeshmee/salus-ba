import 'package:flutter_test/flutter_test.dart';
import 'package:salus/chat.dart';
import 'package:salus_api/api.dart';

import 'fake_chat.dart';

String clientId(int n) => 'c${n.toString().padLeft(7, '0')}-0000-4000-8000-000000000000';

void main() {
  test('pages a long history newest first and walks back by the cursor', () async {
    final chat = FakeChat(history: 2500);
    final id = chat.conversation.id;
    final first = await chat.messages(id);
    expect(first.messages, hasLength(50));
    expect(first.messages.first.seq, 2500);
    var next = first.next;
    var seen = first.messages.length;
    while (next != null) {
      final page = await chat.messages(id, before: next, limit: 100);
      seen += page.messages.length;
      next = page.next;
    }
    expect(seen, 2500);
  });

  test('sends once per client id and counts what the patient has not read', () async {
    final chat = FakeChat();
    final id = chat.conversation.id;
    final a = await chat.send(id, clientId(1), '  Synthetic placeholder  ');
    expect((await chat.send(id, clientId(1), 'typed again')).id, a.id);
    expect(a.body, 'Synthetic placeholder');
    chat.deliver('Synthetic placeholder answer');
    expect((await chat.get(id)).unread, 1);
    await chat.markRead(id, 2);
    await chat.markRead(id, 1);
    expect((await chat.get(id)).unread, 0);
    expect(() => chat.send(id, clientId(2), ' '), throwsA(isA<ChatException>().having((e) => e.code, 'code', 'invalid_request')));
  });

  test('lets only the sender edit the current version and keeps the earlier body', () async {
    final chat = FakeChat();
    final id = chat.conversation.id;
    final m = await chat.send(id, clientId(1), 'Synthetic placeholder');
    final theirs = chat.deliver('Synthetic placeholder answer');
    expect((await chat.edit(id, m.id, 'Synthetic placeholder, edited', 1)).versionId, 2);
    expect(() => chat.edit(id, m.id, 'stale', 1), throwsA(isA<ChatException>().having((e) => e.code, 'code', 'version_conflict')));
    expect(() => chat.edit(id, theirs.id, 'not mine', 1), throwsA(isA<ChatException>().having((e) => e.code, 'code', 'forbidden')));
    expect((await chat.revisions(id, m.id)).single.body, 'Synthetic placeholder');
  });

  test('replays after the cursor, forwards live frames once and ends with the reason', () async {
    final chat = FakeChat(history: 3);
    final id = chat.conversation.id;
    final page = await chat.messages(id);
    final seqs = <int>[];
    final ends = <ChatEnd>[];
    final stop = chat.follow(id, page.messages[1].cursor, (f) => seqs.add(f.payload.seq), ends.add);
    chat.deliver('Synthetic placeholder live');
    expect(seqs, [3, 4]);
    chat.end(ChatEnd.consentRequired);
    expect(ends, [ChatEnd.consentRequired]);
    stop();
    expect(ChatMessageFrameTypeEnum.message.toJson(), 'message');
  });

  test("gives the clinic's chat hours, and not_found for another clinic", () async {
    final closed = ChatHours(
      timeZone: 'Europe/Sarajevo',
      hours: [ChatHour(day: ChatHourDayEnum.mon, start: '08:00', end: '16:00')],
      responseTimeHours: 8,
      openNow: false,
    );
    final chat = FakeChat(hours: closed);
    expect((await chat.chatHours(chat.conversation.organizationId)).openNow, isFalse);
    expect((await FakeChat().chatHours(chat.conversation.organizationId)).openNow, isTrue);
    expect(() => chat.chatHours('00000000-0000-4000-8000-000000000000'), throwsA(isA<ChatException>()));
  });
}
