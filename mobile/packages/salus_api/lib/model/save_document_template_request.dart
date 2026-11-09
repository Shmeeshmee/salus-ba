//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SaveDocumentTemplateRequest {
  /// Returns a new [SaveDocumentTemplateRequest] instance.
  SaveDocumentTemplateRequest({
    required this.title,
    required this.language,
    required this.status,
    required this.typeCode,
    this.fields = const [],
    this.translations = const [],
  });

  String title;

  Language language;

  SaveDocumentTemplateRequestStatusEnum status;

  String typeCode;

  List<DocumentTemplateField> fields;

  List<DocumentTemplateTranslation> translations;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SaveDocumentTemplateRequest &&
          other.title == title &&
          other.language == language &&
          other.status == status &&
          other.typeCode == typeCode &&
          _deepEquality.equals(other.fields, fields) &&
          _deepEquality.equals(other.translations, translations);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (title.hashCode) +
      (language.hashCode) +
      (status.hashCode) +
      (typeCode.hashCode) +
      (fields.hashCode) +
      (translations.hashCode);

  @override
  String toString() =>
      'SaveDocumentTemplateRequest[title=$title, language=$language, status=$status, typeCode=$typeCode, fields=$fields, translations=$translations]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'title'] = this.title;
    json[r'language'] = this.language;
    json[r'status'] = this.status;
    json[r'typeCode'] = this.typeCode;
    json[r'fields'] = this.fields;
    json[r'translations'] = this.translations;
    return json;
  }

  /// Returns a new [SaveDocumentTemplateRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SaveDocumentTemplateRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'title'),
            'Required key "SaveDocumentTemplateRequest[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "SaveDocumentTemplateRequest[title]" has a null value in JSON.');
        assert(json.containsKey(r'language'),
            'Required key "SaveDocumentTemplateRequest[language]" is missing from JSON.');
        assert(json[r'language'] != null,
            'Required key "SaveDocumentTemplateRequest[language]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "SaveDocumentTemplateRequest[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "SaveDocumentTemplateRequest[status]" has a null value in JSON.');
        assert(json.containsKey(r'typeCode'),
            'Required key "SaveDocumentTemplateRequest[typeCode]" is missing from JSON.');
        assert(json[r'typeCode'] != null,
            'Required key "SaveDocumentTemplateRequest[typeCode]" has a null value in JSON.');
        assert(json.containsKey(r'fields'),
            'Required key "SaveDocumentTemplateRequest[fields]" is missing from JSON.');
        assert(json[r'fields'] != null,
            'Required key "SaveDocumentTemplateRequest[fields]" has a null value in JSON.');
        return true;
      }());

      return SaveDocumentTemplateRequest(
        title: mapValueOfType<String>(json, r'title')!,
        language: Language.fromJson(json[r'language'])!,
        status:
            SaveDocumentTemplateRequestStatusEnum.fromJson(json[r'status'])!,
        typeCode: mapValueOfType<String>(json, r'typeCode')!,
        fields: DocumentTemplateField.listFromJson(json[r'fields']),
        translations:
            DocumentTemplateTranslation.listFromJson(json[r'translations']),
      );
    }
    return null;
  }

  static List<SaveDocumentTemplateRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SaveDocumentTemplateRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SaveDocumentTemplateRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SaveDocumentTemplateRequest> mapFromJson(dynamic json) {
    final map = <String, SaveDocumentTemplateRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SaveDocumentTemplateRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SaveDocumentTemplateRequest-objects as value to a dart map
  static Map<String, List<SaveDocumentTemplateRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SaveDocumentTemplateRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SaveDocumentTemplateRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'title',
    'language',
    'status',
    'typeCode',
    'fields',
  };
}

enum SaveDocumentTemplateRequestStatusEnum {
  draft._(r'draft'),
  active._(r'active'),
  ;

  /// Instantiate a new enum with the provided value.
  const SaveDocumentTemplateRequestStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [SaveDocumentTemplateRequestStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SaveDocumentTemplateRequestStatusEnum? fromJson(dynamic value) =>
      SaveDocumentTemplateRequestStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SaveDocumentTemplateRequestStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SaveDocumentTemplateRequestStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SaveDocumentTemplateRequestStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SaveDocumentTemplateRequestStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SaveDocumentTemplateRequestStatusEnum] to String,
/// and [decode] dynamic data back to [SaveDocumentTemplateRequestStatusEnum].
class SaveDocumentTemplateRequestStatusEnumTypeTransformer {
  factory SaveDocumentTemplateRequestStatusEnumTypeTransformer() =>
      _instance ??=
          const SaveDocumentTemplateRequestStatusEnumTypeTransformer._();

  const SaveDocumentTemplateRequestStatusEnumTypeTransformer._();

  String encode(SaveDocumentTemplateRequestStatusEnum data) => data._value;

  /// Returns the instance of [SaveDocumentTemplateRequestStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SaveDocumentTemplateRequestStatusEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is SaveDocumentTemplateRequestStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'draft':
          return SaveDocumentTemplateRequestStatusEnum.draft;
        case r'active':
          return SaveDocumentTemplateRequestStatusEnum.active;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SaveDocumentTemplateRequestStatusEnumTypeTransformer? _instance;
}
