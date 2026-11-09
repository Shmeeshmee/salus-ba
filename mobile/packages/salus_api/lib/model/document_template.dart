//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DocumentTemplate {
  /// Returns a new [DocumentTemplate] instance.
  DocumentTemplate({
    required this.id,
    required this.title,
    required this.language,
    required this.status,
    required this.typeCode,
    this.fields = const [],
    this.translations = const [],
    required this.lastUpdated,
    required this.versionId,
  });

  String id;

  String title;

  Language language;

  DocumentTemplateStatusEnum status;

  /// The LOINC code of the document type (Composition.type), e.g. 11488-4 consult note, 57133-1 referral note.
  String typeCode;

  List<DocumentTemplateField> fields;

  List<DocumentTemplateTranslation> translations;

  DateTime lastUpdated;

  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DocumentTemplate &&
          other.id == id &&
          other.title == title &&
          other.language == language &&
          other.status == status &&
          other.typeCode == typeCode &&
          _deepEquality.equals(other.fields, fields) &&
          _deepEquality.equals(other.translations, translations) &&
          other.lastUpdated == lastUpdated &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (title.hashCode) +
      (language.hashCode) +
      (status.hashCode) +
      (typeCode.hashCode) +
      (fields.hashCode) +
      (translations.hashCode) +
      (lastUpdated.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'DocumentTemplate[id=$id, title=$title, language=$language, status=$status, typeCode=$typeCode, fields=$fields, translations=$translations, lastUpdated=$lastUpdated, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'title'] = this.title;
    json[r'language'] = this.language;
    json[r'status'] = this.status;
    json[r'typeCode'] = this.typeCode;
    json[r'fields'] = this.fields;
    json[r'translations'] = this.translations;
    json[r'lastUpdated'] = this.lastUpdated.toUtc().toIso8601String();
    json[r'versionId'] = this.versionId;
    return json;
  }

  /// Returns a new [DocumentTemplate] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DocumentTemplate? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "DocumentTemplate[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "DocumentTemplate[id]" has a null value in JSON.');
        assert(json.containsKey(r'title'),
            'Required key "DocumentTemplate[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "DocumentTemplate[title]" has a null value in JSON.');
        assert(json.containsKey(r'language'),
            'Required key "DocumentTemplate[language]" is missing from JSON.');
        assert(json[r'language'] != null,
            'Required key "DocumentTemplate[language]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "DocumentTemplate[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "DocumentTemplate[status]" has a null value in JSON.');
        assert(json.containsKey(r'typeCode'),
            'Required key "DocumentTemplate[typeCode]" is missing from JSON.');
        assert(json[r'typeCode'] != null,
            'Required key "DocumentTemplate[typeCode]" has a null value in JSON.');
        assert(json.containsKey(r'fields'),
            'Required key "DocumentTemplate[fields]" is missing from JSON.');
        assert(json[r'fields'] != null,
            'Required key "DocumentTemplate[fields]" has a null value in JSON.');
        assert(json.containsKey(r'translations'),
            'Required key "DocumentTemplate[translations]" is missing from JSON.');
        assert(json[r'translations'] != null,
            'Required key "DocumentTemplate[translations]" has a null value in JSON.');
        assert(json.containsKey(r'lastUpdated'),
            'Required key "DocumentTemplate[lastUpdated]" is missing from JSON.');
        assert(json[r'lastUpdated'] != null,
            'Required key "DocumentTemplate[lastUpdated]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "DocumentTemplate[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "DocumentTemplate[versionId]" has a null value in JSON.');
        return true;
      }());

      return DocumentTemplate(
        id: mapValueOfType<String>(json, r'id')!,
        title: mapValueOfType<String>(json, r'title')!,
        language: Language.fromJson(json[r'language'])!,
        status: DocumentTemplateStatusEnum.fromJson(json[r'status'])!,
        typeCode: mapValueOfType<String>(json, r'typeCode')!,
        fields: DocumentTemplateField.listFromJson(json[r'fields']),
        translations:
            DocumentTemplateTranslation.listFromJson(json[r'translations']),
        lastUpdated: mapDateTime(json, r'lastUpdated', r'')!,
        versionId: mapValueOfType<int>(json, r'versionId')!,
      );
    }
    return null;
  }

  static List<DocumentTemplate> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DocumentTemplate>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DocumentTemplate.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DocumentTemplate> mapFromJson(dynamic json) {
    final map = <String, DocumentTemplate>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DocumentTemplate.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DocumentTemplate-objects as value to a dart map
  static Map<String, List<DocumentTemplate>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DocumentTemplate>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DocumentTemplate.listFromJson(
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
    'title',
    'language',
    'status',
    'typeCode',
    'fields',
    'translations',
    'lastUpdated',
    'versionId',
  };
}

enum DocumentTemplateStatusEnum {
  draft._(r'draft'),
  active._(r'active'),
  ;

  /// Instantiate a new enum with the provided value.
  const DocumentTemplateStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [DocumentTemplateStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static DocumentTemplateStatusEnum? fromJson(dynamic value) =>
      DocumentTemplateStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [DocumentTemplateStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<DocumentTemplateStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DocumentTemplateStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DocumentTemplateStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [DocumentTemplateStatusEnum] to String,
/// and [decode] dynamic data back to [DocumentTemplateStatusEnum].
class DocumentTemplateStatusEnumTypeTransformer {
  factory DocumentTemplateStatusEnumTypeTransformer() =>
      _instance ??= const DocumentTemplateStatusEnumTypeTransformer._();

  const DocumentTemplateStatusEnumTypeTransformer._();

  String encode(DocumentTemplateStatusEnum data) => data._value;

  /// Returns the instance of [DocumentTemplateStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  DocumentTemplateStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is DocumentTemplateStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'draft':
          return DocumentTemplateStatusEnum.draft;
        case r'active':
          return DocumentTemplateStatusEnum.active;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static DocumentTemplateStatusEnumTypeTransformer? _instance;
}
