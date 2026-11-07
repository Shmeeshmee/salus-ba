//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Binary {
  /// Returns a new [Binary] instance.
  Binary({
    required this.id,
    required this.contentType,
    required this.size,
    required this.title,
    required this.status,
  });

  String id;

  /// Detected from the bytes on upload.
  BinaryContentTypeEnum contentType;

  /// Bytes.
  int size;

  /// The file's name as uploaded; may be empty.
  String title;

  /// `quarantined` until the malware scan is done; `clean` may be opened; `rejected` was found to hold malware and is removed from the answer.
  BinaryStatusEnum status;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Binary &&
          other.id == id &&
          other.contentType == contentType &&
          other.size == size &&
          other.title == title &&
          other.status == status;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (contentType.hashCode) +
      (size.hashCode) +
      (title.hashCode) +
      (status.hashCode);

  @override
  String toString() =>
      'Binary[id=$id, contentType=$contentType, size=$size, title=$title, status=$status]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'contentType'] = this.contentType;
    json[r'size'] = this.size;
    json[r'title'] = this.title;
    json[r'status'] = this.status;
    return json;
  }

  /// Returns a new [Binary] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Binary? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Binary[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Binary[id]" has a null value in JSON.');
        assert(json.containsKey(r'contentType'),
            'Required key "Binary[contentType]" is missing from JSON.');
        assert(json[r'contentType'] != null,
            'Required key "Binary[contentType]" has a null value in JSON.');
        assert(json.containsKey(r'size'),
            'Required key "Binary[size]" is missing from JSON.');
        assert(json[r'size'] != null,
            'Required key "Binary[size]" has a null value in JSON.');
        assert(json.containsKey(r'title'),
            'Required key "Binary[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "Binary[title]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "Binary[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "Binary[status]" has a null value in JSON.');
        return true;
      }());

      return Binary(
        id: mapValueOfType<String>(json, r'id')!,
        contentType: BinaryContentTypeEnum.fromJson(json[r'contentType'])!,
        size: mapValueOfType<int>(json, r'size')!,
        title: mapValueOfType<String>(json, r'title')!,
        status: BinaryStatusEnum.fromJson(json[r'status'])!,
      );
    }
    return null;
  }

  static List<Binary> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Binary>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Binary.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Binary> mapFromJson(dynamic json) {
    final map = <String, Binary>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Binary.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Binary-objects as value to a dart map
  static Map<String, List<Binary>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Binary>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Binary.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'id',
    'contentType',
    'size',
    'title',
    'status',
  };
}

/// Detected from the bytes on upload.
enum BinaryContentTypeEnum {
  applicationSlashPdf._(r'application/pdf'),
  imageSlashJpeg._(r'image/jpeg'),
  imageSlashPng._(r'image/png'),
  videoSlashMp4._(r'video/mp4'),
  ;

  /// Instantiate a new enum with the provided value.
  const BinaryContentTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [BinaryContentTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static BinaryContentTypeEnum? fromJson(dynamic value) =>
      BinaryContentTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [BinaryContentTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<BinaryContentTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <BinaryContentTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = BinaryContentTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [BinaryContentTypeEnum] to String,
/// and [decode] dynamic data back to [BinaryContentTypeEnum].
class BinaryContentTypeEnumTypeTransformer {
  factory BinaryContentTypeEnumTypeTransformer() =>
      _instance ??= const BinaryContentTypeEnumTypeTransformer._();

  const BinaryContentTypeEnumTypeTransformer._();

  String encode(BinaryContentTypeEnum data) => data._value;

  /// Returns the instance of [BinaryContentTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  BinaryContentTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is BinaryContentTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'application/pdf':
          return BinaryContentTypeEnum.applicationSlashPdf;
        case r'image/jpeg':
          return BinaryContentTypeEnum.imageSlashJpeg;
        case r'image/png':
          return BinaryContentTypeEnum.imageSlashPng;
        case r'video/mp4':
          return BinaryContentTypeEnum.videoSlashMp4;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static BinaryContentTypeEnumTypeTransformer? _instance;
}

/// `quarantined` until the malware scan is done; `clean` may be opened; `rejected` was found to hold malware and is removed from the answer.
enum BinaryStatusEnum {
  quarantined._(r'quarantined'),
  clean._(r'clean'),
  rejected._(r'rejected'),
  ;

  /// Instantiate a new enum with the provided value.
  const BinaryStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [BinaryStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static BinaryStatusEnum? fromJson(dynamic value) =>
      BinaryStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [BinaryStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<BinaryStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <BinaryStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = BinaryStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [BinaryStatusEnum] to String,
/// and [decode] dynamic data back to [BinaryStatusEnum].
class BinaryStatusEnumTypeTransformer {
  factory BinaryStatusEnumTypeTransformer() =>
      _instance ??= const BinaryStatusEnumTypeTransformer._();

  const BinaryStatusEnumTypeTransformer._();

  String encode(BinaryStatusEnum data) => data._value;

  /// Returns the instance of [BinaryStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  BinaryStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is BinaryStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'quarantined':
          return BinaryStatusEnum.quarantined;
        case r'clean':
          return BinaryStatusEnum.clean;
        case r'rejected':
          return BinaryStatusEnum.rejected;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static BinaryStatusEnumTypeTransformer? _instance;
}
