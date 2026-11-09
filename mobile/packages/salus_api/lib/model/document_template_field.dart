//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DocumentTemplateField {
  /// Returns a new [DocumentTemplateField] instance.
  DocumentTemplateField({
    required this.key,
    required this.type,
    required this.label,
    this.instruction,
    required this.required_,
  });

  String key;

  DocumentTemplateFieldTypeEnum type;

  String label;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? instruction;

  bool required_;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DocumentTemplateField &&
          other.key == key &&
          other.type == type &&
          other.label == label &&
          other.instruction == instruction &&
          other.required_ == required_;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (key.hashCode) +
      (type.hashCode) +
      (label.hashCode) +
      (instruction == null ? 0 : instruction!.hashCode) +
      (required_.hashCode);

  @override
  String toString() =>
      'DocumentTemplateField[key=$key, type=$type, label=$label, instruction=$instruction, required_=$required_]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'key'] = this.key;
    json[r'type'] = this.type;
    json[r'label'] = this.label;
    if (this.instruction != null) {
      json[r'instruction'] = this.instruction;
    } else {
      json[r'instruction'] = null;
    }
    json[r'required'] = this.required_;
    return json;
  }

  /// Returns a new [DocumentTemplateField] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DocumentTemplateField? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'key'),
            'Required key "DocumentTemplateField[key]" is missing from JSON.');
        assert(json[r'key'] != null,
            'Required key "DocumentTemplateField[key]" has a null value in JSON.');
        assert(json.containsKey(r'type'),
            'Required key "DocumentTemplateField[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "DocumentTemplateField[type]" has a null value in JSON.');
        assert(json.containsKey(r'label'),
            'Required key "DocumentTemplateField[label]" is missing from JSON.');
        assert(json[r'label'] != null,
            'Required key "DocumentTemplateField[label]" has a null value in JSON.');
        assert(json.containsKey(r'required'),
            'Required key "DocumentTemplateField[required]" is missing from JSON.');
        assert(json[r'required'] != null,
            'Required key "DocumentTemplateField[required]" has a null value in JSON.');
        return true;
      }());

      return DocumentTemplateField(
        key: mapValueOfType<String>(json, r'key')!,
        type: DocumentTemplateFieldTypeEnum.fromJson(json[r'type'])!,
        label: mapValueOfType<String>(json, r'label')!,
        instruction: mapValueOfType<String>(json, r'instruction'),
        required_: mapValueOfType<bool>(json, r'required')!,
      );
    }
    return null;
  }

  static List<DocumentTemplateField> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DocumentTemplateField>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DocumentTemplateField.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DocumentTemplateField> mapFromJson(dynamic json) {
    final map = <String, DocumentTemplateField>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DocumentTemplateField.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DocumentTemplateField-objects as value to a dart map
  static Map<String, List<DocumentTemplateField>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DocumentTemplateField>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DocumentTemplateField.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'key',
    'type',
    'label',
    'required',
  };
}

enum DocumentTemplateFieldTypeEnum {
  text._(r'text'),
  list._(r'list'),
  date._(r'date'),
  coded._(r'coded'),
  ;

  /// Instantiate a new enum with the provided value.
  const DocumentTemplateFieldTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [DocumentTemplateFieldTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static DocumentTemplateFieldTypeEnum? fromJson(dynamic value) =>
      DocumentTemplateFieldTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [DocumentTemplateFieldTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<DocumentTemplateFieldTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DocumentTemplateFieldTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DocumentTemplateFieldTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [DocumentTemplateFieldTypeEnum] to String,
/// and [decode] dynamic data back to [DocumentTemplateFieldTypeEnum].
class DocumentTemplateFieldTypeEnumTypeTransformer {
  factory DocumentTemplateFieldTypeEnumTypeTransformer() =>
      _instance ??= const DocumentTemplateFieldTypeEnumTypeTransformer._();

  const DocumentTemplateFieldTypeEnumTypeTransformer._();

  String encode(DocumentTemplateFieldTypeEnum data) => data._value;

  /// Returns the instance of [DocumentTemplateFieldTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  DocumentTemplateFieldTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is DocumentTemplateFieldTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'text':
          return DocumentTemplateFieldTypeEnum.text;
        case r'list':
          return DocumentTemplateFieldTypeEnum.list;
        case r'date':
          return DocumentTemplateFieldTypeEnum.date;
        case r'coded':
          return DocumentTemplateFieldTypeEnum.coded;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static DocumentTemplateFieldTypeEnumTypeTransformer? _instance;
}
