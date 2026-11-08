//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DraftEvent {
  /// Returns a new [DraftEvent] instance.
  DraftEvent({
    required this.draftId,
    required this.encounterId,
    required this.templateId,
    required this.status,
    this.fields = const [],
  });

  String draftId;

  String encounterId;

  String templateId;

  DraftEventStatusEnum status;

  /// In the template's order.
  List<DraftField> fields;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DraftEvent &&
          other.draftId == draftId &&
          other.encounterId == encounterId &&
          other.templateId == templateId &&
          other.status == status &&
          _deepEquality.equals(other.fields, fields);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (draftId.hashCode) +
      (encounterId.hashCode) +
      (templateId.hashCode) +
      (status.hashCode) +
      (fields.hashCode);

  @override
  String toString() =>
      'DraftEvent[draftId=$draftId, encounterId=$encounterId, templateId=$templateId, status=$status, fields=$fields]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'draft_id'] = this.draftId;
    json[r'encounter_id'] = this.encounterId;
    json[r'template_id'] = this.templateId;
    json[r'status'] = this.status;
    json[r'fields'] = this.fields;
    return json;
  }

  /// Returns a new [DraftEvent] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DraftEvent? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'draft_id'),
            'Required key "DraftEvent[draft_id]" is missing from JSON.');
        assert(json[r'draft_id'] != null,
            'Required key "DraftEvent[draft_id]" has a null value in JSON.');
        assert(json.containsKey(r'encounter_id'),
            'Required key "DraftEvent[encounter_id]" is missing from JSON.');
        assert(json[r'encounter_id'] != null,
            'Required key "DraftEvent[encounter_id]" has a null value in JSON.');
        assert(json.containsKey(r'template_id'),
            'Required key "DraftEvent[template_id]" is missing from JSON.');
        assert(json[r'template_id'] != null,
            'Required key "DraftEvent[template_id]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "DraftEvent[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "DraftEvent[status]" has a null value in JSON.');
        assert(json.containsKey(r'fields'),
            'Required key "DraftEvent[fields]" is missing from JSON.');
        assert(json[r'fields'] != null,
            'Required key "DraftEvent[fields]" has a null value in JSON.');
        return true;
      }());

      return DraftEvent(
        draftId: mapValueOfType<String>(json, r'draft_id')!,
        encounterId: mapValueOfType<String>(json, r'encounter_id')!,
        templateId: mapValueOfType<String>(json, r'template_id')!,
        status: DraftEventStatusEnum.fromJson(json[r'status'])!,
        fields: DraftField.listFromJson(json[r'fields']),
      );
    }
    return null;
  }

  static List<DraftEvent> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DraftEvent>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DraftEvent.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DraftEvent> mapFromJson(dynamic json) {
    final map = <String, DraftEvent>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DraftEvent.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DraftEvent-objects as value to a dart map
  static Map<String, List<DraftEvent>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DraftEvent>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DraftEvent.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'draft_id',
    'encounter_id',
    'template_id',
    'status',
    'fields',
  };
}

enum DraftEventStatusEnum {
  preliminary._(r'preliminary'),
  final_._(r'final'),
  enteredInError._(r'entered-in-error'),
  ;

  /// Instantiate a new enum with the provided value.
  const DraftEventStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [DraftEventStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static DraftEventStatusEnum? fromJson(dynamic value) =>
      DraftEventStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [DraftEventStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<DraftEventStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DraftEventStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DraftEventStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [DraftEventStatusEnum] to String,
/// and [decode] dynamic data back to [DraftEventStatusEnum].
class DraftEventStatusEnumTypeTransformer {
  factory DraftEventStatusEnumTypeTransformer() =>
      _instance ??= const DraftEventStatusEnumTypeTransformer._();

  const DraftEventStatusEnumTypeTransformer._();

  String encode(DraftEventStatusEnum data) => data._value;

  /// Returns the instance of [DraftEventStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  DraftEventStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is DraftEventStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'preliminary':
          return DraftEventStatusEnum.preliminary;
        case r'final':
          return DraftEventStatusEnum.final_;
        case r'entered-in-error':
          return DraftEventStatusEnum.enteredInError;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static DraftEventStatusEnumTypeTransformer? _instance;
}
