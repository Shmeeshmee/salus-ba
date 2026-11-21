import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:salus_api/api.dart' as api;

import 'messages.dart';
import 'ui.dart' as ui;

/// A price in the currency's minor unit as the UI language writes the amount, then the currency's code, with Latin
/// digits in every language (docs/plans/clinic-cms.md, package B): 5000 BAM is "50.00 BAM" in English and
/// "50,00 BAM" in Bosnian.
String priceText(int priceMinor, String currency, String localeName) =>
    '${NumberFormat.decimalPatternDigits(locale: localeName, decimalDigits: 2).format(priceMinor / 100)} $currency';

/// The services a clinic offers for a visit with their prices, shown at booking. The patient pays at the clinic:
/// nothing here is charged in the app. Shows nothing for a clinic with no price list.
class ServicePrices extends StatelessWidget {
  const ServicePrices({super.key, required this.services});

  final List<api.OfferedService> services;

  @override
  Widget build(BuildContext context) {
    if (services.isEmpty) return const SizedBox.shrink();
    final t = context.t;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ui.Heading(t.pricesHeading),
        for (final s in services)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 4),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(child: Text(s.name)),
                const SizedBox(width: 16),
                Text(priceText(s.priceMinor, s.currency, t.localeName)),
              ],
            ),
          ),
        Text(t.pricesPaidAtClinic, style: Theme.of(context).textTheme.bodySmall),
      ],
    );
  }
}
