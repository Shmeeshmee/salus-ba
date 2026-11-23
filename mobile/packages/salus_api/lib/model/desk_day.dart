//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DeskDay {
  /// Returns a new [DeskDay] instance.
  DeskDay({
    required this.date,
    this.visits = const [],
  });

  DateTime date;

  /// By practitioner, then start.
  List<DeskVisit> visits;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DeskDay &&
          other.date == date &&
          _deepEquality.equals(other.visits, visits);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (date.hashCode) + (visits.hashCode);

  @override
  String toString() => 'DeskDay[date=$date, visits=$visits]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'date'] = _dateFormatter.format(this.date);
    json[r'visits'] = this.visits;
    return json;
  }

  /// Returns a new [DeskDay] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DeskDay? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'date'),
            'Required key "DeskDay[date]" is missing from JSON.');
        assert(json[r'date'] != null,
            'Required key "DeskDay[date]" has a null value in JSON.');
        assert(json.containsKey(r'visits'),
            'Required key "DeskDay[visits]" is missing from JSON.');
        assert(json[r'visits'] != null,
            'Required key "DeskDay[visits]" has a null value in JSON.');
        return true;
      }());

      return DeskDay(
        date: mapDateTime(json, r'date', r'')!,
        visits: DeskVisit.listFromJson(json[r'visits']),
      );
    }
    return null;
  }

  static List<DeskDay> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DeskDay>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DeskDay.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DeskDay> mapFromJson(dynamic json) {
    final map = <String, DeskDay>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DeskDay.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DeskDay-objects as value to a dart map
  static Map<String, List<DeskDay>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DeskDay>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DeskDay.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'date',
    'visits',
  };
}
