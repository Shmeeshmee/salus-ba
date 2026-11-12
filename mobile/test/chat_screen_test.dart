import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salus/booking.dart';
import 'package:salus/chat.dart';
import 'package:salus/chat_bubbles.dart';
import 'package:salus/chat_list_screen.dart';
import 'package:salus/fhir.dart';
import 'package:salus/file_prep.dart' show Prepared;
import 'package:salus/l10n/app_localizations.dart';
import 'package:salus/l10n/language.dart';
import 'package:salus_api/api.dart'
    show
        ChatAttachment,
        ChatAttachmentContentTypeEnum,
        ChatHour,
        ChatHourDayEnum,
        ChatHours,
        ChatAttachmentStatusEnum,
        ChatInvitation,
        ChatInvitationReasonEnum,
        ChatMessage,
        Conversation;

import 'fake_call.dart';
import 'fake_chat.dart';
import 'fake_files.dart';

// The patient's chat screens on the fake (test/fake_chat.dart). Synthetic data only.

final t = lookupAppLocalizations(localeOf('en'));

/// The names the app knows: the treating doctor's clinic and role, and the invited doctor's.
Future<Places> names(Iterable<String> clinicIds) async {
  const clinic = Organization(
    id: 'a0000000-0000-4000-8000-000000000001',
    name: 'Synthetic clinic Čaršija',
    timeZone: 'Europe/Sarajevo',
  );
  const other = Organization(
    id: 'a0000000-0000-4000-8000-000000000002',
    name: 'Synthetic clinic Ilidža',
    timeZone: 'Europe/Sarajevo',
  );
  return const Places({}, {
    'e0000000-0000-4000-8000-000000000001': (
      clinic,
      PractitionerRole(id: 'e0000000-0000-4000-8000-000000000001', name: 'Cy Example'),
    ),
    'e0000000-0000-4000-8000-000000000002': (
      other,
      PractitionerRole(id: 'e0000000-0000-4000-8000-000000000002', name: 'Dee Example'),
    ),
  });
}

class Rig {
  Rig(this.chat, {this.places = names, this.onSessionEnded});

  final FakeChat chat;
  final VoidCallback? onSessionEnded;
  final Future<Places> Function(Iterable<String> clinicIds) places;
  final files = FakeFiles();
  final media = FakeMedia();

  Future<void> pump(WidgetTester tester) async {
    await tester.pumpWidget(
      MaterialApp(
        locale: localeOf('en'),
        supportedLocales: AppLocalizations.supportedLocales,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        home: ChatListScreen(
          chats: chat,
          places: places,
          files: files,
          media: media,
          now: () => DateTime.utc(2026, 11, 12, 9),
          onSessionEnded: onSessionEnded ?? () {},
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  Future<void> open(WidgetTester tester) async {
    await pump(tester);
    await tester.tap(find.text('Cy Example'));
    await tester.pumpAndSettle();
  }
}

/// Fails the first send as unreachable, then sends.
class FlakyChat extends FakeChat {
  final ids = <String>[];

  @override
  Future<ChatMessage> send(String id, String clientMessageId, String body, {List<String> attachmentIds = const []}) {
    ids.add(clientMessageId);
    if (ids.length == 1) throw const ChatException('unreachable');
    return super.send(id, clientMessageId, body, attachmentIds: attachmentIds);
  }
}

/// Fails the first read of the granted invitations, as a dropped connection would.
class GrantedOnceFailingChat extends FakeChat {
  var _failed = false;

  @override
  Future<List<ChatInvitation>> invitations({bool granted = false}) async {
    if (granted && !_failed) {
      _failed = true;
      throw const ChatException('unreachable');
    }
    return super.invitations(granted: granted);
  }
}

/// Refuses every upload: the patient already has 10 files waiting to be sent (SAL-30).
class FullChat extends FakeChat {
  @override
  Future<ChatAttachment> upload(String id, Prepared file) async => throw const ChatException('too_many_files');
}

/// Lists a doctor-only side thread too, as an API that forgot the rule would.
class LeakyChat extends FakeChat {
  @override
  Future<ConversationsPage> list({String? before}) async {
    final page = await super.list();
    final thread = Conversation(
      id: 'c0000000-0000-4000-8000-000000000002',
      organizationId: conversation.organizationId,
      patientId: conversation.patientId,
      practitionerRoleId: 'e0000000-0000-4000-8000-000000000002',
      parentId: conversation.id,
      createdAt: conversation.createdAt,
      lastSeq: 0,
      cursor: 'fake-0',
      unread: 0,
      staffOnly: true,
    );
    return ConversationsPage([...page.conversations, thread], null);
  }
}

final v4 = RegExp(r'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$');

Future<void> write(WidgetTester tester, String text) async {
  await tester.enterText(find.byType(TextField), text);
  await tester.pump();
  await tester.tap(find.byTooltip(t.chatSend));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('the list shows the doctor, the clinic and the unread count; reading the chat clears it', (tester) async {
    final semantics = tester.ensureSemantics();
    final chat = FakeChat(history: 1)
      ..deliver('Synthetic placeholder answer')
      ..deliver('Synthetic placeholder follow-up');
    final rig = Rig(chat);
    await rig.pump(tester);
    expect(find.text(t.chatsTitle), findsOneWidget);
    expect(find.text('Synthetic clinic Čaršija'), findsOneWidget);
    expect(find.bySemanticsLabel(RegExp(t.chatUnread(2))), findsOneWidget);

    await tester.tap(find.text('Cy Example'));
    await tester.pumpAndSettle();
    expect(find.text('Synthetic placeholder follow-up'), findsOneWidget);
    expect((await chat.get(chat.conversation.id)).unread, 0);

    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.bySemanticsLabel(RegExp(t.chatUnread(2))), findsNothing);
    semantics.dispose();
  });

  testWidgets('the list never shows a doctor-only thread', (tester) async {
    await Rig(LeakyChat()).pump(tester);
    expect(find.text('Cy Example'), findsOneWidget);
    expect(find.text('Dee Example'), findsNothing);
  });

  testWidgets('an empty list says so', (tester) async {
    final chat = _Empty();
    await Rig(chat).pump(tester);
    expect(find.text(t.chatsEmpty), findsOneWidget);
  });

  testWidgets('sends with a fresh v4 client id, newest at the bottom', (tester) async {
    final chat = FakeChat(history: 2);
    await Rig(chat).open(tester);
    await write(tester, 'Synthetic placeholder question');
    final sent = (await chat.messages(chat.conversation.id)).messages.first;
    expect(sent.body, 'Synthetic placeholder question');
    expect(sent.clientMessageId, matches(v4));
    expect(find.text('Synthetic placeholder question'), findsOneWidget);
    expect(
      tester.getTopLeft(find.text('Synthetic placeholder question')).dy,
      greaterThan(tester.getTopLeft(find.text('Synthetic placeholder message 2')).dy),
    );
    expect(tester.widget<TextField>(find.byType(TextField)).controller!.text, isEmpty);

    await write(tester, 'Synthetic placeholder second');
    final second = (await chat.messages(chat.conversation.id)).messages.first;
    expect(second.clientMessageId, isNot(sent.clientMessageId));
  });

  testWidgets('a send that failed keeps the text and is retried with the same client id', (tester) async {
    final chat = FlakyChat();
    await Rig(chat).open(tester);
    await write(tester, 'Synthetic placeholder question');
    expect(find.text(t.chatSendFailed), findsOneWidget);
    expect(tester.widget<TextField>(find.byType(TextField)).controller!.text, 'Synthetic placeholder question');

    await tester.tap(find.byTooltip(t.chatSend));
    await tester.pumpAndSettle();
    expect(chat.ids, hasLength(2));
    expect(chat.ids[1], chat.ids[0]);
    expect(find.text(t.chatSendFailed), findsNothing);
    expect(find.text('Synthetic placeholder question'), findsOneWidget);
  });

  testWidgets("the doctor's live message appears once and is marked read", (tester) async {
    final chat = FakeChat(history: 1);
    await Rig(chat).open(tester);
    chat.deliver('Synthetic placeholder answer');
    await tester.pumpAndSettle();
    expect(find.text('Synthetic placeholder answer'), findsOneWidget);
    expect((await chat.get(chat.conversation.id)).unread, 0);
  });

  testWidgets('system messages are written from the catalog, never from their arguments', (tester) async {
    final chat = FakeChat();
    chat
      ..invite(reason: ChatInvitationReasonEnum.specialistReview)
      ..system('participant_removed', {'practitioner_role_id': 'e0000000-0000-4000-8000-000000000002'})
      ..system('something_new', {});
    await Rig(chat).open(tester);
    expect(find.text(t.chatSystemInvited('specialist_review')), findsOneWidget);
    expect(find.text(t.chatSystemRemoved), findsOneWidget);
    expect(find.text(t.chatSystemOther), findsOneWidget);
    expect(find.textContaining('e0000000'), findsNothing);
  });

  testWidgets('the consent prompt explains the invitation; granting lets the patient revoke it later', (tester) async {
    final chat = FakeChat();
    final invitation = chat.invite();
    await Rig(chat).open(tester);
    expect(find.text(t.chatInviteTitle), findsOneWidget);
    expect(find.text(t.chatInviteBody('second_opinion')), findsOneWidget);
    expect(find.text(t.chatInviteWho('Dee Example', 'Synthetic clinic Ilidža')), findsOneWidget);
    expect(find.text(t.chatInviteSees('full')), findsOneWidget);
    expect(find.text(t.chatInviteChoice), findsOneWidget);

    await tester.ensureVisible(find.widgetWithText(FilledButton, t.chatGrant));
    await tester.tap(find.widgetWithText(FilledButton, t.chatGrant));
    await tester.pumpAndSettle();
    expect(find.text(t.chatGranted), findsOneWidget);
    expect(find.text(t.chatInviteTitle), findsNothing);
    expect(find.text(t.chatSystemAdded('second_opinion')), findsOneWidget);

    expect(find.text(t.chatGrantedTo('Dee Example', 'Synthetic clinic Ilidža')), findsOneWidget);
    await tester.tap(find.widgetWithText(OutlinedButton, t.chatRevokeConfirm));
    await tester.pumpAndSettle();
    expect(find.text(t.chatRevokeBody), findsOneWidget);
    await tester.tap(find.widgetWithText(TextButton, t.chatRevokeConfirm));
    await tester.pumpAndSettle();
    expect(chat.revoked, [invitation.id]);
    expect(find.text(t.chatRevoked), findsOneWidget);
    expect(find.text(t.chatGrantedTo('Dee Example', 'Synthetic clinic Ilidža')), findsNothing);
  });

  testWidgets('the prompt says the doctor would read from a message on; declining ends it', (tester) async {
    final chat = FakeChat(history: 3);
    chat.invite(reason: ChatInvitationReasonEnum.handover, fromSeq: 2);
    await Rig(chat).open(tester);
    expect(find.text(t.chatInviteSees('from')), findsOneWidget);
    await tester.ensureVisible(find.widgetWithText(OutlinedButton, t.chatDecline));
    await tester.tap(find.widgetWithText(OutlinedButton, t.chatDecline));
    await tester.pumpAndSettle();
    expect(await chat.invitations(), isEmpty);
    expect(find.text(t.chatDeclined), findsOneWidget);
    expect(find.text(t.chatSystemDeclined), findsOneWidget);
    expect(find.text(t.chatGrantedTo('Dee Example', 'Synthetic clinic Ilidža')), findsNothing);
  });

  testWidgets('a doctor granted earlier can still be revoked after the app restarts', (tester) async {
    final chat = FakeChat();
    final invitation = chat.invite();
    await chat.decide(invitation.id, grant: true);
    await Rig(chat).open(tester);
    expect(find.text(t.chatInviteTitle), findsNothing);

    expect(find.text(t.chatGrantedTo('Dee Example', 'Synthetic clinic Ilidža')), findsOneWidget);
    await tester.tap(find.widgetWithText(OutlinedButton, t.chatRevokeConfirm));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(TextButton, t.chatRevokeConfirm));
    await tester.pumpAndSettle();
    expect(chat.revoked, [invitation.id]);
    expect(find.text(t.chatGrantedTo('Dee Example', 'Synthetic clinic Ilidža')), findsNothing);
  });

  testWidgets('access cannot be granted until the invited doctor and their clinic are shown', (tester) async {
    final chat = FakeChat();
    chat.invite();
    var calls = 0;
    Future<Places> flaky(Iterable<String> ids) {
      if (++calls == 2) throw const SocketException('synthetic');
      return names(ids);
    }

    await Rig(chat, places: flaky).open(tester);
    expect(find.text(t.chatInviteUnresolved), findsOneWidget);
    expect(find.textContaining('Dee Example'), findsNothing);
    expect(tester.widget<FilledButton>(find.widgetWithText(FilledButton, t.chatGrant)).onPressed, isNull);
    expect(tester.widget<OutlinedButton>(find.widgetWithText(OutlinedButton, t.chatDecline)).onPressed, isNotNull);

    await tester.ensureVisible(find.widgetWithText(TextButton, t.tryAgain));
    await tester.tap(find.widgetWithText(TextButton, t.tryAgain));
    await tester.pumpAndSettle();
    expect(find.text(t.chatInviteWho('Dee Example', 'Synthetic clinic Ilidža')), findsOneWidget);
    expect(find.text(t.chatInviteUnresolved), findsNothing);
    expect(tester.widget<FilledButton>(find.widgetWithText(FilledButton, t.chatGrant)).onPressed, isNotNull);
  });

  testWidgets('messages are marked read only while the newest is in view and the app is in the foreground', (
    tester,
  ) async {
    final chat = FakeChat(history: 60);
    final id = chat.conversation.id;
    await Rig(chat).open(tester);
    expect((await chat.get(id)).unread, 0);

    for (final state in [AppLifecycleState.inactive, AppLifecycleState.hidden, AppLifecycleState.paused]) {
      tester.binding.handleAppLifecycleStateChanged(state);
    }
    chat.deliver('Synthetic placeholder in the background');
    await tester.pumpAndSettle();
    expect((await chat.get(id)).unread, 1);
    for (final state in [AppLifecycleState.hidden, AppLifecycleState.inactive, AppLifecycleState.resumed]) {
      tester.binding.handleAppLifecycleStateChanged(state);
    }
    await tester.pumpAndSettle();
    expect((await chat.get(id)).unread, 0);

    await tester.drag(find.byKey(const Key('chat-messages')), const Offset(0, 1500));
    await tester.pumpAndSettle();
    chat.deliver('Synthetic placeholder while reading older ones');
    await tester.pumpAndSettle();
    expect((await chat.get(id)).unread, 1);
    await tester.drag(find.byKey(const Key('chat-messages')), const Offset(0, -5000));
    await tester.pumpAndSettle();
    expect((await chat.get(id)).unread, 0);
  });

  testWidgets('an upload refused for ten files waiting says so', (tester) async {
    final chat = FullChat();
    await Rig(chat).open(tester);
    await tester.tap(find.byTooltip(t.chatAttach));
    await tester.pumpAndSettle();
    await tester.tap(find.text(t.chooseFromPhotos));
    await tester.pumpAndSettle();
    expect(find.text(t.chatTooManyFiles(10)), findsOneWidget);
    expect(find.byType(InputChip), findsNothing);
  });

  testWidgets('an invitation sent while the chat is open is asked at once', (tester) async {
    final chat = FakeChat();
    await Rig(chat).open(tester);
    chat.invite();
    await tester.pumpAndSettle();
    expect(find.text(t.chatInviteTitle), findsOneWidget);
  });

  testWidgets('outside the chat hours a notice gives the response time; the hours are a tap away', (tester) async {
    final chat = FakeChat(
      hours: ChatHours(
        timeZone: 'Europe/Sarajevo',
        hours: [
          for (final day in [ChatHourDayEnum.mon, ChatHourDayEnum.tue])
            ChatHour(day: day, start: '08:00', end: '16:00'),
        ],
        responseTimeHours: 4,
        openNow: false,
      ),
    );
    await Rig(chat).open(tester);
    expect(find.text(t.chatOutOfHours(4)), findsOneWidget);

    await tester.tap(find.byTooltip(t.chatHours));
    await tester.pumpAndSettle();
    expect(find.text(t.chatHoursDay('Monday', '08:00', '16:00')), findsOneWidget);
    expect(find.text(t.chatHoursDay('Tuesday', '08:00', '16:00')), findsOneWidget);
    expect(find.text(t.chatResponseTime(4)), findsOneWidget);
    expect(find.text(t.chatHoursZone('Europe/Sarajevo')), findsOneWidget);
  });

  testWidgets('within the chat hours there is no notice', (tester) async {
    await Rig(FakeChat()).open(tester);
    expect(find.text(t.chatOutOfHours(24)), findsNothing);
  });

  testWidgets('a photo is uploaded, sent, checked, and its thumbnail shown once clean', (tester) async {
    final chat = FakeChat();
    final rig = Rig(chat);
    await rig.open(tester);
    await tester.tap(find.byTooltip(t.chatAttach));
    await tester.pumpAndSettle();
    await tester.tap(find.text(t.chooseFromPhotos));
    await tester.pumpAndSettle();
    expect(find.text('synthetic-photo.jpg'), findsOneWidget);
    expect(rig.files.discarded, 1);

    await tester.tap(find.byTooltip(t.chatSend));
    await tester.pumpAndSettle();
    final sent = (await chat.messages(chat.conversation.id)).messages.first;
    expect(sent.attachments, hasLength(1));
    expect(find.text(t.checkingFile), findsOneWidget);

    chat.scanned(sent.attachments.single.id, ChatAttachmentStatusEnum.clean);
    await tester.pump(const Duration(seconds: 2));
    await tester.pumpAndSettle();
    expect(find.text(t.checkingFile), findsNothing);
    expect(find.byType(Image), findsOneWidget);
  });

  testWidgets('a file can be taken off before it is sent', (tester) async {
    final chat = FakeChat();
    await Rig(chat).open(tester);
    await tester.tap(find.byTooltip(t.chatAttach));
    await tester.pumpAndSettle();
    await tester.tap(find.text(t.chooseDocument));
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip(t.chatFileRemove));
    await tester.pumpAndSettle();
    expect(find.text('synthetic-photo.jpg'), findsNothing);
    expect(tester.widget<IconButton>(find.widgetWithIcon(IconButton, Icons.send)).onPressed, isNull);
  });

  testWidgets("the doctor's files: a rejected one says so, a clean PDF opens in the share sheet", (tester) async {
    final chat = FakeChat()
      ..deliverFile(
        'synthetic-virus.pdf',
        ChatAttachmentContentTypeEnum.applicationSlashPdf,
        ChatAttachmentStatusEnum.rejected,
      )
      ..deliverFile(
        'synthetic-report.pdf',
        ChatAttachmentContentTypeEnum.applicationSlashPdf,
        ChatAttachmentStatusEnum.clean,
      );
    final rig = Rig(chat);
    await rig.open(tester);
    expect(find.text(t.chatFileRejected), findsOneWidget);
    await tester.tap(find.text('synthetic-report.pdf'));
    await tester.pumpAndSettle();
    expect(rig.files.shared.single.$1, 'synthetic-report.pdf');
  });

  testWidgets('when the session ends the chat hands over to sign-in and shows nothing of the conversation', (
    tester,
  ) async {
    final chat = FakeChat(history: 1);
    var ended = 0;
    await Rig(chat, onSessionEnded: () => ended++).open(tester);
    chat.end(ChatEnd.unauthenticated);
    await tester.pumpAndSettle();
    expect(ended, 1);
    expect(find.text(t.chatEndedSession), findsOneWidget);
    expect(find.byType(TextField), findsNothing);
    expect(find.text('Synthetic placeholder message 1'), findsNothing);
  });

  testWidgets('when access ends nothing of the conversation stays: no message, prompt, granted doctor or hours', (
    tester,
  ) async {
    final chat = FakeChat(history: 2);
    final granted = chat.invite();
    await chat.decide(granted.id, grant: true);
    chat.invite();
    var ended = 0;
    await Rig(chat, onSessionEnded: () => ended++).open(tester);
    expect(find.text(t.chatInviteTitle), findsOneWidget);
    expect(find.byTooltip(t.chatHours), findsOneWidget);

    chat.end(ChatEnd.notFound);
    await tester.pumpAndSettle();
    expect(find.text(t.chatEnded), findsOneWidget);
    expect(find.text('Synthetic placeholder message 2'), findsNothing);
    expect(find.text(t.chatInviteTitle), findsNothing);
    expect(find.widgetWithText(FilledButton, t.chatGrant), findsNothing);
    expect(find.widgetWithText(OutlinedButton, t.chatRevokeConfirm), findsNothing);
    expect(find.textContaining('Dee Example'), findsNothing);
    expect(find.byTooltip(t.chatHours), findsNothing);
    expect(find.byType(TextField), findsNothing);
    expect(ended, 0);
  });

  testWidgets('the granted doctors are read on their own, and again when the app comes back if that failed', (
    tester,
  ) async {
    final chat = GrantedOnceFailingChat();
    final granted = chat.invite();
    await chat.decide(granted.id, grant: true);
    chat.invite();
    await Rig(chat).open(tester);
    expect(find.text(t.chatInviteTitle), findsOneWidget);
    expect(find.text(t.chatGrantedTo('Dee Example', 'Synthetic clinic Ilidža')), findsNothing);

    for (final state in [AppLifecycleState.inactive, AppLifecycleState.hidden, AppLifecycleState.paused]) {
      tester.binding.handleAppLifecycleStateChanged(state);
    }
    for (final state in [AppLifecycleState.hidden, AppLifecycleState.inactive, AppLifecycleState.resumed]) {
      tester.binding.handleAppLifecycleStateChanged(state);
    }
    await tester.pumpAndSettle();
    expect(find.text(t.chatGrantedTo('Dee Example', 'Synthetic clinic Ilidža')), findsOneWidget);
    expect(find.text(t.chatInviteTitle), findsOneWidget);
  });

  testWidgets('a decision made elsewhere (consent_granted) updates the prompt and the granted doctors at once', (
    tester,
  ) async {
    final chat = FakeChat();
    final invitation = chat.invite();
    await Rig(chat).open(tester);
    expect(find.text(t.chatInviteTitle), findsOneWidget);
    await chat.decide(invitation.id, grant: true);
    await tester.pumpAndSettle();
    expect(find.text(t.chatInviteTitle), findsNothing);
    expect(find.text(t.chatGrantedTo('Dee Example', 'Synthetic clinic Ilidža')), findsOneWidget);
  });

  testWidgets('a decline made elsewhere (consent_declined) takes the prompt away at once', (tester) async {
    final chat = FakeChat();
    final invitation = chat.invite();
    await Rig(chat).open(tester);
    await chat.decide(invitation.id, grant: false);
    await tester.pumpAndSettle();
    expect(find.text(t.chatInviteTitle), findsNothing);
  });

  testWidgets('5000 messages: every page loads by scrolling up, and only a few dozen are ever built', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final chat = FakeChat(history: 5000);
    await Rig(chat).open(tester);
    var drags = 0;
    while (find.text('Synthetic placeholder message 1').evaluate().isEmpty && drags < 1000) {
      await tester.drag(find.byKey(const Key('chat-messages')), const Offset(0, 2000));
      await tester.pump();
      await tester.pump();
      drags++;
    }
    expect(find.text('Synthetic placeholder message 1'), findsOneWidget);
    final built = find.byType(ChatBubble, skipOffstage: false).evaluate().length;
    // Reported in docs/plans/chat.PROGRESS.md: the bubbles built with all 5000 loaded.
    // ignore: avoid_print
    print('chat virtualization: $built of 5000 message widgets built after $drags drags');
    expect(built, lessThan(60));
  });

  test('the consent prompt and the revocation carry the legal tag', () {
    final en = jsonDecode(File('lib/l10n/app_en.arb').readAsStringSync()) as Map<String, dynamic>;
    for (final key in [
      'chatInviteTitle',
      'chatInviteBody',
      'chatInviteWho',
      'chatInviteSees',
      'chatInviteChoice',
      'chatGrant',
      'chatDecline',
      'chatGranted',
      'chatDeclined',
      'chatRevoke',
      'chatRevokeBody',
      'chatRevokeConfirm',
      'chatRevoked',
    ]) {
      expect((en['@$key'] as Map)['description'] as String, startsWith('[legal]'), reason: key);
    }
  });
}

class _Empty extends FakeChat {
  @override
  Future<ConversationsPage> list({String? before}) async => const ConversationsPage([], null);
}
