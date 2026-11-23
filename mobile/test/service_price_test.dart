import 'package:flutter_test/flutter_test.dart';
import 'package:salus/l10n/app_localizations.dart';
import 'package:salus/service_price.dart';

void main() {
  test('a price is the amount as the language writes it, then the currency', () {
    expect(priceText(5000, 'BAM', 'en'), '50.00 BAM');
    expect(priceText(123450, 'BAM', 'en'), '1,234.50 BAM');
    expect(priceText(4990, 'BAM', 'bs'), '49,90 BAM');
    expect(priceText(0, 'BAM', 'de'), '0,00 BAM');
  });

  test('a price has Latin digits in every language of the app', () {
    for (final locale in AppLocalizations.supportedLocales) {
      final name = locale.scriptCode == null ? locale.languageCode : '${locale.languageCode}_${locale.scriptCode}';
      expect(priceText(123456, 'BAM', name), matches(RegExp(r'^1.?234.56 BAM$')), reason: name);
    }
  });
}
