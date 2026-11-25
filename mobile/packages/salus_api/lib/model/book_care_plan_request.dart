//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class BookCarePlanRequest {
  /// Returns a new [BookCarePlanRequest] instance.
  BookCarePlanRequest({
    required this.startsOn,
    this.slotIds = const [],
  });

  /// Day 0 of the steps.
  DateTime startsOn;

  /// One free slot per step, in the steps' order.
  List<String> slotIds;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is BookCarePlanRequest &&
          other.startsOn == startsOn &&
          _deepEquality.equals(other.slotIds, slotIds);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (startsOn.hashCode) + (slotIds.hashCode);

  @override
  String toString() =>
      'BookCarePlanRequest[startsOn=$startsOn, slotIds=$slotIds]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'startsOn'] = _dateFormatter.format(this.startsOn);
    json[r'slotIds'] = this.slotIds;
    return json;
  }

  /// Returns a new [BookCarePlanRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static BookCarePlanRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'startsOn'),
            'Required key "BookCarePlanRequest[startsOn]" is missing from JSON.');
        assert(json[r'startsOn'] != null,
            'Required key "BookCarePlanRequest[startsOn]" has a null value in JSON.');
        assert(json.containsKey(r'slotIds'),
            'Required key "BookCarePlanRequest[slotIds]" is missing from JSON.');
        assert(json[r'slotIds'] != null,
            'Required key "BookCarePlanRequest[slotIds]" has a null value in JSON.');
        return true;
      }());

      return BookCarePlanRequest(
        startsOn: mapDateTime(json, r'startsOn', r'')!,
        slotIds: json[r'slotIds'] is Iterable
            ? (json[r'slotIds'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
      );
    }
    return null;
  }

  static List<BookCarePlanRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <BookCarePlanRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = BookCarePlanRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, BookCarePlanRequest> mapFromJson(dynamic json) {
    final map = <String, BookCarePlanRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = BookCarePlanRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of BookCarePlanRequest-objects as value to a dart map
  static Map<String, List<BookCarePlanRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<BookCarePlanRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = BookCarePlanRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'startsOn',
    'slotIds',
  };
}
