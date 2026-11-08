//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class QuestionnaireResponse {
  /// Returns a new [QuestionnaireResponse] instance.
  QuestionnaireResponse({
    required this.id,
    required this.appointmentId,
    required this.questionnaireId,
    required this.status,
    required this.source_,
    this.answers = const [],
    required this.dueAt,
    this.authored,
    this.language,
    required this.lastUpdated,
    required this.versionId,
  });

  String id;

  String appointmentId;

  /// The questionnaire version asked.
  String questionnaireId;

  /// In progress, submitted, or withdrawn by the practitioner.
  QuestionnaireResponseStatusEnum status;

  /// The practitioner role's default, with the booking, or added by the practitioner.
  QuestionnaireResponseSource_Enum source_;

  List<Answer> answers;

  /// The appointment's start; the answers are due before it.
  DateTime dueAt;

  /// When the patient submitted; absent before.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? authored;

  /// The language the patient answers in; absent until they start.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  Language? language;

  DateTime lastUpdated;

  /// The version of the answers, for If-Match.
  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is QuestionnaireResponse &&
          other.id == id &&
          other.appointmentId == appointmentId &&
          other.questionnaireId == questionnaireId &&
          other.status == status &&
          other.source_ == source_ &&
          _deepEquality.equals(other.answers, answers) &&
          other.dueAt == dueAt &&
          other.authored == authored &&
          other.language == language &&
          other.lastUpdated == lastUpdated &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (appointmentId.hashCode) +
      (questionnaireId.hashCode) +
      (status.hashCode) +
      (source_.hashCode) +
      (answers.hashCode) +
      (dueAt.hashCode) +
      (authored == null ? 0 : authored!.hashCode) +
      (language == null ? 0 : language!.hashCode) +
      (lastUpdated.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'QuestionnaireResponse[id=$id, appointmentId=$appointmentId, questionnaireId=$questionnaireId, status=$status, source_=$source_, answers=$answers, dueAt=$dueAt, authored=$authored, language=$language, lastUpdated=$lastUpdated, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'appointmentId'] = this.appointmentId;
    json[r'questionnaireId'] = this.questionnaireId;
    json[r'status'] = this.status;
    json[r'source'] = this.source_;
    json[r'answers'] = this.answers;
    json[r'dueAt'] = this.dueAt.toUtc().toIso8601String();
    if (this.authored != null) {
      json[r'authored'] = this.authored!.toUtc().toIso8601String();
    } else {
      json[r'authored'] = null;
    }
    if (this.language != null) {
      json[r'language'] = this.language;
    } else {
      json[r'language'] = null;
    }
    json[r'lastUpdated'] = this.lastUpdated.toUtc().toIso8601String();
    json[r'versionId'] = this.versionId;
    return json;
  }

  /// Returns a new [QuestionnaireResponse] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static QuestionnaireResponse? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "QuestionnaireResponse[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "QuestionnaireResponse[id]" has a null value in JSON.');
        assert(json.containsKey(r'appointmentId'),
            'Required key "QuestionnaireResponse[appointmentId]" is missing from JSON.');
        assert(json[r'appointmentId'] != null,
            'Required key "QuestionnaireResponse[appointmentId]" has a null value in JSON.');
        assert(json.containsKey(r'questionnaireId'),
            'Required key "QuestionnaireResponse[questionnaireId]" is missing from JSON.');
        assert(json[r'questionnaireId'] != null,
            'Required key "QuestionnaireResponse[questionnaireId]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "QuestionnaireResponse[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "QuestionnaireResponse[status]" has a null value in JSON.');
        assert(json.containsKey(r'source'),
            'Required key "QuestionnaireResponse[source]" is missing from JSON.');
        assert(json[r'source'] != null,
            'Required key "QuestionnaireResponse[source]" has a null value in JSON.');
        assert(json.containsKey(r'answers'),
            'Required key "QuestionnaireResponse[answers]" is missing from JSON.');
        assert(json[r'answers'] != null,
            'Required key "QuestionnaireResponse[answers]" has a null value in JSON.');
        assert(json.containsKey(r'dueAt'),
            'Required key "QuestionnaireResponse[dueAt]" is missing from JSON.');
        assert(json[r'dueAt'] != null,
            'Required key "QuestionnaireResponse[dueAt]" has a null value in JSON.');
        assert(json.containsKey(r'lastUpdated'),
            'Required key "QuestionnaireResponse[lastUpdated]" is missing from JSON.');
        assert(json[r'lastUpdated'] != null,
            'Required key "QuestionnaireResponse[lastUpdated]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "QuestionnaireResponse[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "QuestionnaireResponse[versionId]" has a null value in JSON.');
        return true;
      }());

      return QuestionnaireResponse(
        id: mapValueOfType<String>(json, r'id')!,
        appointmentId: mapValueOfType<String>(json, r'appointmentId')!,
        questionnaireId: mapValueOfType<String>(json, r'questionnaireId')!,
        status: QuestionnaireResponseStatusEnum.fromJson(json[r'status'])!,
        source_: QuestionnaireResponseSource_Enum.fromJson(json[r'source'])!,
        answers: Answer.listFromJson(json[r'answers']),
        dueAt: mapDateTime(json, r'dueAt', r'')!,
        authored: mapDateTime(json, r'authored', r''),
        language: Language.fromJson(json[r'language']),
        lastUpdated: mapDateTime(json, r'lastUpdated', r'')!,
        versionId: mapValueOfType<int>(json, r'versionId')!,
      );
    }
    return null;
  }

  static List<QuestionnaireResponse> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuestionnaireResponse>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = QuestionnaireResponse.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, QuestionnaireResponse> mapFromJson(dynamic json) {
    final map = <String, QuestionnaireResponse>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = QuestionnaireResponse.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of QuestionnaireResponse-objects as value to a dart map
  static Map<String, List<QuestionnaireResponse>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<QuestionnaireResponse>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = QuestionnaireResponse.listFromJson(
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
    'appointmentId',
    'questionnaireId',
    'status',
    'source',
    'answers',
    'dueAt',
    'lastUpdated',
    'versionId',
  };
}

/// In progress, submitted, or withdrawn by the practitioner.
enum QuestionnaireResponseStatusEnum {
  inProgress._(r'in-progress'),
  completed._(r'completed'),
  stopped._(r'stopped'),
  ;

  /// Instantiate a new enum with the provided value.
  const QuestionnaireResponseStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [QuestionnaireResponseStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static QuestionnaireResponseStatusEnum? fromJson(dynamic value) =>
      QuestionnaireResponseStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [QuestionnaireResponseStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<QuestionnaireResponseStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuestionnaireResponseStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = QuestionnaireResponseStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [QuestionnaireResponseStatusEnum] to String,
/// and [decode] dynamic data back to [QuestionnaireResponseStatusEnum].
class QuestionnaireResponseStatusEnumTypeTransformer {
  factory QuestionnaireResponseStatusEnumTypeTransformer() =>
      _instance ??= const QuestionnaireResponseStatusEnumTypeTransformer._();

  const QuestionnaireResponseStatusEnumTypeTransformer._();

  String encode(QuestionnaireResponseStatusEnum data) => data._value;

  /// Returns the instance of [QuestionnaireResponseStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  QuestionnaireResponseStatusEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is QuestionnaireResponseStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'in-progress':
          return QuestionnaireResponseStatusEnum.inProgress;
        case r'completed':
          return QuestionnaireResponseStatusEnum.completed;
        case r'stopped':
          return QuestionnaireResponseStatusEnum.stopped;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static QuestionnaireResponseStatusEnumTypeTransformer? _instance;
}

/// The practitioner role's default, with the booking, or added by the practitioner.
enum QuestionnaireResponseSource_Enum {
  default_._(r'default'),
  added._(r'added'),
  ;

  /// Instantiate a new enum with the provided value.
  const QuestionnaireResponseSource_Enum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [QuestionnaireResponseSource_Enum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static QuestionnaireResponseSource_Enum? fromJson(dynamic value) =>
      QuestionnaireResponseSource_EnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [QuestionnaireResponseSource_Enum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<QuestionnaireResponseSource_Enum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuestionnaireResponseSource_Enum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = QuestionnaireResponseSource_Enum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [QuestionnaireResponseSource_Enum] to String,
/// and [decode] dynamic data back to [QuestionnaireResponseSource_Enum].
class QuestionnaireResponseSource_EnumTypeTransformer {
  factory QuestionnaireResponseSource_EnumTypeTransformer() =>
      _instance ??= const QuestionnaireResponseSource_EnumTypeTransformer._();

  const QuestionnaireResponseSource_EnumTypeTransformer._();

  String encode(QuestionnaireResponseSource_Enum data) => data._value;

  /// Returns the instance of [QuestionnaireResponseSource_Enum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  QuestionnaireResponseSource_Enum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is QuestionnaireResponseSource_Enum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'default':
          return QuestionnaireResponseSource_Enum.default_;
        case r'added':
          return QuestionnaireResponseSource_Enum.added;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static QuestionnaireResponseSource_EnumTypeTransformer? _instance;
}
