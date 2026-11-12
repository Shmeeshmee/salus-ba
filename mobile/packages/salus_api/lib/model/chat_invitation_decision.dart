//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatInvitationDecision {
  /// Returns a new [ChatInvitationDecision] instance.
  ChatInvitationDecision({
    required this.decision,
  });

  ChatInvitationDecisionDecisionEnum decision;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatInvitationDecision && other.decision == decision;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (decision.hashCode);

  @override
  String toString() => 'ChatInvitationDecision[decision=$decision]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'decision'] = this.decision;
    return json;
  }

  /// Returns a new [ChatInvitationDecision] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatInvitationDecision? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'decision'),
            'Required key "ChatInvitationDecision[decision]" is missing from JSON.');
        assert(json[r'decision'] != null,
            'Required key "ChatInvitationDecision[decision]" has a null value in JSON.');
        return true;
      }());

      return ChatInvitationDecision(
        decision:
            ChatInvitationDecisionDecisionEnum.fromJson(json[r'decision'])!,
      );
    }
    return null;
  }

  static List<ChatInvitationDecision> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatInvitationDecision>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatInvitationDecision.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatInvitationDecision> mapFromJson(dynamic json) {
    final map = <String, ChatInvitationDecision>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatInvitationDecision.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatInvitationDecision-objects as value to a dart map
  static Map<String, List<ChatInvitationDecision>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatInvitationDecision>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatInvitationDecision.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'decision',
  };
}

enum ChatInvitationDecisionDecisionEnum {
  grant._(r'grant'),
  decline._(r'decline'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatInvitationDecisionDecisionEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatInvitationDecisionDecisionEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatInvitationDecisionDecisionEnum? fromJson(dynamic value) =>
      ChatInvitationDecisionDecisionEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatInvitationDecisionDecisionEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatInvitationDecisionDecisionEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatInvitationDecisionDecisionEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatInvitationDecisionDecisionEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatInvitationDecisionDecisionEnum] to String,
/// and [decode] dynamic data back to [ChatInvitationDecisionDecisionEnum].
class ChatInvitationDecisionDecisionEnumTypeTransformer {
  factory ChatInvitationDecisionDecisionEnumTypeTransformer() =>
      _instance ??= const ChatInvitationDecisionDecisionEnumTypeTransformer._();

  const ChatInvitationDecisionDecisionEnumTypeTransformer._();

  String encode(ChatInvitationDecisionDecisionEnum data) => data._value;

  /// Returns the instance of [ChatInvitationDecisionDecisionEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatInvitationDecisionDecisionEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is ChatInvitationDecisionDecisionEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'grant':
          return ChatInvitationDecisionDecisionEnum.grant;
        case r'decline':
          return ChatInvitationDecisionDecisionEnum.decline;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatInvitationDecisionDecisionEnumTypeTransformer? _instance;
}
