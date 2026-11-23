//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ServiceImportRow {
  /// Returns a new [ServiceImportRow] instance.
  ServiceImportRow({
    required this.line,
    this.service,
    this.errors = const [],
  });

  /// The row's line in the file
  int line;

  /// The service the row would store; absent for a row with errors.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  SaveServiceRequest? service;

  /// Codes of what is wrong with the row; empty when it is valid.
  List<ServiceImportRowErrorsEnum> errors;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ServiceImportRow &&
          other.line == line &&
          other.service == service &&
          _deepEquality.equals(other.errors, errors);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (line.hashCode) +
      (service == null ? 0 : service!.hashCode) +
      (errors.hashCode);

  @override
  String toString() =>
      'ServiceImportRow[line=$line, service=$service, errors=$errors]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'line'] = this.line;
    if (this.service != null) {
      json[r'service'] = this.service;
    } else {
      json[r'service'] = null;
    }
    json[r'errors'] = this.errors;
    return json;
  }

  /// Returns a new [ServiceImportRow] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ServiceImportRow? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'line'),
            'Required key "ServiceImportRow[line]" is missing from JSON.');
        assert(json[r'line'] != null,
            'Required key "ServiceImportRow[line]" has a null value in JSON.');
        assert(json.containsKey(r'errors'),
            'Required key "ServiceImportRow[errors]" is missing from JSON.');
        assert(json[r'errors'] != null,
            'Required key "ServiceImportRow[errors]" has a null value in JSON.');
        return true;
      }());

      return ServiceImportRow(
        line: mapValueOfType<int>(json, r'line')!,
        service: SaveServiceRequest.fromJson(json[r'service']),
        errors: ServiceImportRowErrorsEnum.listFromJson(json[r'errors']),
      );
    }
    return null;
  }

  static List<ServiceImportRow> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ServiceImportRow>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ServiceImportRow.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ServiceImportRow> mapFromJson(dynamic json) {
    final map = <String, ServiceImportRow>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ServiceImportRow.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ServiceImportRow-objects as value to a dart map
  static Map<String, List<ServiceImportRow>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ServiceImportRow>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ServiceImportRow.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'line',
    'errors',
  };
}

enum ServiceImportRowErrorsEnum {
  nameMissing._(r'name_missing'),
  nameInvalid._(r'name_invalid'),
  codeInvalid._(r'code_invalid'),
  priceInvalid._(r'price_invalid'),
  durationInvalid._(r'duration_invalid'),
  vatInvalid._(r'vat_invalid'),
  modeInvalid._(r'mode_invalid'),
  duplicate._(r'duplicate'),
  ;

  /// Instantiate a new enum with the provided value.
  const ServiceImportRowErrorsEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ServiceImportRowErrorsEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ServiceImportRowErrorsEnum? fromJson(dynamic value) =>
      ServiceImportRowErrorsEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ServiceImportRowErrorsEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ServiceImportRowErrorsEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ServiceImportRowErrorsEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ServiceImportRowErrorsEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ServiceImportRowErrorsEnum] to String,
/// and [decode] dynamic data back to [ServiceImportRowErrorsEnum].
class ServiceImportRowErrorsEnumTypeTransformer {
  factory ServiceImportRowErrorsEnumTypeTransformer() =>
      _instance ??= const ServiceImportRowErrorsEnumTypeTransformer._();

  const ServiceImportRowErrorsEnumTypeTransformer._();

  String encode(ServiceImportRowErrorsEnum data) => data._value;

  /// Returns the instance of [ServiceImportRowErrorsEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ServiceImportRowErrorsEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ServiceImportRowErrorsEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'name_missing':
          return ServiceImportRowErrorsEnum.nameMissing;
        case r'name_invalid':
          return ServiceImportRowErrorsEnum.nameInvalid;
        case r'code_invalid':
          return ServiceImportRowErrorsEnum.codeInvalid;
        case r'price_invalid':
          return ServiceImportRowErrorsEnum.priceInvalid;
        case r'duration_invalid':
          return ServiceImportRowErrorsEnum.durationInvalid;
        case r'vat_invalid':
          return ServiceImportRowErrorsEnum.vatInvalid;
        case r'mode_invalid':
          return ServiceImportRowErrorsEnum.modeInvalid;
        case r'duplicate':
          return ServiceImportRowErrorsEnum.duplicate;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ServiceImportRowErrorsEnumTypeTransformer? _instance;
}
