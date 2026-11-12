//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class MessageRevision {
  /// Returns a new [MessageRevision] instance.
  MessageRevision({
    required this.editedAt,
    required this.body,
    required this.versionId,
  });

  /// When this body was written.
  DateTime editedAt;

  String body;

  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is MessageRevision &&
          other.editedAt == editedAt &&
          other.body == body &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (editedAt.hashCode) + (body.hashCode) + (versionId.hashCode);

  @override
  String toString() =>
      'MessageRevision[editedAt=$editedAt, body=$body, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'edited_at'] = this.editedAt.toUtc().toIso8601String();
    json[r'body'] = this.body;
    json[r'version_id'] = this.versionId;
    return json;
  }

  /// Returns a new [MessageRevision] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static MessageRevision? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'edited_at'),
            'Required key "MessageRevision[edited_at]" is missing from JSON.');
        assert(json[r'edited_at'] != null,
            'Required key "MessageRevision[edited_at]" has a null value in JSON.');
        assert(json.containsKey(r'body'),
            'Required key "MessageRevision[body]" is missing from JSON.');
        assert(json[r'body'] != null,
            'Required key "MessageRevision[body]" has a null value in JSON.');
        assert(json.containsKey(r'version_id'),
            'Required key "MessageRevision[version_id]" is missing from JSON.');
        assert(json[r'version_id'] != null,
            'Required key "MessageRevision[version_id]" has a null value in JSON.');
        return true;
      }());

      return MessageRevision(
        editedAt: mapDateTime(json, r'edited_at', r'')!,
        body: mapValueOfType<String>(json, r'body')!,
        versionId: mapValueOfType<int>(json, r'version_id')!,
      );
    }
    return null;
  }

  static List<MessageRevision> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <MessageRevision>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = MessageRevision.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, MessageRevision> mapFromJson(dynamic json) {
    final map = <String, MessageRevision>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = MessageRevision.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of MessageRevision-objects as value to a dart map
  static Map<String, List<MessageRevision>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<MessageRevision>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = MessageRevision.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'edited_at',
    'body',
    'version_id',
  };
}
