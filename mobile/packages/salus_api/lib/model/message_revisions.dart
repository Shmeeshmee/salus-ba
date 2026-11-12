//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class MessageRevisions {
  /// Returns a new [MessageRevisions] instance.
  MessageRevisions({
    this.revisions = const [],
  });

  List<MessageRevision> revisions;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is MessageRevisions &&
          _deepEquality.equals(other.revisions, revisions);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (revisions.hashCode);

  @override
  String toString() => 'MessageRevisions[revisions=$revisions]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'revisions'] = this.revisions;
    return json;
  }

  /// Returns a new [MessageRevisions] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static MessageRevisions? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'revisions'),
            'Required key "MessageRevisions[revisions]" is missing from JSON.');
        assert(json[r'revisions'] != null,
            'Required key "MessageRevisions[revisions]" has a null value in JSON.');
        return true;
      }());

      return MessageRevisions(
        revisions: MessageRevision.listFromJson(json[r'revisions']),
      );
    }
    return null;
  }

  static List<MessageRevisions> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <MessageRevisions>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = MessageRevisions.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, MessageRevisions> mapFromJson(dynamic json) {
    final map = <String, MessageRevisions>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = MessageRevisions.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of MessageRevisions-objects as value to a dart map
  static Map<String, List<MessageRevisions>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<MessageRevisions>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = MessageRevisions.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'revisions',
  };
}
