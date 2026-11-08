import 'package:flutter/widgets.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// The app's languages as BCP 47 tags, the values of the API's `language` (docs/i18n/SPEC.md §3 "Tags").
const languages = ['bs', 'hr', 'sr-Latn', 'sr-Cyrl', 'en', 'de', 'ar', 'tr', 'it', 'fr'];

/// The language of a device that lists none the app has: a tourist's phone is more likely to read English.
const fallbackLanguage = 'en';

/// The bundled font for Arabic letters (pubspec.yaml), a fallback after the theme's own font.
const arabicFont = 'NotoSansArabic';

/// The first of the device's languages, in the device's order, that the app has (SPEC §3 "Matching").
String matchLanguage(Iterable<Locale> device) {
  for (final locale in device) {
    final tag = _match(locale);
    if (tag != null) return tag;
  }
  return fallbackLanguage;
}

String? _match(Locale locale) => switch (locale.languageCode) {
  'bs' || 'hr' || 'en' || 'de' || 'ar' || 'tr' || 'it' || 'fr' => locale.languageCode,
  'sh' || 'cnr' => 'sr-Latn',
  // CLDR's default script for Serbian is Cyrillic; Montenegro's is Latin.
  'sr' => switch ((locale.scriptCode, locale.countryCode)) {
    ('Latn', _) || (null, 'ME') => 'sr-Latn',
    _ => 'sr-Cyrl',
  },
  _ => null,
};

/// The Flutter locale of one of [languages].
Locale localeOf(String tag) => switch (tag) {
  'sr-Latn' => const Locale.fromSubtags(languageCode: 'sr', scriptCode: 'Latn'),
  'sr-Cyrl' => const Locale.fromSubtags(languageCode: 'sr', scriptCode: 'Cyrl'),
  _ => Locale(tag),
};

/// Where the language the patient chose is kept on the device, so it holds before sign-in (SPEC §3 "Picking and
/// saving"). Tests fake it.
abstract interface class LanguageStore {
  Future<String?> read();
  Future<void> write(String tag);
  Future<void> delete();
}

/// The chosen language in the platform's secure storage, beside the session token.
class SecureLanguageStore implements LanguageStore {
  const SecureLanguageStore(this._storage);

  static const _key = 'language';

  final FlutterSecureStorage _storage;

  @override
  Future<String?> read() => _storage.read(key: _key);

  @override
  Future<void> write(String tag) => _storage.write(key: _key, value: tag);

  @override
  Future<void> delete() => _storage.delete(key: _key);
}
