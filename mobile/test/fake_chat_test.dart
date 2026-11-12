import 'package:flutter_test/flutter_test.dart';
import 'package:salus/file_prep.dart';
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

  test('an invitation of a doctor from another clinic is pending until the patient decides', () async {
    final chat = FakeChat();
    final id = chat.conversation.id;
    final invitation = chat.invite();
    expect((await chat.invitations()).single.id, invitation.id);
    expect(invitation.conversationId, id);
    final pending = (await chat.get(id)).participants.singleWhere((p) => p.id == invitation.id);
    expect((pending.status, pending.historyFromSeq), (ConversationParticipantStatusEnum.pendingConsent, 0));
    final invited = (await chat.messages(id)).messages.first;
    expect(invited.kind, ChatMessageKindEnum.system);
    expect(invited.system!.code, 'participant_invited');
    expect(invited.system!.args, {
      'practitioner_role_id': invitation.practitionerRoleId,
      'organization_id': invitation.organizationId,
      'reason': 'second_opinion',
    });

    await chat.decide(invitation.id, grant: true);
    expect(await chat.invitations(), isEmpty);
    expect([for (final g in await chat.invitations(granted: true)) (g.id, g.status)], [(invitation.id, ChatInvitationStatusEnum.active)]);
    expect([for (final m in (await chat.messages(id)).messages.take(2)) m.system!.code], ['participant_added', 'consent_granted']);
    expect(() => chat.decide(invitation.id, grant: false), throwsA(isA<ChatException>().having((e) => e.code, 'code', 'not_found')));

    await chat.revoke(invitation.id);
    expect((await chat.messages(id)).messages.first.system!.code, 'participant_removed');
    expect(chat.revoked, [invitation.id]);
    expect(await chat.invitations(granted: true), isEmpty);
    expect((await chat.get(id)).participants.where((p) => p.id == invitation.id), isEmpty);
  });

  test('a declined invitation writes consent_declined and cannot be revoked', () async {
    final chat = FakeChat();
    final invitation = chat.invite(reason: ChatInvitationReasonEnum.handover, fromSeq: 1);
    expect((await chat.get(chat.conversation.id)).participants.last.historyFromSeq, 1);
    await chat.decide(invitation.id, grant: false);
    final last = (await chat.messages(chat.conversation.id)).messages.first;
    expect(last.system!.code, 'consent_declined');
    expect(last.system!.args, {'practitioner_role_id': invitation.practitionerRoleId});
    expect(() => chat.revoke(invitation.id), throwsA(isA<ChatException>().having((e) => e.code, 'code', 'not_found')));
    expect((await chat.get(chat.conversation.id)).participants.where((p) => p.id == invitation.id), isEmpty);
  });

  test('an attachment is scanning until the test moves it, and its bytes are read once clean', () async {
    final chat = FakeChat();
    final id = chat.conversation.id;
    final file = Prepared(
      name: 'synthetic-photo.jpg',
      size: 3,
      kind: FileKind.photo,
      read: () => Stream.value([1, 2, 3]),
      discard: () async {},
    );
    final a = await chat.upload(id, file);
    expect(a.status, ChatAttachmentStatusEnum.scanning);
    expect(a.contentType, ChatAttachmentContentTypeEnum.imageSlashJpeg);
    expect(() => chat.content(id, a.id), throwsA(isA<ChatException>()));

    final m = await chat.send(id, clientId(1), '', attachmentIds: [a.id]);
    expect(m.body, isNull);
    expect(m.attachments.single.id, a.id);
    expect(() => chat.send(id, clientId(2), '', attachmentIds: ['unknown']), throwsA(isA<ChatException>()));

    chat.scanned(a.id, ChatAttachmentStatusEnum.clean);
    final shown = (await chat.messages(id)).messages.first;
    expect(shown.attachments.single.status, ChatAttachmentStatusEnum.clean);
    expect(shown.attachments.single.thumbnail, isTrue);
    expect(await chat.content(id, a.id), [1, 2, 3]);
    expect(await chat.thumbnail(id, a.id), isNotEmpty);
  });

  test('the doctor sends an attachment', () async {
    final chat = FakeChat();
    final m = chat.deliverFile('synthetic-report.pdf', ChatAttachmentContentTypeEnum.applicationSlashPdf, ChatAttachmentStatusEnum.clean);
    expect(m.attachments.single.filename, 'synthetic-report.pdf');
    expect(m.attachments.single.thumbnail, isFalse);
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
