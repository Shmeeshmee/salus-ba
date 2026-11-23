//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Quote {
  /// Returns a new [Quote] instance.
  Quote({
    required this.id,
    required this.organizationId,
    required this.patientId,
    this.appointmentId,
    required this.issuedAt,
    required this.validUntil,
    required this.state,
    this.note,
    required this.currency,
    required this.totalMinor,
    this.lines = const [],
  });

  String id;

  String organizationId;

  String patientId;

  /// The visit it was written at; absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? appointmentId;

  DateTime issuedAt;

  DateTime validUntil;

  QuoteState state;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? note;

  String currency;

  int totalMinor;

  List<ReceiptLine> lines;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Quote &&
          other.id == id &&
          other.organizationId == organizationId &&
          other.patientId == patientId &&
          other.appointmentId == appointmentId &&
          other.issuedAt == issuedAt &&
          other.validUntil == validUntil &&
          other.state == state &&
          other.note == note &&
          other.currency == currency &&
          other.totalMinor == totalMinor &&
          _deepEquality.equals(other.lines, lines);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (organizationId.hashCode) +
      (patientId.hashCode) +
      (appointmentId == null ? 0 : appointmentId!.hashCode) +
      (issuedAt.hashCode) +
      (validUntil.hashCode) +
      (state.hashCode) +
      (note == null ? 0 : note!.hashCode) +
      (currency.hashCode) +
      (totalMinor.hashCode) +
      (lines.hashCode);

  @override
  String toString() =>
      'Quote[id=$id, organizationId=$organizationId, patientId=$patientId, appointmentId=$appointmentId, issuedAt=$issuedAt, validUntil=$validUntil, state=$state, note=$note, currency=$currency, totalMinor=$totalMinor, lines=$lines]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'organizationId'] = this.organizationId;
    json[r'patientId'] = this.patientId;
    if (this.appointmentId != null) {
      json[r'appointmentId'] = this.appointmentId;
    } else {
      json[r'appointmentId'] = null;
    }
    json[r'issuedAt'] = this.issuedAt.toUtc().toIso8601String();
    json[r'validUntil'] = this.validUntil.toUtc().toIso8601String();
    json[r'state'] = this.state;
    if (this.note != null) {
      json[r'note'] = this.note;
    } else {
      json[r'note'] = null;
    }
    json[r'currency'] = this.currency;
    json[r'totalMinor'] = this.totalMinor;
    json[r'lines'] = this.lines;
    return json;
  }

  /// Returns a new [Quote] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Quote? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Quote[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Quote[id]" has a null value in JSON.');
        assert(json.containsKey(r'organizationId'),
            'Required key "Quote[organizationId]" is missing from JSON.');
        assert(json[r'organizationId'] != null,
            'Required key "Quote[organizationId]" has a null value in JSON.');
        assert(json.containsKey(r'patientId'),
            'Required key "Quote[patientId]" is missing from JSON.');
        assert(json[r'patientId'] != null,
            'Required key "Quote[patientId]" has a null value in JSON.');
        assert(json.containsKey(r'issuedAt'),
            'Required key "Quote[issuedAt]" is missing from JSON.');
        assert(json[r'issuedAt'] != null,
            'Required key "Quote[issuedAt]" has a null value in JSON.');
        assert(json.containsKey(r'validUntil'),
            'Required key "Quote[validUntil]" is missing from JSON.');
        assert(json[r'validUntil'] != null,
            'Required key "Quote[validUntil]" has a null value in JSON.');
        assert(json.containsKey(r'state'),
            'Required key "Quote[state]" is missing from JSON.');
        assert(json[r'state'] != null,
            'Required key "Quote[state]" has a null value in JSON.');
        assert(json.containsKey(r'currency'),
            'Required key "Quote[currency]" is missing from JSON.');
        assert(json[r'currency'] != null,
            'Required key "Quote[currency]" has a null value in JSON.');
        assert(json.containsKey(r'totalMinor'),
            'Required key "Quote[totalMinor]" is missing from JSON.');
        assert(json[r'totalMinor'] != null,
            'Required key "Quote[totalMinor]" has a null value in JSON.');
        assert(json.containsKey(r'lines'),
            'Required key "Quote[lines]" is missing from JSON.');
        assert(json[r'lines'] != null,
            'Required key "Quote[lines]" has a null value in JSON.');
        return true;
      }());

      return Quote(
        id: mapValueOfType<String>(json, r'id')!,
        organizationId: mapValueOfType<String>(json, r'organizationId')!,
        patientId: mapValueOfType<String>(json, r'patientId')!,
        appointmentId: mapValueOfType<String>(json, r'appointmentId'),
        issuedAt: mapDateTime(json, r'issuedAt', r'')!,
        validUntil: mapDateTime(json, r'validUntil', r'')!,
        state: QuoteState.fromJson(json[r'state'])!,
        note: mapValueOfType<String>(json, r'note'),
        currency: mapValueOfType<String>(json, r'currency')!,
        totalMinor: mapValueOfType<int>(json, r'totalMinor')!,
        lines: ReceiptLine.listFromJson(json[r'lines']),
      );
    }
    return null;
  }

  static List<Quote> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Quote>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Quote.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Quote> mapFromJson(dynamic json) {
    final map = <String, Quote>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Quote.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Quote-objects as value to a dart map
  static Map<String, List<Quote>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Quote>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Quote.listFromJson(
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
    'organizationId',
    'patientId',
    'issuedAt',
    'validUntil',
    'state',
    'currency',
    'totalMinor',
    'lines',
  };
}
