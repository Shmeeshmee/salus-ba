//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class BookAppointmentRequest {
  /// Returns a new [BookAppointmentRequest] instance.
  BookAppointmentRequest({
    required this.slotId,
  });

  /// A free slot, as Slot?schedule=…&status=free gives it.
  String slotId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is BookAppointmentRequest && other.slotId == slotId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (slotId.hashCode);

  @override
  String toString() => 'BookAppointmentRequest[slotId=$slotId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'slotId'] = this.slotId;
    return json;
  }

  /// Returns a new [BookAppointmentRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static BookAppointmentRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'slotId'),
            'Required key "BookAppointmentRequest[slotId]" is missing from JSON.');
        assert(json[r'slotId'] != null,
            'Required key "BookAppointmentRequest[slotId]" has a null value in JSON.');
        return true;
      }());

      return BookAppointmentRequest(
        slotId: mapValueOfType<String>(json, r'slotId')!,
      );
    }
    return null;
  }

  static List<BookAppointmentRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <BookAppointmentRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = BookAppointmentRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, BookAppointmentRequest> mapFromJson(dynamic json) {
    final map = <String, BookAppointmentRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = BookAppointmentRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of BookAppointmentRequest-objects as value to a dart map
  static Map<String, List<BookAppointmentRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<BookAppointmentRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = BookAppointmentRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'slotId',
  };
}
