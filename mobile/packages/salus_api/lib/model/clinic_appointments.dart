//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ClinicAppointments {
  /// Returns a new [ClinicAppointments] instance.
  ClinicAppointments({
    this.series = const [],
    this.heatmap = const [],
    this.practitioners = const [],
  });

  List<ClinicAppointmentsSeriesInner> series;

  List<ClinicAppointmentsHeatmapInner> heatmap;

  /// For the clinic's admin only.
  List<ClinicAppointmentsPractitionersInner> practitioners;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ClinicAppointments &&
          _deepEquality.equals(other.series, series) &&
          _deepEquality.equals(other.heatmap, heatmap) &&
          _deepEquality.equals(other.practitioners, practitioners);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (series.hashCode) + (heatmap.hashCode) + (practitioners.hashCode);

  @override
  String toString() =>
      'ClinicAppointments[series=$series, heatmap=$heatmap, practitioners=$practitioners]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'series'] = this.series;
    json[r'heatmap'] = this.heatmap;
    json[r'practitioners'] = this.practitioners;
    return json;
  }

  /// Returns a new [ClinicAppointments] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ClinicAppointments? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'series'),
            'Required key "ClinicAppointments[series]" is missing from JSON.');
        assert(json[r'series'] != null,
            'Required key "ClinicAppointments[series]" has a null value in JSON.');
        assert(json.containsKey(r'heatmap'),
            'Required key "ClinicAppointments[heatmap]" is missing from JSON.');
        assert(json[r'heatmap'] != null,
            'Required key "ClinicAppointments[heatmap]" has a null value in JSON.');
        return true;
      }());

      return ClinicAppointments(
        series: ClinicAppointmentsSeriesInner.listFromJson(json[r'series']),
        heatmap: ClinicAppointmentsHeatmapInner.listFromJson(json[r'heatmap']),
        practitioners: ClinicAppointmentsPractitionersInner.listFromJson(
            json[r'practitioners']),
      );
    }
    return null;
  }

  static List<ClinicAppointments> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ClinicAppointments>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ClinicAppointments.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ClinicAppointments> mapFromJson(dynamic json) {
    final map = <String, ClinicAppointments>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ClinicAppointments.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ClinicAppointments-objects as value to a dart map
  static Map<String, List<ClinicAppointments>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ClinicAppointments>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ClinicAppointments.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'series',
    'heatmap',
  };
}
