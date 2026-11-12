//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatHour {
  /// Returns a new [ChatHour] instance.
  ChatHour({
    required this.day,
    required this.start,
    required this.end,
  });

  /// FHIR days-of-week.
  ChatHourDayEnum day;

  /// HH:MM in the clinic's time zone.
  String start;

  /// HH:MM, after start; 24:00 for midnight.
  String end;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatHour &&
          other.day == day &&
          other.start == start &&
          other.end == end;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (day.hashCode) + (start.hashCode) + (end.hashCode);

  @override
  String toString() => 'ChatHour[day=$day, start=$start, end=$end]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'day'] = this.day;
    json[r'start'] = this.start;
    json[r'end'] = this.end;
    return json;
  }

  /// Returns a new [ChatHour] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatHour? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'day'),
            'Required key "ChatHour[day]" is missing from JSON.');
        assert(json[r'day'] != null,
            'Required key "ChatHour[day]" has a null value in JSON.');
        assert(json.containsKey(r'start'),
            'Required key "ChatHour[start]" is missing from JSON.');
        assert(json[r'start'] != null,
            'Required key "ChatHour[start]" has a null value in JSON.');
        assert(json.containsKey(r'end'),
            'Required key "ChatHour[end]" is missing from JSON.');
        assert(json[r'end'] != null,
            'Required key "ChatHour[end]" has a null value in JSON.');
        return true;
      }());

      return ChatHour(
        day: ChatHourDayEnum.fromJson(json[r'day'])!,
        start: mapValueOfType<String>(json, r'start')!,
        end: mapValueOfType<String>(json, r'end')!,
      );
    }
    return null;
  }

  static List<ChatHour> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatHour>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatHour.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatHour> mapFromJson(dynamic json) {
    final map = <String, ChatHour>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatHour.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatHour-objects as value to a dart map
  static Map<String, List<ChatHour>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatHour>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatHour.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'day',
    'start',
    'end',
  };
}

/// FHIR days-of-week.
enum ChatHourDayEnum {
  mon._(r'mon'),
  tue._(r'tue'),
  wed._(r'wed'),
  thu._(r'thu'),
  fri._(r'fri'),
  sat._(r'sat'),
  sun._(r'sun'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatHourDayEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatHourDayEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatHourDayEnum? fromJson(dynamic value) =>
      ChatHourDayEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatHourDayEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatHourDayEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatHourDayEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatHourDayEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatHourDayEnum] to String,
/// and [decode] dynamic data back to [ChatHourDayEnum].
class ChatHourDayEnumTypeTransformer {
  factory ChatHourDayEnumTypeTransformer() =>
      _instance ??= const ChatHourDayEnumTypeTransformer._();

  const ChatHourDayEnumTypeTransformer._();

  String encode(ChatHourDayEnum data) => data._value;

  /// Returns the instance of [ChatHourDayEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatHourDayEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatHourDayEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'mon':
          return ChatHourDayEnum.mon;
        case r'tue':
          return ChatHourDayEnum.tue;
        case r'wed':
          return ChatHourDayEnum.wed;
        case r'thu':
          return ChatHourDayEnum.thu;
        case r'fri':
          return ChatHourDayEnum.fri;
        case r'sat':
          return ChatHourDayEnum.sat;
        case r'sun':
          return ChatHourDayEnum.sun;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatHourDayEnumTypeTransformer? _instance;
}
