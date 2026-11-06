//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Questionnaire {
  /// Returns a new [Questionnaire] instance.
  Questionnaire({
    required this.id,
    required this.url,
    required this.status,
    required this.title,
    this.description,
    this.item = const [],
    required this.lastUpdated,
    required this.version,
    required this.versionId,
  });

  String id;

  /// The canonical url, the same for every version of the questionnaire.
  String url;

  QuestionnaireStatusEnum status;

  String title;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? description;

  List<QuestionnaireItem> item;

  DateTime lastUpdated;

  /// 1, 2, … within the questionnaire.
  int version;

  /// The version of this row, for If-Match.
  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Questionnaire &&
          other.id == id &&
          other.url == url &&
          other.status == status &&
          other.title == title &&
          other.description == description &&
          _deepEquality.equals(other.item, item) &&
          other.lastUpdated == lastUpdated &&
          other.version == version &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (url.hashCode) +
      (status.hashCode) +
      (title.hashCode) +
      (description == null ? 0 : description!.hashCode) +
      (item.hashCode) +
      (lastUpdated.hashCode) +
      (version.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'Questionnaire[id=$id, url=$url, status=$status, title=$title, description=$description, item=$item, lastUpdated=$lastUpdated, version=$version, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'url'] = this.url;
    json[r'status'] = this.status;
    json[r'title'] = this.title;
    if (this.description != null) {
      json[r'description'] = this.description;
    } else {
      json[r'description'] = null;
    }
    json[r'item'] = this.item;
    json[r'lastUpdated'] = this.lastUpdated.toUtc().toIso8601String();
    json[r'version'] = this.version;
    json[r'versionId'] = this.versionId;
    return json;
  }

  /// Returns a new [Questionnaire] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Questionnaire? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Questionnaire[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Questionnaire[id]" has a null value in JSON.');
        assert(json.containsKey(r'url'),
            'Required key "Questionnaire[url]" is missing from JSON.');
        assert(json[r'url'] != null,
            'Required key "Questionnaire[url]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "Questionnaire[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "Questionnaire[status]" has a null value in JSON.');
        assert(json.containsKey(r'title'),
            'Required key "Questionnaire[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "Questionnaire[title]" has a null value in JSON.');
        assert(json.containsKey(r'item'),
            'Required key "Questionnaire[item]" is missing from JSON.');
        assert(json[r'item'] != null,
            'Required key "Questionnaire[item]" has a null value in JSON.');
        assert(json.containsKey(r'lastUpdated'),
            'Required key "Questionnaire[lastUpdated]" is missing from JSON.');
        assert(json[r'lastUpdated'] != null,
            'Required key "Questionnaire[lastUpdated]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "Questionnaire[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "Questionnaire[version]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "Questionnaire[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "Questionnaire[versionId]" has a null value in JSON.');
        return true;
      }());

      return Questionnaire(
        id: mapValueOfType<String>(json, r'id')!,
        url: mapValueOfType<String>(json, r'url')!,
        status: QuestionnaireStatusEnum.fromJson(json[r'status'])!,
        title: mapValueOfType<String>(json, r'title')!,
        description: mapValueOfType<String>(json, r'description'),
        item: QuestionnaireItem.listFromJson(json[r'item']),
        lastUpdated: mapDateTime(json, r'lastUpdated', r'')!,
        version: mapValueOfType<int>(json, r'version')!,
        versionId: mapValueOfType<int>(json, r'versionId')!,
      );
    }
    return null;
  }

  static List<Questionnaire> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Questionnaire>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Questionnaire.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Questionnaire> mapFromJson(dynamic json) {
    final map = <String, Questionnaire>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Questionnaire.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Questionnaire-objects as value to a dart map
  static Map<String, List<Questionnaire>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Questionnaire>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Questionnaire.listFromJson(
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
    'url',
    'status',
    'title',
    'item',
    'lastUpdated',
    'version',
    'versionId',
  };
}

enum QuestionnaireStatusEnum {
  draft._(r'draft'),
  active._(r'active'),
  retired._(r'retired'),
  ;

  /// Instantiate a new enum with the provided value.
  const QuestionnaireStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [QuestionnaireStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static QuestionnaireStatusEnum? fromJson(dynamic value) =>
      QuestionnaireStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [QuestionnaireStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<QuestionnaireStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuestionnaireStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = QuestionnaireStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [QuestionnaireStatusEnum] to String,
/// and [decode] dynamic data back to [QuestionnaireStatusEnum].
class QuestionnaireStatusEnumTypeTransformer {
  factory QuestionnaireStatusEnumTypeTransformer() =>
      _instance ??= const QuestionnaireStatusEnumTypeTransformer._();

  const QuestionnaireStatusEnumTypeTransformer._();

  String encode(QuestionnaireStatusEnum data) => data._value;

  /// Returns the instance of [QuestionnaireStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  QuestionnaireStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is QuestionnaireStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'draft':
          return QuestionnaireStatusEnum.draft;
        case r'active':
          return QuestionnaireStatusEnum.active;
        case r'retired':
          return QuestionnaireStatusEnum.retired;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static QuestionnaireStatusEnumTypeTransformer? _instance;
}
