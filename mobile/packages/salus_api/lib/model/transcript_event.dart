//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class TranscriptEvent {
  /// Returns a new [TranscriptEvent] instance.
  TranscriptEvent({
    required this.encounterId,
    required this.segmentId,
    required this.speaker,
    required this.lang,
    required this.original,
    required this.translation,
    required this.isFinal,
    required this.tsStart,
    required this.tsEnd,
  });

  String encounterId;

  String segmentId;

  TranscriptEventSpeakerEnum speaker;

  /// The language the segment was spoken in, a BCP 47 tag.
  String lang;

  String original;

  /// Into the other participant's language; empty until it is translated.
  String translation;

  bool isFinal;

  DateTime tsStart;

  DateTime tsEnd;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TranscriptEvent &&
          other.encounterId == encounterId &&
          other.segmentId == segmentId &&
          other.speaker == speaker &&
          other.lang == lang &&
          other.original == original &&
          other.translation == translation &&
          other.isFinal == isFinal &&
          other.tsStart == tsStart &&
          other.tsEnd == tsEnd;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (encounterId.hashCode) +
      (segmentId.hashCode) +
      (speaker.hashCode) +
      (lang.hashCode) +
      (original.hashCode) +
      (translation.hashCode) +
      (isFinal.hashCode) +
      (tsStart.hashCode) +
      (tsEnd.hashCode);

  @override
  String toString() =>
      'TranscriptEvent[encounterId=$encounterId, segmentId=$segmentId, speaker=$speaker, lang=$lang, original=$original, translation=$translation, isFinal=$isFinal, tsStart=$tsStart, tsEnd=$tsEnd]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'encounter_id'] = this.encounterId;
    json[r'segment_id'] = this.segmentId;
    json[r'speaker'] = this.speaker;
    json[r'lang'] = this.lang;
    json[r'original'] = this.original;
    json[r'translation'] = this.translation;
    json[r'is_final'] = this.isFinal;
    json[r'ts_start'] = this.tsStart.toUtc().toIso8601String();
    json[r'ts_end'] = this.tsEnd.toUtc().toIso8601String();
    return json;
  }

  /// Returns a new [TranscriptEvent] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TranscriptEvent? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'encounter_id'),
            'Required key "TranscriptEvent[encounter_id]" is missing from JSON.');
        assert(json[r'encounter_id'] != null,
            'Required key "TranscriptEvent[encounter_id]" has a null value in JSON.');
        assert(json.containsKey(r'segment_id'),
            'Required key "TranscriptEvent[segment_id]" is missing from JSON.');
        assert(json[r'segment_id'] != null,
            'Required key "TranscriptEvent[segment_id]" has a null value in JSON.');
        assert(json.containsKey(r'speaker'),
            'Required key "TranscriptEvent[speaker]" is missing from JSON.');
        assert(json[r'speaker'] != null,
            'Required key "TranscriptEvent[speaker]" has a null value in JSON.');
        assert(json.containsKey(r'lang'),
            'Required key "TranscriptEvent[lang]" is missing from JSON.');
        assert(json[r'lang'] != null,
            'Required key "TranscriptEvent[lang]" has a null value in JSON.');
        assert(json.containsKey(r'original'),
            'Required key "TranscriptEvent[original]" is missing from JSON.');
        assert(json[r'original'] != null,
            'Required key "TranscriptEvent[original]" has a null value in JSON.');
        assert(json.containsKey(r'translation'),
            'Required key "TranscriptEvent[translation]" is missing from JSON.');
        assert(json[r'translation'] != null,
            'Required key "TranscriptEvent[translation]" has a null value in JSON.');
        assert(json.containsKey(r'is_final'),
            'Required key "TranscriptEvent[is_final]" is missing from JSON.');
        assert(json[r'is_final'] != null,
            'Required key "TranscriptEvent[is_final]" has a null value in JSON.');
        assert(json.containsKey(r'ts_start'),
            'Required key "TranscriptEvent[ts_start]" is missing from JSON.');
        assert(json[r'ts_start'] != null,
            'Required key "TranscriptEvent[ts_start]" has a null value in JSON.');
        assert(json.containsKey(r'ts_end'),
            'Required key "TranscriptEvent[ts_end]" is missing from JSON.');
        assert(json[r'ts_end'] != null,
            'Required key "TranscriptEvent[ts_end]" has a null value in JSON.');
        return true;
      }());

      return TranscriptEvent(
        encounterId: mapValueOfType<String>(json, r'encounter_id')!,
        segmentId: mapValueOfType<String>(json, r'segment_id')!,
        speaker: TranscriptEventSpeakerEnum.fromJson(json[r'speaker'])!,
        lang: mapValueOfType<String>(json, r'lang')!,
        original: mapValueOfType<String>(json, r'original')!,
        translation: mapValueOfType<String>(json, r'translation')!,
        isFinal: mapValueOfType<bool>(json, r'is_final')!,
        tsStart: mapDateTime(json, r'ts_start', r'')!,
        tsEnd: mapDateTime(json, r'ts_end', r'')!,
      );
    }
    return null;
  }

  static List<TranscriptEvent> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TranscriptEvent>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TranscriptEvent.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TranscriptEvent> mapFromJson(dynamic json) {
    final map = <String, TranscriptEvent>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TranscriptEvent.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TranscriptEvent-objects as value to a dart map
  static Map<String, List<TranscriptEvent>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<TranscriptEvent>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TranscriptEvent.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'encounter_id',
    'segment_id',
    'speaker',
    'lang',
    'original',
    'translation',
    'is_final',
    'ts_start',
    'ts_end',
  };
}

enum TranscriptEventSpeakerEnum {
  practitioner._(r'practitioner'),
  patient._(r'patient'),
  ;

  /// Instantiate a new enum with the provided value.
  const TranscriptEventSpeakerEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [TranscriptEventSpeakerEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static TranscriptEventSpeakerEnum? fromJson(dynamic value) =>
      TranscriptEventSpeakerEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [TranscriptEventSpeakerEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<TranscriptEventSpeakerEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TranscriptEventSpeakerEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TranscriptEventSpeakerEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [TranscriptEventSpeakerEnum] to String,
/// and [decode] dynamic data back to [TranscriptEventSpeakerEnum].
class TranscriptEventSpeakerEnumTypeTransformer {
  factory TranscriptEventSpeakerEnumTypeTransformer() =>
      _instance ??= const TranscriptEventSpeakerEnumTypeTransformer._();

  const TranscriptEventSpeakerEnumTypeTransformer._();

  String encode(TranscriptEventSpeakerEnum data) => data._value;

  /// Returns the instance of [TranscriptEventSpeakerEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  TranscriptEventSpeakerEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is TranscriptEventSpeakerEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'practitioner':
          return TranscriptEventSpeakerEnum.practitioner;
        case r'patient':
          return TranscriptEventSpeakerEnum.patient;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static TranscriptEventSpeakerEnumTypeTransformer? _instance;
}
