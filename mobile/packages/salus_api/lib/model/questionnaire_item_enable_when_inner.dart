//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class QuestionnaireItemEnableWhenInner {
  /// Returns a new [QuestionnaireItemEnableWhenInner] instance.
  QuestionnaireItemEnableWhenInner({
    required this.question,
    required this.operator_,
    this.answerBoolean,
    this.answerString,
    this.answerInteger,
    this.answerDecimal,
    this.answerDate,
  });

  String question;

  QuestionnaireItemEnableWhenInnerOperator_Enum operator_;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  bool? answerBoolean;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? answerString;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? answerInteger;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  num? answerDecimal;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? answerDate;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is QuestionnaireItemEnableWhenInner &&
          other.question == question &&
          other.operator_ == operator_ &&
          other.answerBoolean == answerBoolean &&
          other.answerString == answerString &&
          other.answerInteger == answerInteger &&
          other.answerDecimal == answerDecimal &&
          other.answerDate == answerDate;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (question.hashCode) +
      (operator_.hashCode) +
      (answerBoolean == null ? 0 : answerBoolean!.hashCode) +
      (answerString == null ? 0 : answerString!.hashCode) +
      (answerInteger == null ? 0 : answerInteger!.hashCode) +
      (answerDecimal == null ? 0 : answerDecimal!.hashCode) +
      (answerDate == null ? 0 : answerDate!.hashCode);

  @override
  String toString() =>
      'QuestionnaireItemEnableWhenInner[question=$question, operator_=$operator_, answerBoolean=$answerBoolean, answerString=$answerString, answerInteger=$answerInteger, answerDecimal=$answerDecimal, answerDate=$answerDate]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'question'] = this.question;
    json[r'operator'] = this.operator_;
    if (this.answerBoolean != null) {
      json[r'answerBoolean'] = this.answerBoolean;
    } else {
      json[r'answerBoolean'] = null;
    }
    if (this.answerString != null) {
      json[r'answerString'] = this.answerString;
    } else {
      json[r'answerString'] = null;
    }
    if (this.answerInteger != null) {
      json[r'answerInteger'] = this.answerInteger;
    } else {
      json[r'answerInteger'] = null;
    }
    if (this.answerDecimal != null) {
      json[r'answerDecimal'] = this.answerDecimal;
    } else {
      json[r'answerDecimal'] = null;
    }
    if (this.answerDate != null) {
      json[r'answerDate'] = _dateFormatter.format(this.answerDate!);
    } else {
      json[r'answerDate'] = null;
    }
    return json;
  }

  /// Returns a new [QuestionnaireItemEnableWhenInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static QuestionnaireItemEnableWhenInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'question'),
            'Required key "QuestionnaireItemEnableWhenInner[question]" is missing from JSON.');
        assert(json[r'question'] != null,
            'Required key "QuestionnaireItemEnableWhenInner[question]" has a null value in JSON.');
        assert(json.containsKey(r'operator'),
            'Required key "QuestionnaireItemEnableWhenInner[operator]" is missing from JSON.');
        assert(json[r'operator'] != null,
            'Required key "QuestionnaireItemEnableWhenInner[operator]" has a null value in JSON.');
        return true;
      }());

      return QuestionnaireItemEnableWhenInner(
        question: mapValueOfType<String>(json, r'question')!,
        operator_: QuestionnaireItemEnableWhenInnerOperator_Enum.fromJson(
            json[r'operator'])!,
        answerBoolean: mapValueOfType<bool>(json, r'answerBoolean'),
        answerString: mapValueOfType<String>(json, r'answerString'),
        answerInteger: mapValueOfType<int>(json, r'answerInteger'),
        answerDecimal: json[r'answerDecimal'] == null
            ? null
            : num.parse('${json[r'answerDecimal']}'),
        answerDate: mapDateTime(json, r'answerDate', r''),
      );
    }
    return null;
  }

  static List<QuestionnaireItemEnableWhenInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuestionnaireItemEnableWhenInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = QuestionnaireItemEnableWhenInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, QuestionnaireItemEnableWhenInner> mapFromJson(
      dynamic json) {
    final map = <String, QuestionnaireItemEnableWhenInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = QuestionnaireItemEnableWhenInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of QuestionnaireItemEnableWhenInner-objects as value to a dart map
  static Map<String, List<QuestionnaireItemEnableWhenInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<QuestionnaireItemEnableWhenInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = QuestionnaireItemEnableWhenInner.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'question',
    'operator',
  };
}

enum QuestionnaireItemEnableWhenInnerOperator_Enum {
  exists._(r'exists'),
  equal._(r'='),
  exclamationEqual._(r'!='),
  greaterThan._(r'>'),
  lessThan._(r'<'),
  greaterThanEqual._(r'>='),
  lessThanEqual._(r'<='),
  ;

  /// Instantiate a new enum with the provided value.
  const QuestionnaireItemEnableWhenInnerOperator_Enum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [QuestionnaireItemEnableWhenInnerOperator_Enum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static QuestionnaireItemEnableWhenInnerOperator_Enum? fromJson(
          dynamic value) =>
      QuestionnaireItemEnableWhenInnerOperator_EnumTypeTransformer()
          .decode(value);

  /// Returns a [List] containing instances of [QuestionnaireItemEnableWhenInnerOperator_Enum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<QuestionnaireItemEnableWhenInnerOperator_Enum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuestionnaireItemEnableWhenInnerOperator_Enum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value =
            QuestionnaireItemEnableWhenInnerOperator_Enum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [QuestionnaireItemEnableWhenInnerOperator_Enum] to String,
/// and [decode] dynamic data back to [QuestionnaireItemEnableWhenInnerOperator_Enum].
class QuestionnaireItemEnableWhenInnerOperator_EnumTypeTransformer {
  factory QuestionnaireItemEnableWhenInnerOperator_EnumTypeTransformer() =>
      _instance ??=
          const QuestionnaireItemEnableWhenInnerOperator_EnumTypeTransformer
              ._();

  const QuestionnaireItemEnableWhenInnerOperator_EnumTypeTransformer._();

  String encode(QuestionnaireItemEnableWhenInnerOperator_Enum data) =>
      data._value;

  /// Returns the instance of [QuestionnaireItemEnableWhenInnerOperator_Enum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  QuestionnaireItemEnableWhenInnerOperator_Enum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is QuestionnaireItemEnableWhenInnerOperator_Enum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'exists':
          return QuestionnaireItemEnableWhenInnerOperator_Enum.exists;
        case r'=':
          return QuestionnaireItemEnableWhenInnerOperator_Enum.equal;
        case r'!=':
          return QuestionnaireItemEnableWhenInnerOperator_Enum.exclamationEqual;
        case r'>':
          return QuestionnaireItemEnableWhenInnerOperator_Enum.greaterThan;
        case r'<':
          return QuestionnaireItemEnableWhenInnerOperator_Enum.lessThan;
        case r'>=':
          return QuestionnaireItemEnableWhenInnerOperator_Enum.greaterThanEqual;
        case r'<=':
          return QuestionnaireItemEnableWhenInnerOperator_Enum.lessThanEqual;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static QuestionnaireItemEnableWhenInnerOperator_EnumTypeTransformer?
      _instance;
}
