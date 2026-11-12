//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatReceipt {
  /// Returns a new [ChatReceipt] instance.
  ChatReceipt({
    required this.participantId,
    required this.seq,
  });

  String participantId;

  int seq;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatReceipt &&
          other.participantId == participantId &&
          other.seq == seq;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (participantId.hashCode) + (seq.hashCode);

  @override
  String toString() => 'ChatReceipt[participantId=$participantId, seq=$seq]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'participant_id'] = this.participantId;
    json[r'seq'] = this.seq;
    return json;
  }

  /// Returns a new [ChatReceipt] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatReceipt? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'participant_id'),
            'Required key "ChatReceipt[participant_id]" is missing from JSON.');
        assert(json[r'participant_id'] != null,
            'Required key "ChatReceipt[participant_id]" has a null value in JSON.');
        assert(json.containsKey(r'seq'),
            'Required key "ChatReceipt[seq]" is missing from JSON.');
        assert(json[r'seq'] != null,
            'Required key "ChatReceipt[seq]" has a null value in JSON.');
        return true;
      }());

      return ChatReceipt(
        participantId: mapValueOfType<String>(json, r'participant_id')!,
        seq: mapValueOfType<int>(json, r'seq')!,
      );
    }
    return null;
  }

  static List<ChatReceipt> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatReceipt>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatReceipt.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatReceipt> mapFromJson(dynamic json) {
    final map = <String, ChatReceipt>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatReceipt.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatReceipt-objects as value to a dart map
  static Map<String, List<ChatReceipt>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatReceipt>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatReceipt.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'participant_id',
    'seq',
  };
}
