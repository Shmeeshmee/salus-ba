//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DeskReceipt {
  /// Returns a new [DeskReceipt] instance.
  DeskReceipt({
    required this.id,
    required this.number,
    required this.reversed,
  });

  String id;

  int number;

  bool reversed;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DeskReceipt &&
          other.id == id &&
          other.number == number &&
          other.reversed == reversed;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) + (number.hashCode) + (reversed.hashCode);

  @override
  String toString() =>
      'DeskReceipt[id=$id, number=$number, reversed=$reversed]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'number'] = this.number;
    json[r'reversed'] = this.reversed;
    return json;
  }

  /// Returns a new [DeskReceipt] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DeskReceipt? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "DeskReceipt[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "DeskReceipt[id]" has a null value in JSON.');
        assert(json.containsKey(r'number'),
            'Required key "DeskReceipt[number]" is missing from JSON.');
        assert(json[r'number'] != null,
            'Required key "DeskReceipt[number]" has a null value in JSON.');
        assert(json.containsKey(r'reversed'),
            'Required key "DeskReceipt[reversed]" is missing from JSON.');
        assert(json[r'reversed'] != null,
            'Required key "DeskReceipt[reversed]" has a null value in JSON.');
        return true;
      }());

      return DeskReceipt(
        id: mapValueOfType<String>(json, r'id')!,
        number: mapValueOfType<int>(json, r'number')!,
        reversed: mapValueOfType<bool>(json, r'reversed')!,
      );
    }
    return null;
  }

  static List<DeskReceipt> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DeskReceipt>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DeskReceipt.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DeskReceipt> mapFromJson(dynamic json) {
    final map = <String, DeskReceipt>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DeskReceipt.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DeskReceipt-objects as value to a dart map
  static Map<String, List<DeskReceipt>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DeskReceipt>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DeskReceipt.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'id',
    'number',
    'reversed',
  };
}
