//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SuggestionEvent {
  /// Returns a new [SuggestionEvent] instance.
  SuggestionEvent({
    required this.id,
    required this.encounterId,
    required this.type,
    required this.text,
    required this.confidence,
    this.sourceSegmentIds = const [],
    required this.status,
  });

  String id;

  String encounterId;

  SuggestionEventTypeEnum type;

  String text;

  /// Minimum value: 0
  /// Maximum value: 1
  num confidence;

  List<String> sourceSegmentIds;

  SuggestionEventStatusEnum status;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SuggestionEvent &&
          other.id == id &&
          other.encounterId == encounterId &&
          other.type == type &&
          other.text == text &&
          other.confidence == confidence &&
          _deepEquality.equals(other.sourceSegmentIds, sourceSegmentIds) &&
          other.status == status;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (encounterId.hashCode) +
      (type.hashCode) +
      (text.hashCode) +
      (confidence.hashCode) +
      (sourceSegmentIds.hashCode) +
      (status.hashCode);

  @override
  String toString() =>
      'SuggestionEvent[id=$id, encounterId=$encounterId, type=$type, text=$text, confidence=$confidence, sourceSegmentIds=$sourceSegmentIds, status=$status]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'encounter_id'] = this.encounterId;
    json[r'type'] = this.type;
    json[r'text'] = this.text;
    json[r'confidence'] = this.confidence;
    json[r'source_segment_ids'] = this.sourceSegmentIds;
    json[r'status'] = this.status;
    return json;
  }

  /// Returns a new [SuggestionEvent] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SuggestionEvent? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "SuggestionEvent[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "SuggestionEvent[id]" has a null value in JSON.');
        assert(json.containsKey(r'encounter_id'),
            'Required key "SuggestionEvent[encounter_id]" is missing from JSON.');
        assert(json[r'encounter_id'] != null,
            'Required key "SuggestionEvent[encounter_id]" has a null value in JSON.');
        assert(json.containsKey(r'type'),
            'Required key "SuggestionEvent[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "SuggestionEvent[type]" has a null value in JSON.');
        assert(json.containsKey(r'text'),
            'Required key "SuggestionEvent[text]" is missing from JSON.');
        assert(json[r'text'] != null,
            'Required key "SuggestionEvent[text]" has a null value in JSON.');
        assert(json.containsKey(r'confidence'),
            'Required key "SuggestionEvent[confidence]" is missing from JSON.');
        assert(json[r'confidence'] != null,
            'Required key "SuggestionEvent[confidence]" has a null value in JSON.');
        assert(json.containsKey(r'source_segment_ids'),
            'Required key "SuggestionEvent[source_segment_ids]" is missing from JSON.');
        assert(json[r'source_segment_ids'] != null,
            'Required key "SuggestionEvent[source_segment_ids]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "SuggestionEvent[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "SuggestionEvent[status]" has a null value in JSON.');
        return true;
      }());

      return SuggestionEvent(
        id: mapValueOfType<String>(json, r'id')!,
        encounterId: mapValueOfType<String>(json, r'encounter_id')!,
        type: SuggestionEventTypeEnum.fromJson(json[r'type'])!,
        text: mapValueOfType<String>(json, r'text')!,
        confidence: num.parse('${json[r'confidence']}'),
        sourceSegmentIds: json[r'source_segment_ids'] is Iterable
            ? (json[r'source_segment_ids'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
        status: SuggestionEventStatusEnum.fromJson(json[r'status'])!,
      );
    }
    return null;
  }

  static List<SuggestionEvent> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SuggestionEvent>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SuggestionEvent.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SuggestionEvent> mapFromJson(dynamic json) {
    final map = <String, SuggestionEvent>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SuggestionEvent.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SuggestionEvent-objects as value to a dart map
  static Map<String, List<SuggestionEvent>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SuggestionEvent>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SuggestionEvent.listFromJson(
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
    'encounter_id',
    'type',
    'text',
    'confidence',
    'source_segment_ids',
    'status',
  };
}

enum SuggestionEventTypeEnum {
  diagnosis._(r'diagnosis'),
  question._(r'question'),
  redFlag._(r'red_flag'),
  term._(r'term'),
  ;

  /// Instantiate a new enum with the provided value.
  const SuggestionEventTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [SuggestionEventTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SuggestionEventTypeEnum? fromJson(dynamic value) =>
      SuggestionEventTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SuggestionEventTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SuggestionEventTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SuggestionEventTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SuggestionEventTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SuggestionEventTypeEnum] to String,
/// and [decode] dynamic data back to [SuggestionEventTypeEnum].
class SuggestionEventTypeEnumTypeTransformer {
  factory SuggestionEventTypeEnumTypeTransformer() =>
      _instance ??= const SuggestionEventTypeEnumTypeTransformer._();

  const SuggestionEventTypeEnumTypeTransformer._();

  String encode(SuggestionEventTypeEnum data) => data._value;

  /// Returns the instance of [SuggestionEventTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SuggestionEventTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is SuggestionEventTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'diagnosis':
          return SuggestionEventTypeEnum.diagnosis;
        case r'question':
          return SuggestionEventTypeEnum.question;
        case r'red_flag':
          return SuggestionEventTypeEnum.redFlag;
        case r'term':
          return SuggestionEventTypeEnum.term;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SuggestionEventTypeEnumTypeTransformer? _instance;
}

enum SuggestionEventStatusEnum {
  shown._(r'shown'),
  dismissed._(r'dismissed'),
  pinned._(r'pinned'),
  ;

  /// Instantiate a new enum with the provided value.
  const SuggestionEventStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [SuggestionEventStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SuggestionEventStatusEnum? fromJson(dynamic value) =>
      SuggestionEventStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SuggestionEventStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SuggestionEventStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SuggestionEventStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SuggestionEventStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SuggestionEventStatusEnum] to String,
/// and [decode] dynamic data back to [SuggestionEventStatusEnum].
class SuggestionEventStatusEnumTypeTransformer {
  factory SuggestionEventStatusEnumTypeTransformer() =>
      _instance ??= const SuggestionEventStatusEnumTypeTransformer._();

  const SuggestionEventStatusEnumTypeTransformer._();

  String encode(SuggestionEventStatusEnum data) => data._value;

  /// Returns the instance of [SuggestionEventStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SuggestionEventStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is SuggestionEventStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'shown':
          return SuggestionEventStatusEnum.shown;
        case r'dismissed':
          return SuggestionEventStatusEnum.dismissed;
        case r'pinned':
          return SuggestionEventStatusEnum.pinned;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SuggestionEventStatusEnumTypeTransformer? _instance;
}
