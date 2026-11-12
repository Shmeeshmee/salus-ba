// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for German (`de`).
class AppLocalizationsDe extends AppLocalizations {
  AppLocalizationsDe([String locale = 'de']) : super(locale);

  @override
  String get tryAgain => 'Erneut versuchen';

  @override
  String get language => 'Sprache';

  @override
  String get useDeviceLanguage => 'Sprache des Geräts verwenden';

  @override
  String get languageSaved => 'Sprache gespeichert.';

  @override
  String get somethingWrong =>
      'Etwas ist schiefgelaufen. Versuchen Sie es erneut.';

  @override
  String get unreachable =>
      'Salus ist nicht erreichbar. Prüfen Sie Ihre Verbindung und versuchen Sie es erneut.';

  @override
  String get wrongCredentials => 'E-Mail-Adresse oder Passwort ist falsch.';

  @override
  String get wrongCode =>
      'Der Code ist falsch. Prüfen Sie die Uhrzeit auf Ihrem Gerät und versuchen Sie es erneut.';

  @override
  String get challengeEnded =>
      'Die Anmeldung ist abgelaufen. Geben Sie Ihre E-Mail-Adresse und Ihr Passwort erneut ein.';

  @override
  String get emailTaken =>
      'Für diese E-Mail-Adresse gibt es bereits ein Konto. Melden Sie sich stattdessen an.';

  @override
  String get invalidEmail => 'Geben Sie eine gültige E-Mail-Adresse ein.';

  @override
  String get invalidPassword =>
      'Das Passwort muss 12 bis 128 Zeichen lang sein.';

  @override
  String get missingPassword => 'Geben Sie Ihr Passwort ein.';

  @override
  String get invalidCode => 'Der Code hat 6 Ziffern.';

  @override
  String get staffOnly =>
      'Diese App ist für Patienten gedacht. Klinikpersonal verwendet die Web-App von Salus.';

  @override
  String get accountReady =>
      'Ihr Konto ist eingerichtet. Melden Sie sich an, um fortzufahren.';

  @override
  String get signOutFailed =>
      'Abmelden nicht möglich. Prüfen Sie Ihre Verbindung und versuchen Sie es erneut.';

  @override
  String get tooManyAttemptsLater =>
      'Zu viele Versuche. Versuchen Sie es später erneut.';

  @override
  String tooManyAttempts(int minutes) {
    String _temp0 = intl.Intl.pluralLogic(
      minutes,
      locale: localeName,
      other: 'Zu viele Versuche. Versuchen Sie es in $minutes Minuten erneut.',
      one: 'Zu viele Versuche. Versuchen Sie es in $minutes Minute erneut.',
    );
    return '$_temp0';
  }

  @override
  String get signInTitle => 'Bei Salus anmelden';

  @override
  String get email => 'E-Mail';

  @override
  String get password => 'Passwort';

  @override
  String get signIn => 'Anmelden';

  @override
  String get createAnAccount => 'Konto erstellen';

  @override
  String get codePrompt =>
      'Geben Sie den 6-stelligen Code aus Ihrer Authenticator-App ein.';

  @override
  String get code => 'Code';

  @override
  String get useAnotherAccount => 'Anderes Konto verwenden';

  @override
  String get passwordHint => '12 bis 128 Zeichen';

  @override
  String get createAccount => 'Konto erstellen';

  @override
  String get haveAccount => 'Ich habe bereits ein Konto';

  @override
  String signedInAs(String email) {
    return 'Angemeldet als $email';
  }

  @override
  String get completeProfile => 'Profil vervollständigen';

  @override
  String get completeProfileFirst =>
      'Vervollständigen Sie Ihr Profil, bevor Sie einen Termin buchen.';

  @override
  String get bookAppointment => 'Termin buchen';

  @override
  String get myAppointments => 'Meine Termine';

  @override
  String get myConsents => 'Meine Einwilligungen';

  @override
  String get myProfile => 'Mein Profil';

  @override
  String get signOut => 'Abmelden';

  @override
  String get profileIntro =>
      'Kliniken, bei denen Sie einen Termin buchen, sehen diese Angaben. Die Telefonnummer ist freiwillig; Ihr Behandler nutzt sie nur, um Sie wegen eines Termins zu kontaktieren.';

  @override
  String get givenName => 'Vorname';

  @override
  String get familyName => 'Nachname';

  @override
  String get dateOfBirth => 'Geburtsdatum';

  @override
  String get dateFormatHint => 'JJJJ-MM-TT';

  @override
  String get phoneOptional => 'Telefon (optional)';

  @override
  String phoneHint(String example) {
    return 'Mit Ländervorwahl, zum Beispiel $example';
  }

  @override
  String get save => 'Speichern';

  @override
  String get notNow => 'Nicht jetzt';

  @override
  String get invalidGivenName =>
      'Geben Sie Ihren Vornamen ein, höchstens 100 Zeichen.';

  @override
  String get invalidFamilyName =>
      'Geben Sie Ihren Nachnamen ein, höchstens 100 Zeichen.';

  @override
  String get invalidName =>
      'Geben Sie Ihren Vor- und Nachnamen ein, jeweils höchstens 100 Zeichen.';

  @override
  String get birthDateFormat =>
      'Geben Sie Ihr Geburtsdatum im Format JJJJ-MM-TT ein.';

  @override
  String get birthDateRange =>
      'Geben Sie ein Geburtsdatum in der Vergangenheit ein, frühestens 1900.';

  @override
  String invalidPhone(String example) {
    return 'Geben Sie die Telefonnummer mit Ländervorwahl ein, zum Beispiel $example.';
  }

  @override
  String get profileChanged =>
      'Ihr Profil wurde inzwischen geändert. Prüfen Sie es und speichern Sie erneut.';

  @override
  String get chooseClinic => 'Klinik auswählen';

  @override
  String get noClinics => 'Es sind noch keine Kliniken eingetragen.';

  @override
  String get noPractitioners =>
      'Für diese Klinik sind noch keine Behandler eingetragen.';

  @override
  String get choosePractitioner => 'Wählen Sie einen Behandler.';

  @override
  String get noOpenTimes =>
      'Dieser Behandler hat noch keine Sprechzeiten freigegeben.';

  @override
  String timesIn(String zone) {
    return 'Alle Zeiten in $zone.';
  }

  @override
  String get previousDay => 'Vorheriger Tag';

  @override
  String get nextDay => 'Nächster Tag';

  @override
  String get noFreeTimes => 'An diesem Tag sind keine Zeiten frei.';

  @override
  String get bookThisTime => 'Diese Zeit buchen';

  @override
  String get chooseAnotherTime => 'Andere Zeit wählen';

  @override
  String get book => 'Buchen';

  @override
  String timeRange(String start, String end, String zone) {
    return '$start bis $end ($zone)';
  }

  @override
  String get slotTaken =>
      'Diese Zeit ist nicht mehr frei. Wählen Sie eine andere.';

  @override
  String get consentNeeded =>
      'Für eine Buchung bei dieser Klinik ist Ihre Einwilligung erforderlich.';

  @override
  String get appointmentBooked => 'Ihr Termin ist gebucht.';

  @override
  String get unlistedClinic => 'Eine nicht mehr eingetragene Klinik';

  @override
  String get cancelTitle => 'Diesen Termin stornieren?';

  @override
  String cancelBody(String date, String time) {
    return 'Ihr Termin am $date um $time wird storniert. Sie erhalten eine Bestätigung per E-Mail.';
  }

  @override
  String get keepIt => 'Termin behalten';

  @override
  String get cancelAppointment => 'Termin stornieren';

  @override
  String get noAppointments => 'Sie haben noch keine Termine.';

  @override
  String get upcoming => 'Bevorstehend';

  @override
  String get past => 'Vergangen';

  @override
  String get statusBooked => 'Gebucht';

  @override
  String get statusCancelled => 'Storniert';

  @override
  String get statusAttended => 'Wahrgenommen';

  @override
  String get statusNotAttended => 'Nicht wahrgenommen';

  @override
  String get appointmentCancelled => 'Ihr Termin ist storniert.';

  @override
  String get cancelWindowHint =>
      'Kann bis 24 Stunden vor Beginn in der App storniert werden.';

  @override
  String get cancelWindowClosed =>
      'Dieser Termin beginnt in weniger als 24 Stunden und kann nicht mehr in der App storniert werden.';

  @override
  String get appointmentClosed => 'Dieser Termin ist nicht mehr gebucht.';

  @override
  String consentTitle(String clinic) {
    return 'Ihre Daten mit $clinic teilen';
  }

  @override
  String consentIntro(String clinic) {
    return 'Vor Ihrer ersten Buchung bei $clinic benötigen wir Ihre Einwilligung, Ihre Daten mit dieser Klinik zu teilen.';
  }

  @override
  String get whatIsShared => 'Was geteilt wird';

  @override
  String get whatIsSharedBody =>
      'Ihr Profil (Ihr Name, Ihr Geburtsdatum und, falls angegeben, Ihre Telefonnummer) sowie Ihre Termine bei dieser Klinik.';

  @override
  String get whoSeesIt => 'Wer die Daten sieht';

  @override
  String whoSeesItBody(String clinic) {
    return '$clinic. Der Behandler, bei dem Sie buchen, sieht Ihr Profil; die Administratoren der Klinik sehen Ihren Namen und Ihre Terminzeiten.';
  }

  @override
  String get whatFor => 'Wofür';

  @override
  String get whatForBody =>
      'Ausschließlich für Ihre Behandlung in dieser Klinik: zur Vorbereitung und Durchführung Ihrer Termine.';

  @override
  String get whoIsResponsible => 'Wer verantwortlich ist';

  @override
  String whoIsResponsibleBody(String clinic) {
    return '$clinic ist für Ihre Daten verantwortlich. Salus verarbeitet sie im Auftrag der Klinik.';
  }

  @override
  String get readPrivacyNotice => 'Datenschutzhinweise lesen';

  @override
  String get privacyNoticeFailed =>
      'Die Datenschutzhinweise konnten nicht geöffnet werden.';

  @override
  String get howToRevoke => 'So widerrufen Sie';

  @override
  String get howToRevokeBody =>
      'Sie können diese Einwilligung jederzeit unter „Meine Einwilligungen“ widerrufen. Die Klinik sieht Ihr Profil dann nicht mehr. Ihren Namen und die Angaben zu bereits stattgefundenen Terminen behält sie als Teil ihrer Behandlungsunterlagen. Ihre künftigen Termine dort werden storniert.';

  @override
  String get agreeAndBook => 'Zustimmen und buchen';

  @override
  String get revokeTitle => 'Einwilligung widerrufen?';

  @override
  String revokeBody(String clinic) {
    return '$clinic sieht Ihr Profil dann nicht mehr. Ihren Namen und die Angaben zu bereits stattgefundenen Terminen behält die Klinik als Teil ihrer Behandlungsunterlagen. Ihre künftigen Termine bei $clinic werden storniert.';
  }

  @override
  String get keepConsent => 'Einwilligung behalten';

  @override
  String get revokeConsent => 'Einwilligung widerrufen';

  @override
  String get noConsents => 'Sie haben noch keine Einwilligungen erteilt.';

  @override
  String get consentsIntro =>
      'Mit diesen Kliniken teilen Sie Ihr Profil und Ihre Termine für Ihre Behandlung.';

  @override
  String givenOn(String date) {
    return 'Erteilt am $date';
  }

  @override
  String get consentActive => 'Aktiv';

  @override
  String revokedOn(String date) {
    return 'Widerrufen am $date';
  }

  @override
  String get consentRevoked =>
      'Ihre Einwilligung ist widerrufen. Ihre künftigen Termine bei dieser Klinik werden storniert, und Sie erhalten für jeden eine E-Mail.';

  @override
  String get consentAlreadyRevoked =>
      'Diese Einwilligung wurde bereits widerrufen.';

  @override
  String get joinConsultation => 'Sprechstunde beitreten';

  @override
  String get joinFrom => 'Sie können ab 10 Minuten vor Beginn beitreten.';

  @override
  String get checkTitle => 'Kamera und Mikrofon prüfen';

  @override
  String get checkIntro =>
      'Sobald Sie beitreten, kann Ihr Behandler Sie sehen und hören.';

  @override
  String get noSound =>
      'Kein Ton erkannt. Prüfen Sie, ob Ihr Mikrofon eingeschaltet und nicht stummgeschaltet ist.';

  @override
  String get joinCameraOff => 'Mit ausgeschalteter Kamera beitreten';

  @override
  String get joinMicrophoneOff => 'Mit ausgeschaltetem Mikrofon beitreten';

  @override
  String get joinNow => 'Beitreten';

  @override
  String get joining => 'Verbindung wird hergestellt…';

  @override
  String get cameraDenied =>
      'Salus hat keinen Zugriff auf Ihre Kamera. Sie können den Zugriff in den Einstellungen Ihres Geräts erlauben oder ohne Video teilnehmen.';

  @override
  String get microphoneDenied =>
      'Salus hat keinen Zugriff auf Ihr Mikrofon. Sie können den Zugriff in den Einstellungen Ihres Geräts erlauben. Ohne Mikrofon kann Ihr Behandler Sie nicht hören.';

  @override
  String get openSettings => 'Einstellungen öffnen';

  @override
  String get noCamera => 'Keine Kamera gefunden.';

  @override
  String get noMicrophone => 'Kein Mikrofon gefunden.';

  @override
  String get waiting => 'Warten auf Ihren Behandler';

  @override
  String joinsYouHere(String name) {
    return '$name tritt hier bei. Lassen Sie diesen Bildschirm geöffnet.';
  }

  @override
  String get mute => 'Mikrofon stummschalten';

  @override
  String get unmute => 'Stummschaltung aufheben';

  @override
  String get cameraOff => 'Kamera ausschalten';

  @override
  String get cameraOn => 'Kamera einschalten';

  @override
  String get leave => 'Verlassen';

  @override
  String peerCameraOff(String name) {
    return 'Die Kamera von $name ist ausgeschaltet';
  }

  @override
  String peerMuted(String name) {
    return '$name ist stummgeschaltet';
  }

  @override
  String peerLeft(String name) {
    return '$name hat die Sprechstunde verlassen und kann bis zu ihrem Ende wieder beitreten.';
  }

  @override
  String get reconnecting =>
      'Verbindung unterbrochen. Verbindung wird wiederhergestellt…';

  @override
  String get notRestored =>
      'Die Verbindung konnte nicht wiederhergestellt werden.';

  @override
  String get reconnected => 'Wieder verbunden';

  @override
  String get consultationOver => 'Die Sprechstunde ist beendet';

  @override
  String get closeScreen => 'Sie können diesen Bildschirm schließen.';

  @override
  String get timeLimit =>
      'Die Sprechstunde hat ihre maximale Dauer erreicht und wurde beendet.';

  @override
  String get practitionerAbsent => 'Ihr Behandler ist nicht beigetreten';

  @override
  String get timePassed =>
      'Die Zeit für den Beitritt zu dieser Sprechstunde ist abgelaufen. Die Klinik kann sehen, dass Sie beigetreten sind. Die Klinik meldet sich möglicherweise bei Ihnen, oder Sie buchen einen neuen Termin.';

  @override
  String get withdrewConsent =>
      'Sie haben Ihre Einwilligung für diese Klinik widerrufen und deshalb die Sprechstunde verlassen.';

  @override
  String get appointmentWasCancelled =>
      'Dieser Termin wurde storniert, daher ist die Sprechstunde beendet.';

  @override
  String get windowNotOpen =>
      'Diese Sprechstunde öffnet 10 Minuten vor Beginn.';

  @override
  String get consultationEnded => 'Diese Sprechstunde ist beendet.';

  @override
  String get joinNeedsConsent =>
      'Für die Teilnahme ist Ihre Einwilligung für diese Klinik erforderlich.';

  @override
  String get consultationNotAvailable =>
      'Diese Sprechstunde ist nicht verfügbar.';

  @override
  String get cannotConnect =>
      'Die Videoverbindung konnte nicht hergestellt werden. Prüfen Sie Ihre Verbindung und versuchen Sie es erneut.';

  @override
  String get whyCameraAndMicrophone =>
      'Für die Teilnahme an der Videosprechstunde benötigt Salus Zugriff auf Ihre Kamera und Ihr Mikrofon. Die Sprechstunde wird weder als Ton noch als Video aufgezeichnet. Wenn Sie einwilligen, wird das Gesprochene transkribiert und übersetzt.';

  @override
  String get transcriptionOn => 'Transkription ist aktiv';

  @override
  String get transcriptionTitle => 'Transkription und Übersetzung';

  @override
  String transcriptionIntro(String clinic) {
    return 'In Ihren Videosprechstunden bei $clinic kann das, was Sie und Ihr Behandler sagen, beim Sprechen transkribiert und übersetzt werden, damit Ihr Behandler Ihnen in Ihrer Sprache folgen kann.';
  }

  @override
  String get transcriptionAssistance =>
      'Aus dem Transkript erhält Ihr Behandler außerdem KI-Unterstützung: Vorschläge während der Sprechstunde und Entwürfe von Dokumenten. Es sind nur Vorschläge; Ihr Behandler prüft sie und trifft jede Entscheidung selbst.';

  @override
  String transcriptionKept(String clinic) {
    return 'Das Transkript wird mit Ihren Behandlungsunterlagen bei $clinic aufbewahrt. Die Sprechstunde wird weder als Ton noch als Video aufgezeichnet.';
  }

  @override
  String get transcriptionDecline =>
      'Wenn Sie nicht einwilligen, findet die Sprechstunde wie gewohnt statt, ohne Transkription, Übersetzung und KI-Unterstützung.';

  @override
  String get transcriptionWithdraw =>
      'Sie können Ihre Einwilligung jederzeit widerrufen, auch während der Sprechstunde: Tippen Sie auf „Transkription ist aktiv“ oder öffnen Sie Meine Einwilligungen. Die Transkription endet sofort; was bis dahin transkribiert wurde, bleibt in Ihren Unterlagen.';

  @override
  String get transcriptionAgree => 'Einwilligen und fortfahren';

  @override
  String get transcriptionWithout => 'Ohne Transkription fortfahren';

  @override
  String get stopTranscriptionTitle => 'Transkription beenden?';

  @override
  String stopTranscriptionBody(String clinic) {
    return 'Transkription, Übersetzung und KI-Unterstützung enden sofort in Ihren Sprechstunden bei $clinic. Eine laufende Sprechstunde geht ohne sie weiter, und Ihre Termine bleiben bestehen. Was bis jetzt transkribiert wurde, bleibt in Ihren Unterlagen.';
  }

  @override
  String get keepTranscription => 'Transkription behalten';

  @override
  String get stopTranscription => 'Transkription beenden';

  @override
  String get transcriptionStopped =>
      'Die Transkription ist aus. Ihre Sprechstunden bei dieser Klinik finden ohne sie statt.';

  @override
  String get consentForCare => 'Weitergabe für die Behandlung';

  @override
  String get consentForTranscription => 'Transkription und KI-Unterstützung';

  @override
  String get toFillIn => 'Auszufüllen';

  @override
  String get inProgress => 'In Bearbeitung';

  @override
  String get submittedStatus => 'Abgesendet';

  @override
  String get closedStatus => 'Geschlossen';

  @override
  String get fillIn => 'Ausfüllen';

  @override
  String get continueFilling => 'Weiter ausfüllen';

  @override
  String get viewAnswers => 'Antworten ansehen';

  @override
  String get submitBeforeStart =>
      'Bitte senden Sie ihn vor Beginn Ihres Termins ab.';

  @override
  String get questionnaireClosed =>
      'Der Termin hat begonnen, daher kann dieser Fragebogen nicht mehr ausgefüllt werden.';

  @override
  String savedAsYouGo(String clinic) {
    return 'Ihre Antworten werden laufend gespeichert. $clinic sieht sie erst, wenn Sie sie abgesendet haben.';
  }

  @override
  String get saved => 'Gespeichert';

  @override
  String get saving => 'Wird gespeichert…';

  @override
  String get notSaved =>
      'Nicht gespeichert. Prüfen Sie Ihre Verbindung. Das Speichern wird erneut versucht.';

  @override
  String get requiredQuestion => 'Erforderlich';

  @override
  String get moreInformation => 'Weitere Informationen';

  @override
  String get yourAnswer => 'Ihre Antwort';

  @override
  String get chooseDate => 'Datum auswählen';

  @override
  String get selectOne => 'Wählen Sie eine Option.';

  @override
  String get selectAll => 'Wählen Sie alle zutreffenden Optionen.';

  @override
  String get yes => 'Ja';

  @override
  String get no => 'Nein';

  @override
  String get questionRequired => 'Diese Frage muss beantwortet werden.';

  @override
  String get enterWholeNumber => 'Geben Sie eine ganze Zahl ein.';

  @override
  String get enterNumber => 'Geben Sie eine Zahl ein.';

  @override
  String get numberTooLarge => 'Diese Zahl ist zu groß.';

  @override
  String upToCharacters(int max) {
    return 'Geben Sie höchstens $max Zeichen ein.';
  }

  @override
  String get enterValidDate => 'Geben Sie ein gültiges Datum ein.';

  @override
  String get submit => 'Absenden';

  @override
  String get submitTitle => 'Antworten absenden?';

  @override
  String submitBody(String clinic) {
    return 'Nach dem Absenden können Sie sie nicht mehr ändern. Ihr Behandler bei $clinic wird sie sehen.';
  }

  @override
  String get cancel => 'Abbrechen';

  @override
  String get requiredMissing => 'Einige Pflichtfragen sind nicht beantwortet.';

  @override
  String get answersSubmitted => 'Antworten abgesendet';

  @override
  String get answersSubmittedBody =>
      'Ihr Behandler kann sie vor Ihrem Termin einsehen. Sie finden sie unter „Meine Termine“.';

  @override
  String get yourAnswers => 'Ihre Antworten';

  @override
  String submittedAt(String when) {
    return 'Abgesendet am $when';
  }

  @override
  String dateAndTime(String date, String time) {
    return '$date, $time';
  }

  @override
  String get notAnswered => 'Nicht beantwortet';

  @override
  String get leaveUnsaved =>
      'Ihre letzte Antwort ist noch nicht gespeichert. Trotzdem verlassen?';

  @override
  String get stay => 'Bleiben';

  @override
  String get fillingNeedsConsent =>
      'Zum Ausfüllen dieses Fragebogens ist Ihre Einwilligung für diese Klinik erforderlich.';

  @override
  String get pastDue =>
      'Der Termin hat begonnen, daher kann dieser Fragebogen nicht mehr geändert werden.';

  @override
  String get withdrawnByClinic =>
      'Die Klinik hat diesen Fragebogen zurückgezogen. Sie müssen ihn nicht ausfüllen.';

  @override
  String get alreadySubmitted =>
      'Sie haben diesen Fragebogen bereits abgesendet.';

  @override
  String get changedElsewhere =>
      'Diese Antworten wurden auf einem anderen Gerät geändert. Angezeigt werden die neuesten Antworten.';

  @override
  String get questionnaireNotAvailable =>
      'Dieser Fragebogen ist nicht verfügbar.';

  @override
  String get done => 'Fertig';

  @override
  String get addFile => 'Datei hinzufügen';

  @override
  String get photosUpTo => 'Fotos bis 20 MB.';

  @override
  String get documentsUpTo => 'Dokumente (PDF) bis 20 MB.';

  @override
  String get videosUpTo => 'Videos bis 200 MB.';

  @override
  String get oneFileHint => '1 Datei.';

  @override
  String upToFilesHint(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Bis zu $n Dateien.',
      one: 'Bis zu $n Datei.',
    );
    return '$_temp0';
  }

  @override
  String get takePhoto => 'Foto aufnehmen';

  @override
  String get recordVideo => 'Video aufnehmen';

  @override
  String get chooseFromPhotos => 'Aus Fotos auswählen';

  @override
  String get chooseDocument => 'Dokument auswählen';

  @override
  String get preparing => 'Wird vorbereitet…';

  @override
  String uploadingPercent(int n) {
    return 'Wird hochgeladen… $n %';
  }

  @override
  String get checkingFile => 'Datei wird geprüft…';

  @override
  String get added => 'Hinzugefügt';

  @override
  String get notUploaded => 'Nicht hochgeladen. Prüfen Sie Ihre Verbindung.';

  @override
  String get remove => 'Entfernen';

  @override
  String get removeFile => 'Datei entfernen';

  @override
  String get removeFileTitle => 'Diese Datei entfernen?';

  @override
  String get removeFileBody => 'Sie wird aus Ihren Antworten gelöscht.';

  @override
  String get fileRejected =>
      'Diese Datei hat die Sicherheitsprüfung nicht bestanden und wurde gelöscht. Wählen Sie eine andere Datei.';

  @override
  String fileTooLarge(int max) {
    return 'Diese Datei ist größer als $max MB.';
  }

  @override
  String get fileTypeRefused =>
      'Dieser Dateityp kann zu dieser Frage nicht hinzugefügt werden.';

  @override
  String tooManyFiles(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Sie können zu dieser Frage höchstens $n Dateien hinzufügen.',
      one: 'Sie können zu dieser Frage höchstens $n Datei hinzufügen.',
    );
    return '$_temp0';
  }

  @override
  String get filesFull =>
      'Die Dateien in diesem Fragebogen haben 500 MB erreicht. Entfernen Sie eine Datei, um eine weitere hinzuzufügen.';

  @override
  String get fileNotUploaded =>
      'Die Datei konnte nicht hochgeladen werden. Versuchen Sie es erneut.';

  @override
  String get cannotPrepareFile =>
      'Diese Datei konnte nicht für das Hochladen vorbereitet werden. Wählen Sie eine andere Datei.';

  @override
  String get filesBeingChecked =>
      'Einige Dateien werden noch geprüft. Sie können absenden, sobald sie bereit sind.';

  @override
  String get leaveUploading =>
      'Eine Datei wird noch hochgeladen. Trotzdem verlassen? Das Hochladen wird abgebrochen.';

  @override
  String fileCount(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n Dateien',
      one: '$n Datei',
    );
    return '$_temp0';
  }

  @override
  String get photoKind => 'Foto';

  @override
  String get documentKind => 'Dokument';

  @override
  String get videoKind => 'Video';

  @override
  String kilobytes(int n) {
    return '$n KB';
  }

  @override
  String megabytes(String mb) {
    return '$mb MB';
  }

  @override
  String kindAndSize(String kind, String size) {
    return '$kind, $size';
  }

  @override
  String uploadingLabel(String title, int n) {
    return '$title wird hochgeladen, $n Prozent';
  }

  @override
  String fileLabel(String kind, String title, String status) {
    return '$kind, $title, $status';
  }

  @override
  String get allowCamera => 'Kamerazugriff erlauben';

  @override
  String get whyCameraForFiles =>
      'Salus benötigt die Kamera, um das Foto oder Video aufzunehmen, das Sie diesem Fragebogen hinzufügen.';

  @override
  String get continueLabel => 'Weiter';

  @override
  String get cameraDeniedForFiles =>
      'Salus hat keinen Zugriff auf Ihre Kamera. Sie können den Zugriff in den Einstellungen Ihres Geräts erlauben oder stattdessen eine Datei auswählen.';

  @override
  String get photosDenied =>
      'Salus hat keinen Zugriff auf Ihre Fotos. Sie können den Zugriff in den Einstellungen Ihres Geräts erlauben.';

  @override
  String cameraPhotoTitle(String date, String time) {
    return 'Foto, $date, $time';
  }

  @override
  String cameraVideoTitle(String date, String time) {
    return 'Video, $date, $time';
  }

  @override
  String get specialty394814009 => 'Allgemeinmedizin';

  @override
  String get specialty419192003 => 'Innere Medizin';

  @override
  String get specialty394579002 => 'Kardiologie';

  @override
  String get specialty394582007 => 'Dermatologie';

  @override
  String get specialty394586005 => 'Gynäkologie';

  @override
  String get specialty394537008 => 'Kinderheilkunde';

  @override
  String get specialty394587001 => 'Psychiatrie';

  @override
  String get specialty394591006 => 'Neurologie';

  @override
  String get specialty418960008 => 'HNO-Heilkunde';

  @override
  String get specialty394594003 => 'Augenheilkunde';

  @override
  String get specialty394801008 => 'Orthopädie';

  @override
  String get specialty394583002 => 'Endokrinologie';

  @override
  String questionnaireNotInLanguage(String language) {
    return 'Dieser Fragebogen ist nicht auf $language verfügbar.';
  }

  @override
  String questionnaireLanguages(String languages) {
    return 'Er ist in diesen Sprachen verfügbar: $languages. Wählen Sie die Sprache, die Sie am besten verstehen.';
  }

  @override
  String continueInLanguage(String language) {
    return 'Weiter auf $language';
  }

  @override
  String get languageUnavailable =>
      'Diese Sprache ist für diesen Fragebogen nicht mehr verfügbar. Wählen Sie eine andere.';

  @override
  String specialtyAtClinic(String specialty, String clinic) {
    return '$specialty, $clinic';
  }

  @override
  String get documents => 'Dokumente';

  @override
  String get noDocuments =>
      'Für diesen Termin gibt es noch keine unterschriebenen Dokumente. Ein Dokument erscheint hier, sobald Ihr Behandler es unterschrieben hat.';

  @override
  String documentSignedOn(String date) {
    return 'Unterschrieben am $date';
  }

  @override
  String documentLanguage(String language) {
    return 'Sprache: $language';
  }

  @override
  String get sharePdf => 'PDF speichern oder teilen';

  @override
  String get transcript => 'Transkript';

  @override
  String get transcriptNote =>
      'Die Sprechstunde, wie sie automatisch transkribiert und übersetzt wurde. Sie kann Fehler enthalten.';

  @override
  String get transcriptEmpty =>
      'In dieser Sprechstunde wurde nichts transkribiert.';

  @override
  String get speakerYou => 'Sie';

  @override
  String get speakerPractitioner => 'Behandler';

  @override
  String get shareTranscript => 'Transkript teilen';

  @override
  String transcriptTranslation(String text) {
    return 'Übersetzung: $text';
  }

  @override
  String get termExplanationNote => 'Zur Information, keine Diagnose';

  @override
  String get termNotReviewed => 'Noch nicht geprüft';

  @override
  String get termClose => 'Schließen';

  @override
  String get allergies => 'Allergien';

  @override
  String get currentMedications => 'Aktuelle Medikamente';

  @override
  String get allergiesIntro =>
      'Kliniken, denen Sie Ihre Einwilligung erteilt haben, sehen die Allergien in Ihrem Profil, wenn Sie dort einen Termin haben.';

  @override
  String get medicationsIntro =>
      'Kliniken, denen Sie Ihre Einwilligung erteilt haben, sehen die Medikamente in Ihrem Profil, wenn Sie dort einen Termin haben.';

  @override
  String get noAllergies => 'Sie haben noch keine Allergien hinzugefügt.';

  @override
  String get noMedications => 'Sie haben noch keine Medikamente hinzugefügt.';

  @override
  String get addAllergy => 'Allergie hinzufügen';

  @override
  String get addMedication => 'Medikament hinzufügen';

  @override
  String get changeAllergy => 'Allergie ändern';

  @override
  String get changeMedication => 'Medikament ändern';

  @override
  String get change => 'Ändern';

  @override
  String get allergySubstance => 'Stoff oder Lebensmittel';

  @override
  String get medicationName => 'Medikament';

  @override
  String get termSearchHint =>
      'Tippen Sie, um zu suchen, oder schreiben Sie es in eigenen Worten.';

  @override
  String get allergyCategory => 'Art (optional)';

  @override
  String get notSpecified => 'Keine Angabe';

  @override
  String get categoryFood => 'Lebensmittel';

  @override
  String get categoryMedication => 'Medikament';

  @override
  String get categoryEnvironment => 'Umwelt';

  @override
  String get categoryBiologic => 'Biologisches Produkt';

  @override
  String get reactionOptional => 'Reaktion (optional)';

  @override
  String get severity => 'Schweregrad';

  @override
  String get severityMild => 'Leicht';

  @override
  String get severityModerate => 'Mittel';

  @override
  String get severitySevere => 'Schwer';

  @override
  String get entryStatus => 'Status';

  @override
  String get allergyActive => 'Aktiv';

  @override
  String get allergyInactive => 'Inaktiv';

  @override
  String get allergyResolved => 'Abgeklungen';

  @override
  String reactionLine(String reaction) {
    return 'Reaktion: $reaction';
  }

  @override
  String reactionWithSeverity(String reaction, String severity) {
    return 'Reaktion: $reaction ($severity)';
  }

  @override
  String get confirmedByClinician => 'Ärztlich bestätigt';

  @override
  String get recordedByClinic =>
      'Von einer Klinik bei einem Ihrer Termine erfasst';

  @override
  String get confirmedChangeNote =>
      'Wenn Sie diesen Eintrag ändern, ist er nicht mehr ärztlich bestätigt.';

  @override
  String get nameRequired =>
      'Geben Sie einen Namen mit höchstens 200 Zeichen ein.';

  @override
  String get addToProfile => 'Zu meinem Profil hinzufügen';

  @override
  String get addToProfileTitle => 'Zu Ihrem Profil hinzufügen?';

  @override
  String get addToProfileBody =>
      'Dann sieht ihn jede Klinik, der Sie Ihre Einwilligung erteilt haben, ohne den Namen der Klinik, die ihn erfasst hat.';

  @override
  String get addedToProfile => 'Der Eintrag ist jetzt in Ihrem Profil.';

  @override
  String get removeEntryTitle => 'Diesen Eintrag entfernen?';

  @override
  String get removeEntryBody =>
      'Er wird weder Ihnen noch einer Klinik mehr angezeigt. Hat eine Klinik ihn erfasst oder bestätigt, bleibt er in Ihrer Patientenakte, als entfernt markiert.';

  @override
  String get entryRemoved => 'Der Eintrag ist entfernt.';

  @override
  String get entrySaved => 'In Ihrem Profil gespeichert.';

  @override
  String get entryChanged =>
      'Dieser Eintrag wurde inzwischen geändert. Angezeigt wird die neueste Fassung.';

  @override
  String get entryInvalid =>
      'Einige Angaben sind ungültig. Prüfen Sie sie und versuchen Sie es erneut.';

  @override
  String get entryNotAllowed => 'Sie können diesen Eintrag nicht ändern.';

  @override
  String get entryNotFound => 'Dieser Eintrag ist nicht mehr verfügbar.';

  @override
  String get doseHeading => 'Dosis (optional)';

  @override
  String get doseAmount => 'Menge';

  @override
  String get doseUnit => 'Einheit';

  @override
  String get unitMg => 'Milligramm (mg)';

  @override
  String get unitG => 'Gramm (g)';

  @override
  String get unitUg => 'Mikrogramm (µg)';

  @override
  String get unitMl => 'Milliliter (ml)';

  @override
  String get unitIu => 'Internationale Einheiten (I.E.)';

  @override
  String get unitTablet => 'Tabletten';

  @override
  String get unitCapsule => 'Kapseln';

  @override
  String get unitDrop => 'Tropfen';

  @override
  String get unitPuff => 'Hübe (Inhalator)';

  @override
  String get unitPatch => 'Pflaster';

  @override
  String get doseBoth =>
      'Geben Sie Menge und Einheit an oder keines von beiden.';

  @override
  String get numberAboveZero => 'Geben Sie eine Zahl über 0 ein.';

  @override
  String get howOftenHeading => 'Wie oft (optional)';

  @override
  String get timesLabel => 'Mal';

  @override
  String get everyLabel => 'Alle';

  @override
  String get periodUnitLabel => 'Zeiteinheit';

  @override
  String get periodHours => 'Stunden';

  @override
  String get periodDays => 'Tage';

  @override
  String get periodWeeks => 'Wochen';

  @override
  String get periodMonths => 'Monate';

  @override
  String get howOftenAll =>
      'Füllen Sie alle drei Felder aus oder lassen Sie sie leer.';

  @override
  String get asNeeded => 'Bei Bedarf';

  @override
  String get medicationActive => 'Nehme ich derzeit';

  @override
  String get medicationStopped => 'Abgesetzt';

  @override
  String doseMg(String value) {
    return '$value mg';
  }

  @override
  String doseG(String value) {
    return '$value g';
  }

  @override
  String doseUg(String value) {
    return '$value µg';
  }

  @override
  String doseMl(String value) {
    return '$value ml';
  }

  @override
  String doseIu(String value) {
    return '$value I.E.';
  }

  @override
  String doseTablet(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value Tabletten',
      one: '$value Tablette',
    );
    return '$_temp0';
  }

  @override
  String doseCapsule(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value Kapseln',
      one: '$value Kapsel',
    );
    return '$_temp0';
  }

  @override
  String doseDrop(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value Tropfen',
      one: '$value Tropfen',
    );
    return '$_temp0';
  }

  @override
  String dosePuff(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value Hübe',
      one: '$value Hub',
    );
    return '$_temp0';
  }

  @override
  String dosePatch(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value Pflaster',
      one: '$value Pflaster',
    );
    return '$_temp0';
  }

  @override
  String doseLine(String dose) {
    return 'Dosis: $dose';
  }

  @override
  String howOftenPerHour(int frequency) {
    return '$frequency× pro Stunde';
  }

  @override
  String howOftenPerDay(int frequency) {
    return '$frequency× täglich';
  }

  @override
  String howOftenPerWeek(int frequency) {
    return '$frequency× pro Woche';
  }

  @override
  String howOftenPerMonth(int frequency) {
    return '$frequency× pro Monat';
  }

  @override
  String howOftenEveryHours(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× alle $every Stunden',
      one: '$frequency× alle $every Stunde',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryDays(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× alle $every Tage',
      one: '$frequency× alle $every Tag',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryWeeks(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× alle $every Wochen',
      one: '$frequency× alle $every Woche',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryMonths(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× alle $every Monate',
      one: '$frequency× alle $every Monat',
    );
    return '$_temp0';
  }
}
