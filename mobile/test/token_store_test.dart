import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salus/token_store.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('keeps the token in secure storage and deletes it', () async {
    FlutterSecureStorage.setMockInitialValues({});
    const store = SecureTokenStore(FlutterSecureStorage());

    expect(await store.read(), isNull);
    await store.write('synthetic-session-token');
    expect(await store.read(), 'synthetic-session-token');
    expect(await const FlutterSecureStorage().readAll(), {'session_token': 'synthetic-session-token'});

    await store.delete();
    expect(await store.read(), isNull);
  });
}
