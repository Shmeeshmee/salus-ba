//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SendQuoteRequest {
  /// Returns a new [SendQuoteRequest] instance.
  SendQuoteRequest({
    required this.patientId,
    this.appointmentId,
    this.note,
    this.lines = const [],
  });

  String patientId;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? appointmentId;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? note;

  List<ReceiptPick> lines;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SendQuoteRequest &&
          other.patientId == patientId &&
          other.appointmentId == appointmentId &&
          other.note == note &&
          _deepEquality.equals(other.lines, lines);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (patientId.hashCode) +
      (appointmentId == null ? 0 : appointmentId!.hashCode) +
      (note == null ? 0 : note!.hashCode) +
      (lines.hashCode);

  @override
  String toString() =>
      'SendQuoteRequest[patientId=$patientId, appointmentId=$appointmentId, note=$note, lines=$lines]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'patientId'] = this.patientId;
    if (this.appointmentId != null) {
      json[r'appointmentId'] = this.appointmentId;
    } else {
      json[r'appointmentId'] = null;
    }
    if (this.note != null) {
      json[r'note'] = this.note;
    } else {
      json[r'note'] = null;
    }
    json[r'lines'] = this.lines;
    return json;
  }

  /// Returns a new [SendQuoteRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SendQuoteRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'patientId'),
            'Required key "SendQuoteRequest[patientId]" is missing from JSON.');
        assert(json[r'patientId'] != null,
            'Required key "SendQuoteRequest[patientId]" has a null value in JSON.');
        assert(json.containsKey(r'lines'),
            'Required key "SendQuoteRequest[lines]" is missing from JSON.');
        assert(json[r'lines'] != null,
            'Required key "SendQuoteRequest[lines]" has a null value in JSON.');
        return true;
      }());

      return SendQuoteRequest(
        patientId: mapValueOfType<String>(json, r'patientId')!,
        appointmentId: mapValueOfType<String>(json, r'appointmentId'),
        note: mapValueOfType<String>(json, r'note'),
        lines: ReceiptPick.listFromJson(json[r'lines']),
      );
    }
    return null;
  }

  static List<SendQuoteRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SendQuoteRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SendQuoteRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SendQuoteRequest> mapFromJson(dynamic json) {
    final map = <String, SendQuoteRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SendQuoteRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SendQuoteRequest-objects as value to a dart map
  static Map<String, List<SendQuoteRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SendQuoteRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SendQuoteRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'patientId',
    'lines',
  };
}
