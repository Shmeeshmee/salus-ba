import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' show CarePlan, CarePlanStatus, CarePlanStep, VisitMode;

import 'booking.dart';
import 'fhir.dart';
import 'loading.dart';
import 'messages.dart';
import 'ui.dart' as ui;
import 'visit_mode.dart';
import 'zones.dart';

/// The patient's booked treatment plans from every clinic, newest first (docs/clinic-cms, U6, package TP2): each with
/// its clinic, its state, its first day and its steps in order, each a visit on its day (its visit's, in the clinic's
/// zone, which a move changes) and of its type, with its service as the clinic's price list names it. A step's visit
/// is one of the patient's visits; moving or cancelling it is done there. The server never sends a draft. Each card
/// names its clinic from the directory, as the quotes screen does.
class CarePlansScreen extends StatefulWidget {
  const CarePlansScreen({super.key, required this.booking});

  final Booking booking;

  @override
  State<CarePlansScreen> createState() => _CarePlansScreenState();
}

typedef _Plans = ({List<CarePlan> all, Map<String, Organization> clinics});

class _CarePlansScreenState extends LoadingState<CarePlansScreen, _Plans> {
  @override
  Future<_Plans> load() async {
    final all = await widget.booking.carePlans();
    if (all.isEmpty) return (all: all, clinics: const <String, Organization>{});
    return (all: all, clinics: {for (final c in await widget.booking.fhir.listedClinics()) c.id: c});
  }

  @override
  Widget build(BuildContext context) => ui.Page(
    title: context.t.myCarePlans,
    children: body(
      (data) => [
        Text(data.all.isEmpty ? context.t.noCarePlans : context.t.carePlansIntro),
        for (final plan in data.all) _card(plan, data.clinics[plan.organizationId]),
      ],
    ),
  );

  Widget _card(CarePlan plan, Organization? clinic) {
    final t = context.t;
    final startsOn = plan.startsOn;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ui.Heading(plan.title),
            Text(clinic?.name ?? t.unlistedClinic),
            Text(_status(plan.status)),
            if (startsOn != null) ...[
              Text(t.carePlanStartsOn(Zone.longDate(startsOn, t.localeName))),
              for (final (i, step) in plan.steps.indexed) ...[
                Text(
                  t.carePlanStep(
                    i + 1,
                    _day(step, startsOn, clinic),
                    visitModeName(t, inPerson: step.visitMode == VisitMode.inPerson),
                  ),
                ),
                if (step.serviceName case final name? when name.isNotEmpty) Text(name),
              ],
            ],
          ],
        ),
      ),
    );
  }

  /// The step's day: its visit's, in the clinic's zone (a moved visit's new day), or the first day plus its days.
  String _day(CarePlanStep step, DateTime startsOn, Organization? clinic) {
    final locale = context.t.localeName;
    final visit = step.visitStart;
    if (visit != null) {
      final zone = clinic == null || clinic.timeZone.isEmpty ? 'Europe/Sarajevo' : clinic.timeZone;
      return Zone(zone).date(visit, locale);
    }
    return Zone.longDate(DateTime.utc(startsOn.year, startsOn.month, startsOn.day + step.dayOffset), locale);
  }

  String _status(CarePlanStatus status) {
    final t = context.t;
    if (status == CarePlanStatus.completed) return t.carePlanCompleted;
    if (status == CarePlanStatus.revoked) return t.carePlanRevoked;
    return t.carePlanActive;
  }
}
