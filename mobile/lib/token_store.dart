import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Where the session token is kept between launches. Tests fake it.
abstract interface class TokenStore {
  Future<String?> read();
  Future<void> write(String token);
  Future<void> delete();
}

/// The token in the platform's secure storage: the Keychain on iOS, the Keystore on Android. Never in shared
/// preferences.
class SecureTokenStore implements TokenStore {
  const SecureTokenStore(this._storage);

  static const _key = 'session_token';

  final FlutterSecureStorage _storage;

  @override
  Future<String?> read() => _storage.read(key: _key);

  @override
  Future<void> write(String token) => _storage.write(key: _key, value: token);

  @override
  Future<void> delete() => _storage.delete(key: _key);
}
