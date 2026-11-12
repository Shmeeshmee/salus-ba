import 'package:flutter_test/flutter_test.dart';
import 'package:salus/app.dart';
import 'package:salus/l10n/app_localizations.dart';
import 'package:salus/l10n/language.dart';

import 'fake_chat.dart';
import 'support.dart';

// The home screen's way to the conversations, with the app wired as main.dart wires it.

final t = lookupAppLocalizations(localeOf('en'));

void main() {
  testWidgets('Messages on the home screen opens the conversations and a chat', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    answerDirectory(h.api);
    final chat = FakeChat(history: 2);
    await tester.pumpWidget(
      SalusApp(
        session: h.session,
        booking: h.booking,
        chats: chat,
        now: () => h.now,
        media: h.media,
        files: h.files,
        languages: h.languages,
        privacyNoticeUrl: privacyNoticeUrl,
        openUrl: (_) async => true,
      ),
    );
    await tester.pumpAndSettle();

    await tapOutlined(tester, t.chatsTitle);
    expect(find.text(t.chatDoctorUnknown), findsOneWidget);
    await tapText(tester, t.chatDoctorUnknown);
    expect(find.text('Synthetic placeholder message 2'), findsOneWidget);
  });

  testWidgets('without the chat wired in, the home screen has no Messages', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    await h.pump(tester);
    expect(find.text(t.chatsTitle), findsNothing);
  });
}
