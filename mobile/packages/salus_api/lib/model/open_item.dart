//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class OpenItem {
  /// Returns a new [OpenItem] instance.
  OpenItem({
    required this.type,
    required this.reference,
  });

  OpenItemTypeEnum type;

  String reference;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is OpenItem && other.type == type && other.reference == reference;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (reference.hashCode);

  @override
  String toString() => 'OpenItem[type=$type, reference=$reference]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'reference'] = this.reference;
    return json;
  }

  /// Returns a new [OpenItem] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static OpenItem? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "OpenItem[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "OpenItem[type]" has a null value in JSON.');
        assert(json.containsKey(r'reference'),
            'Required key "OpenItem[reference]" is missing from JSON.');
        assert(json[r'reference'] != null,
            'Required key "OpenItem[reference]" has a null value in JSON.');
        return true;
      }());

      return OpenItem(
        type: OpenItemTypeEnum.fromJson(json[r'type'])!,
        reference: mapValueOfType<String>(json, r'reference')!,
      );
    }
    return null;
  }

  static List<OpenItem> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <OpenItem>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = OpenItem.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, OpenItem> mapFromJson(dynamic json) {
    final map = <String, OpenItem>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = OpenItem.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of OpenItem-objects as value to a dart map
  static Map<String, List<OpenItem>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<OpenItem>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = OpenItem.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'type',
    'reference',
  };
}

enum OpenItemTypeEnum {
  questionnairePending._(r'questionnaire_pending'),
  draftUnsigned._(r'draft_unsigned'),
  ;

  /// Instantiate a new enum with the provided value.
  const OpenItemTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [OpenItemTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static OpenItemTypeEnum? fromJson(dynamic value) =>
      OpenItemTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [OpenItemTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<OpenItemTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <OpenItemTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = OpenItemTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [OpenItemTypeEnum] to String,
/// and [decode] dynamic data back to [OpenItemTypeEnum].
class OpenItemTypeEnumTypeTransformer {
  factory OpenItemTypeEnumTypeTransformer() =>
      _instance ??= const OpenItemTypeEnumTypeTransformer._();

  const OpenItemTypeEnumTypeTransformer._();

  String encode(OpenItemTypeEnum data) => data._value;

  /// Returns the instance of [OpenItemTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  OpenItemTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is OpenItemTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'questionnaire_pending':
          return OpenItemTypeEnum.questionnairePending;
        case r'draft_unsigned':
          return OpenItemTypeEnum.draftUnsigned;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static OpenItemTypeEnumTypeTransformer? _instance;
}
