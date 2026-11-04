//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class InviteList {
  /// Returns a new [InviteList] instance.
  InviteList({
    this.invites = const [],
  });

  List<Invite> invites;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is InviteList && _deepEquality.equals(other.invites, invites);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (invites.hashCode);

  @override
  String toString() => 'InviteList[invites=$invites]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'invites'] = this.invites;
    return json;
  }

  /// Returns a new [InviteList] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static InviteList? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'invites'),
            'Required key "InviteList[invites]" is missing from JSON.');
        assert(json[r'invites'] != null,
            'Required key "InviteList[invites]" has a null value in JSON.');
        return true;
      }());

      return InviteList(
        invites: Invite.listFromJson(json[r'invites']),
      );
    }
    return null;
  }

  static List<InviteList> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <InviteList>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = InviteList.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, InviteList> mapFromJson(dynamic json) {
    final map = <String, InviteList>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = InviteList.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of InviteList-objects as value to a dart map
  static Map<String, List<InviteList>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<InviteList>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = InviteList.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'invites',
  };
}
