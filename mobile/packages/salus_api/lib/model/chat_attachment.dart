//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatAttachment {
  /// Returns a new [ChatAttachment] instance.
  ChatAttachment({
    required this.id,
    required this.contentType,
    required this.size,
    required this.filename,
    required this.status,
    required this.thumbnail,
  });

  String id;

  /// Detected from the bytes.
  ChatAttachmentContentTypeEnum contentType;

  /// Bytes.
  int size;

  /// The name it was uploaded with.
  String filename;

  ChatAttachmentStatusEnum status;

  /// An image's thumbnail is there to read (`getChatAttachmentThumbnail`).
  bool thumbnail;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatAttachment &&
          other.id == id &&
          other.contentType == contentType &&
          other.size == size &&
          other.filename == filename &&
          other.status == status &&
          other.thumbnail == thumbnail;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (contentType.hashCode) +
      (size.hashCode) +
      (filename.hashCode) +
      (status.hashCode) +
      (thumbnail.hashCode);

  @override
  String toString() =>
      'ChatAttachment[id=$id, contentType=$contentType, size=$size, filename=$filename, status=$status, thumbnail=$thumbnail]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'content_type'] = this.contentType;
    json[r'size'] = this.size;
    json[r'filename'] = this.filename;
    json[r'status'] = this.status;
    json[r'thumbnail'] = this.thumbnail;
    return json;
  }

  /// Returns a new [ChatAttachment] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatAttachment? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "ChatAttachment[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "ChatAttachment[id]" has a null value in JSON.');
        assert(json.containsKey(r'content_type'),
            'Required key "ChatAttachment[content_type]" is missing from JSON.');
        assert(json[r'content_type'] != null,
            'Required key "ChatAttachment[content_type]" has a null value in JSON.');
        assert(json.containsKey(r'size'),
            'Required key "ChatAttachment[size]" is missing from JSON.');
        assert(json[r'size'] != null,
            'Required key "ChatAttachment[size]" has a null value in JSON.');
        assert(json.containsKey(r'filename'),
            'Required key "ChatAttachment[filename]" is missing from JSON.');
        assert(json[r'filename'] != null,
            'Required key "ChatAttachment[filename]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "ChatAttachment[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "ChatAttachment[status]" has a null value in JSON.');
        assert(json.containsKey(r'thumbnail'),
            'Required key "ChatAttachment[thumbnail]" is missing from JSON.');
        assert(json[r'thumbnail'] != null,
            'Required key "ChatAttachment[thumbnail]" has a null value in JSON.');
        return true;
      }());

      return ChatAttachment(
        id: mapValueOfType<String>(json, r'id')!,
        contentType:
            ChatAttachmentContentTypeEnum.fromJson(json[r'content_type'])!,
        size: mapValueOfType<int>(json, r'size')!,
        filename: mapValueOfType<String>(json, r'filename')!,
        status: ChatAttachmentStatusEnum.fromJson(json[r'status'])!,
        thumbnail: mapValueOfType<bool>(json, r'thumbnail')!,
      );
    }
    return null;
  }

  static List<ChatAttachment> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatAttachment>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatAttachment.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatAttachment> mapFromJson(dynamic json) {
    final map = <String, ChatAttachment>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatAttachment.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatAttachment-objects as value to a dart map
  static Map<String, List<ChatAttachment>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatAttachment>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatAttachment.listFromJson(
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
    'content_type',
    'size',
    'filename',
    'status',
    'thumbnail',
  };
}

/// Detected from the bytes.
enum ChatAttachmentContentTypeEnum {
  applicationSlashPdf._(r'application/pdf'),
  imageSlashJpeg._(r'image/jpeg'),
  imageSlashPng._(r'image/png'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatAttachmentContentTypeEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatAttachmentContentTypeEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatAttachmentContentTypeEnum? fromJson(dynamic value) =>
      ChatAttachmentContentTypeEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatAttachmentContentTypeEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatAttachmentContentTypeEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatAttachmentContentTypeEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatAttachmentContentTypeEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatAttachmentContentTypeEnum] to String,
/// and [decode] dynamic data back to [ChatAttachmentContentTypeEnum].
class ChatAttachmentContentTypeEnumTypeTransformer {
  factory ChatAttachmentContentTypeEnumTypeTransformer() =>
      _instance ??= const ChatAttachmentContentTypeEnumTypeTransformer._();

  const ChatAttachmentContentTypeEnumTypeTransformer._();

  String encode(ChatAttachmentContentTypeEnum data) => data._value;

  /// Returns the instance of [ChatAttachmentContentTypeEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatAttachmentContentTypeEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatAttachmentContentTypeEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'application/pdf':
          return ChatAttachmentContentTypeEnum.applicationSlashPdf;
        case r'image/jpeg':
          return ChatAttachmentContentTypeEnum.imageSlashJpeg;
        case r'image/png':
          return ChatAttachmentContentTypeEnum.imageSlashPng;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatAttachmentContentTypeEnumTypeTransformer? _instance;
}

enum ChatAttachmentStatusEnum {
  scanning._(r'scanning'),
  clean._(r'clean'),
  rejected._(r'rejected'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatAttachmentStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatAttachmentStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatAttachmentStatusEnum? fromJson(dynamic value) =>
      ChatAttachmentStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatAttachmentStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatAttachmentStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatAttachmentStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatAttachmentStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatAttachmentStatusEnum] to String,
/// and [decode] dynamic data back to [ChatAttachmentStatusEnum].
class ChatAttachmentStatusEnumTypeTransformer {
  factory ChatAttachmentStatusEnumTypeTransformer() =>
      _instance ??= const ChatAttachmentStatusEnumTypeTransformer._();

  const ChatAttachmentStatusEnumTypeTransformer._();

  String encode(ChatAttachmentStatusEnum data) => data._value;

  /// Returns the instance of [ChatAttachmentStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatAttachmentStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatAttachmentStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'scanning':
          return ChatAttachmentStatusEnum.scanning;
        case r'clean':
          return ChatAttachmentStatusEnum.clean;
        case r'rejected':
          return ChatAttachmentStatusEnum.rejected;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatAttachmentStatusEnumTypeTransformer? _instance;
}
