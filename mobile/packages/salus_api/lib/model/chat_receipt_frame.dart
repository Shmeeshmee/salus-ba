//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatReceiptFrame {
  /// Returns a new [ChatReceiptFrame] instance.
  ChatReceiptFrame({
    required this.type,
    required this.version,
    required this.payload,
  });

  ChatReceiptFrameTypeEnum type;

  ChatReceiptFrameVersionEnum version;

  ChatReceipt payload;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatReceiptFrame &&
          other.type == type &&
          other.version == version &&
          other.payload == payload;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (type.hashCode) + (version.hashCode) + (payload.hashCode);

  @override
  String toString() =>
      'ChatReceiptFrame[type=$type, version=$version, payload=$payload]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'type'] = this.type;
    json[r'version'] = this.version;
    json[r'payload'] = this.payload;
    return json;
  }

  /// Returns a new [ChatReceiptFrame] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatReceiptFrame? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'type'),
            'Required key "ChatReceiptFrame[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "ChatReceiptFrame[type]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "ChatReceiptFrame[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "ChatReceiptFrame[version]" has a null value in JSON.');
        assert(json.containsKey(r'payload'),
            'Required key "ChatReceiptFrame[payload]" is missing from JSON.');
        assert(json[r'payload'] != null,
            'Required key "ChatReceiptFrame[payload]" has a null value in JSON.');
        return true;
      }());

      return ChatReceiptFrame(
        type: ChatReceiptFrameTypeEnum.fromJson(json[r'type'])!,
        version: ChatReceiptFrameVersionEnum.fromJson(json[r'version'])!,
        payload: ChatReceipt.fromJson(json[r'payload'])!,
      );
    }
    return null;
  }

  static List<ChatReceiptFrame> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatReceiptFrame>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatReceiptFrame.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatReceiptFrame> mapFromJson(dynamic json) {
    final map = <String, ChatReceiptFrame>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatReceiptFrame.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatReceiptFrame-objects as value to a dart map
  static Map<String, List<ChatReceiptFrame>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatReceiptFrame>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatReceiptFrame.listFromJson(
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
    'version',
    'payload',
  };
}

enum ChatReceiptFrameTypeEnum {
  receipt._(r'receipt'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatReceiptFrameTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatReceiptFrameTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatReceiptFrameTypeEnum? fromJson(dynamic value) =>
      ChatReceiptFrameTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatReceiptFrameTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatReceiptFrameTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatReceiptFrameTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatReceiptFrameTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatReceiptFrameTypeEnum] to String,
/// and [decode] dynamic data back to [ChatReceiptFrameTypeEnum].
class ChatReceiptFrameTypeEnumTypeTransformer {
  factory ChatReceiptFrameTypeEnumTypeTransformer() =>
      _instance ??= const ChatReceiptFrameTypeEnumTypeTransformer._();

  const ChatReceiptFrameTypeEnumTypeTransformer._();

  String encode(ChatReceiptFrameTypeEnum data) => data._value;

  /// Returns the instance of [ChatReceiptFrameTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatReceiptFrameTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatReceiptFrameTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'receipt':
          return ChatReceiptFrameTypeEnum.receipt;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatReceiptFrameTypeEnumTypeTransformer? _instance;
}

enum ChatReceiptFrameVersionEnum {
  number1._(1),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatReceiptFrameVersionEnum._(this._value);

  /// The underlying value of this enum member.
  final int _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  int toJson() => _value;

  /// Returns the instance of [ChatReceiptFrameVersionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatReceiptFrameVersionEnum? fromJson(dynamic value) =>
      ChatReceiptFrameVersionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatReceiptFrameVersionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatReceiptFrameVersionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatReceiptFrameVersionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatReceiptFrameVersionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatReceiptFrameVersionEnum] to int,
/// and [decode] dynamic data back to [ChatReceiptFrameVersionEnum].
class ChatReceiptFrameVersionEnumTypeTransformer {
  factory ChatReceiptFrameVersionEnumTypeTransformer() =>
      _instance ??= const ChatReceiptFrameVersionEnumTypeTransformer._();

  const ChatReceiptFrameVersionEnumTypeTransformer._();

  int encode(ChatReceiptFrameVersionEnum data) => data._value;

  /// Returns the instance of [ChatReceiptFrameVersionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatReceiptFrameVersionEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatReceiptFrameVersionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 1:
          return ChatReceiptFrameVersionEnum.number1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatReceiptFrameVersionEnumTypeTransformer? _instance;
}
