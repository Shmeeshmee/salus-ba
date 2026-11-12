//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class WebhookEventDataMessage {
  /// Returns a new [WebhookEventDataMessage] instance.
  WebhookEventDataMessage({
    required this.sentAt,
    required this.patientId,
    required this.senderRole,
    required this.body,
    required this.seq,
  });

  DateTime sentAt;

  String patientId;

  WebhookEventDataMessageSenderRoleEnum senderRole;

  String body;

  int seq;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is WebhookEventDataMessage &&
          other.sentAt == sentAt &&
          other.patientId == patientId &&
          other.senderRole == senderRole &&
          other.body == body &&
          other.seq == seq;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (sentAt.hashCode) +
      (patientId.hashCode) +
      (senderRole.hashCode) +
      (body.hashCode) +
      (seq.hashCode);

  @override
  String toString() =>
      'WebhookEventDataMessage[sentAt=$sentAt, patientId=$patientId, senderRole=$senderRole, body=$body, seq=$seq]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'sent_at'] = this.sentAt.toUtc().toIso8601String();
    json[r'patient_id'] = this.patientId;
    json[r'sender_role'] = this.senderRole;
    json[r'body'] = this.body;
    json[r'seq'] = this.seq;
    return json;
  }

  /// Returns a new [WebhookEventDataMessage] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static WebhookEventDataMessage? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'sent_at'),
            'Required key "WebhookEventDataMessage[sent_at]" is missing from JSON.');
        assert(json[r'sent_at'] != null,
            'Required key "WebhookEventDataMessage[sent_at]" has a null value in JSON.');
        assert(json.containsKey(r'patient_id'),
            'Required key "WebhookEventDataMessage[patient_id]" is missing from JSON.');
        assert(json[r'patient_id'] != null,
            'Required key "WebhookEventDataMessage[patient_id]" has a null value in JSON.');
        assert(json.containsKey(r'sender_role'),
            'Required key "WebhookEventDataMessage[sender_role]" is missing from JSON.');
        assert(json[r'sender_role'] != null,
            'Required key "WebhookEventDataMessage[sender_role]" has a null value in JSON.');
        assert(json.containsKey(r'body'),
            'Required key "WebhookEventDataMessage[body]" is missing from JSON.');
        assert(json[r'body'] != null,
            'Required key "WebhookEventDataMessage[body]" has a null value in JSON.');
        assert(json.containsKey(r'seq'),
            'Required key "WebhookEventDataMessage[seq]" is missing from JSON.');
        assert(json[r'seq'] != null,
            'Required key "WebhookEventDataMessage[seq]" has a null value in JSON.');
        return true;
      }());

      return WebhookEventDataMessage(
        sentAt: mapDateTime(json, r'sent_at', r'')!,
        patientId: mapValueOfType<String>(json, r'patient_id')!,
        senderRole: WebhookEventDataMessageSenderRoleEnum.fromJson(
            json[r'sender_role'])!,
        body: mapValueOfType<String>(json, r'body')!,
        seq: mapValueOfType<int>(json, r'seq')!,
      );
    }
    return null;
  }

  static List<WebhookEventDataMessage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <WebhookEventDataMessage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = WebhookEventDataMessage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, WebhookEventDataMessage> mapFromJson(dynamic json) {
    final map = <String, WebhookEventDataMessage>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = WebhookEventDataMessage.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of WebhookEventDataMessage-objects as value to a dart map
  static Map<String, List<WebhookEventDataMessage>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<WebhookEventDataMessage>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = WebhookEventDataMessage.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'sent_at',
    'patient_id',
    'sender_role',
    'body',
    'seq',
  };
}

enum WebhookEventDataMessageSenderRoleEnum {
  patient._(r'patient'),
  practitioner._(r'practitioner'),
  ;

  /// Instantiate a new enum with the provided value.
  const WebhookEventDataMessageSenderRoleEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [WebhookEventDataMessageSenderRoleEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static WebhookEventDataMessageSenderRoleEnum? fromJson(dynamic value) =>
      WebhookEventDataMessageSenderRoleEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [WebhookEventDataMessageSenderRoleEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<WebhookEventDataMessageSenderRoleEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <WebhookEventDataMessageSenderRoleEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = WebhookEventDataMessageSenderRoleEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [WebhookEventDataMessageSenderRoleEnum] to String,
/// and [decode] dynamic data back to [WebhookEventDataMessageSenderRoleEnum].
class WebhookEventDataMessageSenderRoleEnumTypeTransformer {
  factory WebhookEventDataMessageSenderRoleEnumTypeTransformer() =>
      _instance ??=
          const WebhookEventDataMessageSenderRoleEnumTypeTransformer._();

  const WebhookEventDataMessageSenderRoleEnumTypeTransformer._();

  String encode(WebhookEventDataMessageSenderRoleEnum data) => data._value;

  /// Returns the instance of [WebhookEventDataMessageSenderRoleEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  WebhookEventDataMessageSenderRoleEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is WebhookEventDataMessageSenderRoleEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'patient':
          return WebhookEventDataMessageSenderRoleEnum.patient;
        case r'practitioner':
          return WebhookEventDataMessageSenderRoleEnum.practitioner;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static WebhookEventDataMessageSenderRoleEnumTypeTransformer? _instance;
}
