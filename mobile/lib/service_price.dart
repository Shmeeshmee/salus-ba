import 'package:intl/intl.dart';

/// A price in the currency's minor unit as the UI language writes the amount, then the currency's code, with Latin
/// digits in every language (docs/plans/clinic-cms.md, package B): 5000 BAM is "50.00 BAM" in English and
/// "50,00 BAM" in Bosnian.
String priceText(int priceMinor, String currency, String localeName) =>
    '${NumberFormat.decimalPatternDigits(locale: localeName, decimalDigits: 2).format(priceMinor / 100)} $currency';
