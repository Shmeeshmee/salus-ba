import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' as api;

import 'messages.dart';
import 'service_price.dart';
import 'ui.dart' as ui;

/// The services a clinic offers for a visit with their prices, shown at booking, one of which the patient may pick
/// (docs/plans/clinic-cms.md, package E): the booking names it and the clinic sees it. Leaving it to the clinic is the
/// first choice and the default. The patient pays at the clinic: nothing here is charged in the app. Shows nothing for
/// a clinic with no price list.
class ServicePicker extends StatelessWidget {
  const ServicePicker({super.key, required this.services, required this.selected, required this.onChanged});

  final List<api.OfferedService> services;

  /// The id of the service picked, null for none.
  final String? selected;
  final ValueChanged<String?>? onChanged;

  @override
  Widget build(BuildContext context) {
    if (services.isEmpty) return const SizedBox.shrink();
    final t = context.t;
    final change = onChanged;
    Widget choice(String? id, Widget title, {Widget? price}) => ListTile(
      contentPadding: EdgeInsets.zero,
      selected: selected == id,
      leading: Icon(selected == id ? Icons.radio_button_checked : Icons.radio_button_unchecked),
      title: title,
      trailing: price,
      onTap: change == null ? null : () => change(id),
    );
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ui.Heading(t.pricesHeading),
        choice(null, Text(t.serviceAtClinic)),
        for (final s in services) choice(s.id, Text(s.name), price: Text(priceText(s.priceMinor, s.currency, t.localeName))),
        Text(t.pricesPaidAtClinic, style: Theme.of(context).textTheme.bodySmall),
      ],
    );
  }
}
