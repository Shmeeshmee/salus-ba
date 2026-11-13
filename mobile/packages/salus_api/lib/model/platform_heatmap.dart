//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class PlatformHeatmap {
  /// Returns a new [PlatformHeatmap] instance.
  PlatformHeatmap({
    this.cells = const [],
  });

  List<PlatformHeatmapCellsInner> cells;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PlatformHeatmap && _deepEquality.equals(other.cells, cells);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (cells.hashCode);

  @override
  String toString() => 'PlatformHeatmap[cells=$cells]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'cells'] = this.cells;
    return json;
  }

  /// Returns a new [PlatformHeatmap] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PlatformHeatmap? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'cells'),
            'Required key "PlatformHeatmap[cells]" is missing from JSON.');
        assert(json[r'cells'] != null,
            'Required key "PlatformHeatmap[cells]" has a null value in JSON.');
        return true;
      }());

      return PlatformHeatmap(
        cells: PlatformHeatmapCellsInner.listFromJson(json[r'cells']),
      );
    }
    return null;
  }

  static List<PlatformHeatmap> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PlatformHeatmap>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PlatformHeatmap.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PlatformHeatmap> mapFromJson(dynamic json) {
    final map = <String, PlatformHeatmap>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PlatformHeatmap.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PlatformHeatmap-objects as value to a dart map
  static Map<String, List<PlatformHeatmap>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PlatformHeatmap>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PlatformHeatmap.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'cells',
  };
}
