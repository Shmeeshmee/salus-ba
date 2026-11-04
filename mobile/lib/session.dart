import 'dart:convert';
import 'dart:io';

import 'package:salus_api/api.dart';

import 'token_store.dart';

/// A failed call: the API's error code, or none when the API could not be reached ([status] 0). Nothing from
/// the response but the fixed code is kept, so nothing else can reach the screen.
class AuthFailure implements Exception {
  const AuthFailure({this.status = 0, this.code, this.retryAfter});

  final int status;
  final String? code;
  final Duration? retryAfter;

  bool get unreachable => status == 0;

  factory AuthFailure.fromApi(ApiException e) {
    if (e.innerException != null) return const AuthFailure();
    return AuthFailure(status: e.code, code: _errorCode(e.message));
  }

  /// A refused login, which also says when to try again (`Retry-After`, in seconds).
  factory AuthFailure.fromLogin(int status, String body, String? retryAfter) {
    final seconds = int.tryParse(retryAfter ?? '');
    return AuthFailure(
      status: status,
      code: _errorCode(body),
      retryAfter: seconds == null || seconds <= 0 ? null : Duration(seconds: seconds),
    );
  }

  static String? _errorCode(String? body) {
    try {
      final json = jsonDecode(body ?? '');
      return json is Map && json['error'] is String ? json['error'] as String : null;
    } on FormatException {
      return null;
    }
  }
}

/// Where a sign-in step leads.
sealed class SignIn {}

/// The patient is signed in and the token is stored.
final class SignedIn extends SignIn {
  SignedIn(this.account);
  final Account account;
}

/// The account has MFA: the code goes with [challenge] to [Session.verify].
final class CodeNeeded extends SignIn {
  CodeNeeded(this.challenge);
  final String challenge;
}

/// A clinic staff account: the app is for patients, so no session is kept.
final class StaffAccount extends SignIn {}

/// The patient's session: registration, sign-in with the TOTP step, the stored token and sign-out, through the
/// generated client. Requests carry `Authorization: Bearer` and never `X-Salus-CSRF`, so the API returns the
/// token in the body instead of a cookie.
class Session {
  Session({required this._api, required this._bearer, required this._store});

  final AuthApi _api;
  final HttpBearerAuth _bearer;
  final TokenStore _store;

  /// The account of the stored token, or null when there is none or the API no longer accepts it.
  Future<Account?> restore() async {
    final token = await _store.read();
    if (token == null) return null;
    _bearer.accessToken = token;
    try {
      return await _me();
    } on AuthFailure catch (f) {
      if (f.status != HttpStatus.unauthorized) rethrow;
      await _forget();
      return null;
    }
  }

  Future<void> register(String email, String password) =>
      _call(() => _api.register(RegisterRequest(email: email, password: password)));

  Future<SignIn> login(String email, String password) async {
    // The plain call drops the headers; a 429 needs Retry-After.
    final response = await _call(() => _api.loginWithHttpInfo(LoginRequest(email: email, password: password)));
    if (response.statusCode >= HttpStatus.badRequest) {
      throw AuthFailure.fromLogin(response.statusCode, response.body, response.headers['retry-after']);
    }
    final step = LoginResponse.fromJson(jsonDecode(response.body));
    return switch (step?.next) {
      LoginResponseNextEnum.done => _open(step!.token),
      LoginResponseNextEnum.verify => CodeNeeded(step!.challenge ?? ''),
      LoginResponseNextEnum.enrol => StaffAccount(),
      _ => throw const AuthFailure(status: HttpStatus.badGateway),
    };
  }

  Future<SignIn> verify(String challenge, String code) async {
    final step = await _call(() => _api.verifyMfa(VerifyRequest(challenge: challenge, code: code)));
    return _open(step?.token);
  }

  /// Ends the session. A session the API already ended counts as signed out; any other failure keeps it.
  Future<void> signOut() async {
    try {
      await _call(_api.logout);
    } on AuthFailure catch (f) {
      if (f.status != HttpStatus.unauthorized) rethrow;
    }
    await _forget();
  }

  /// Keeps [token] when it belongs to a patient; a staff session is ended at once and not stored.
  Future<SignIn> _open(String? token) async {
    if (token == null || token.isEmpty) throw const AuthFailure(status: HttpStatus.badGateway);
    _bearer.accessToken = token;
    final Account account;
    try {
      account = await _me();
    } on AuthFailure {
      _bearer.accessToken = '';
      rethrow;
    }
    if (account.role == AccountRoleEnum.patient) {
      await _store.write(token);
      return SignedIn(account);
    }
    try {
      await _call(_api.logout);
    } on AuthFailure {
      // Best effort: the token is dropped either way and expires on its own.
    }
    _bearer.accessToken = '';
    return StaffAccount();
  }

  Future<Account> _me() async {
    final account = await _call(_api.getMe);
    if (account == null) throw const AuthFailure(status: HttpStatus.badGateway);
    return account;
  }

  Future<void> _forget() async {
    _bearer.accessToken = '';
    await _store.delete();
  }

  static Future<T> _call<T>(Future<T> Function() call) async {
    try {
      return await call();
    } on ApiException catch (e) {
      throw AuthFailure.fromApi(e);
    }
  }
}
