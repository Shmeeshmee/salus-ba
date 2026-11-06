//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class QuestionnaireItem {
  /// Returns a new [QuestionnaireItem] instance.
  QuestionnaireItem({
    required this.linkId,
    required this.text,
    required this.type,
    this.required_,
    this.repeats,
    this.help,
    this.answerOption = const [],
    this.enableWhen,
    this.item = const [],
  });

  /// Opaque and random, unique within the questionnaire, kept by later versions.
  String linkId;

  /// Not blank.
  String text;

  QuestionnaireItemTypeEnum type;

  /// Questions only.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  bool? required_;

  /// Choice only, several answers.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  bool? repeats;

  /// Help text, on questions and sections.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? help;

  /// Choice only, each value different.
  List<QuestionnaireItemAnswerOptionInner> answerOption;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  QuestionnaireItemEnableWhen? enableWhen;

  /// A section's questions and instructions.
  List<QuestionnaireItem> item;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is QuestionnaireItem &&
          other.linkId == linkId &&
          other.text == text &&
          other.type == type &&
          other.required_ == required_ &&
          other.repeats == repeats &&
          other.help == help &&
          _deepEquality.equals(other.answerOption, answerOption) &&
          other.enableWhen == enableWhen &&
          _deepEquality.equals(other.item, item);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (linkId.hashCode) +
      (text.hashCode) +
      (type.hashCode) +
      (required_ == null ? 0 : required_!.hashCode) +
      (repeats == null ? 0 : repeats!.hashCode) +
      (help == null ? 0 : help!.hashCode) +
      (answerOption.hashCode) +
      (enableWhen == null ? 0 : enableWhen!.hashCode) +
      (item.hashCode);

  @override
  String toString() =>
      'QuestionnaireItem[linkId=$linkId, text=$text, type=$type, required_=$required_, repeats=$repeats, help=$help, answerOption=$answerOption, enableWhen=$enableWhen, item=$item]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'linkId'] = this.linkId;
    json[r'text'] = this.text;
    json[r'type'] = this.type;
    if (this.required_ != null) {
      json[r'required'] = this.required_;
    } else {
      json[r'required'] = null;
    }
    if (this.repeats != null) {
      json[r'repeats'] = this.repeats;
    } else {
      json[r'repeats'] = null;
    }
    if (this.help != null) {
      json[r'help'] = this.help;
    } else {
      json[r'help'] = null;
    }
    json[r'answerOption'] = this.answerOption;
    if (this.enableWhen != null) {
      json[r'enableWhen'] = this.enableWhen;
    } else {
      json[r'enableWhen'] = null;
    }
    json[r'item'] = this.item;
    return json;
  }

  /// Returns a new [QuestionnaireItem] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static QuestionnaireItem? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'linkId'),
            'Required key "QuestionnaireItem[linkId]" is missing from JSON.');
        assert(json[r'linkId'] != null,
            'Required key "QuestionnaireItem[linkId]" has a null value in JSON.');
        assert(json.containsKey(r'text'),
            'Required key "QuestionnaireItem[text]" is missing from JSON.');
        assert(json[r'text'] != null,
            'Required key "QuestionnaireItem[text]" has a null value in JSON.');
        assert(json.containsKey(r'type'),
            'Required key "QuestionnaireItem[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "QuestionnaireItem[type]" has a null value in JSON.');
        return true;
      }());

      return QuestionnaireItem(
        linkId: mapValueOfType<String>(json, r'linkId')!,
        text: mapValueOfType<String>(json, r'text')!,
        type: QuestionnaireItemTypeEnum.fromJson(json[r'type'])!,
        required_: mapValueOfType<bool>(json, r'required'),
        repeats: mapValueOfType<bool>(json, r'repeats'),
        help: mapValueOfType<String>(json, r'help'),
        answerOption: QuestionnaireItemAnswerOptionInner.listFromJson(
            json[r'answerOption']),
        enableWhen: QuestionnaireItemEnableWhen.fromJson(json[r'enableWhen']),
        item: QuestionnaireItem.listFromJson(json[r'item']),
      );
    }
    return null;
  }

  static List<QuestionnaireItem> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuestionnaireItem>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = QuestionnaireItem.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, QuestionnaireItem> mapFromJson(dynamic json) {
    final map = <String, QuestionnaireItem>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = QuestionnaireItem.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of QuestionnaireItem-objects as value to a dart map
  static Map<String, List<QuestionnaireItem>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<QuestionnaireItem>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = QuestionnaireItem.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'linkId',
    'text',
    'type',
  };
}

enum QuestionnaireItemTypeEnum {
  group._(r'group'),
  display._(r'display'),
  string._(r'string'),
  text._(r'text'),
  boolean._(r'boolean'),
  choice._(r'choice'),
  integer._(r'integer'),
  decimal._(r'decimal'),
  date._(r'date'),
  ;

  /// Instantiate a new enum with the provided value.
  const QuestionnaireItemTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [QuestionnaireItemTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static QuestionnaireItemTypeEnum? fromJson(dynamic value) =>
      QuestionnaireItemTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [QuestionnaireItemTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<QuestionnaireItemTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuestionnaireItemTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = QuestionnaireItemTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [QuestionnaireItemTypeEnum] to String,
/// and [decode] dynamic data back to [QuestionnaireItemTypeEnum].
class QuestionnaireItemTypeEnumTypeTransformer {
  factory QuestionnaireItemTypeEnumTypeTransformer() =>
      _instance ??= const QuestionnaireItemTypeEnumTypeTransformer._();

  const QuestionnaireItemTypeEnumTypeTransformer._();

  String encode(QuestionnaireItemTypeEnum data) => data._value;

  /// Returns the instance of [QuestionnaireItemTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  QuestionnaireItemTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is QuestionnaireItemTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'group':
          return QuestionnaireItemTypeEnum.group;
        case r'display':
          return QuestionnaireItemTypeEnum.display;
        case r'string':
          return QuestionnaireItemTypeEnum.string;
        case r'text':
          return QuestionnaireItemTypeEnum.text;
        case r'boolean':
          return QuestionnaireItemTypeEnum.boolean;
        case r'choice':
          return QuestionnaireItemTypeEnum.choice;
        case r'integer':
          return QuestionnaireItemTypeEnum.integer;
        case r'decimal':
          return QuestionnaireItemTypeEnum.decimal;
        case r'date':
          return QuestionnaireItemTypeEnum.date;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static QuestionnaireItemTypeEnumTypeTransformer? _instance;
}
