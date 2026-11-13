import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_ar.dart';
import 'app_localizations_bs.dart';
import 'app_localizations_de.dart';
import 'app_localizations_en.dart';
import 'app_localizations_fr.dart';
import 'app_localizations_hr.dart';
import 'app_localizations_it.dart';
import 'app_localizations_sr.dart';
import 'app_localizations_tr.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('ar'),
    Locale('bs'),
    Locale('de'),
    Locale('en'),
    Locale('fr'),
    Locale('hr'),
    Locale('it'),
    Locale('sr'),
    Locale.fromSubtags(languageCode: 'sr', scriptCode: 'Cyrl'),
    Locale.fromSubtags(languageCode: 'sr', scriptCode: 'Latn'),
    Locale('tr'),
  ];

  /// Button: try once more after an action failed (M20).
  ///
  /// In en, this message translates to:
  /// **'Try again'**
  String get tryAgain;

  /// The language picker: its button tooltip and its title (i18n L2).
  ///
  /// In en, this message translates to:
  /// **'Language'**
  String get language;

  /// The picker's first option: follow the phone's language again (i18n L3).
  ///
  /// In en, this message translates to:
  /// **'Use the device\'s language'**
  String get useDeviceLanguage;

  /// After a change while signed in (i18n L4).
  ///
  /// In en, this message translates to:
  /// **'Language saved.'**
  String get languageSaved;

  /// Any failure without a sentence of its own.
  ///
  /// In en, this message translates to:
  /// **'Something went wrong. Try again.'**
  String get somethingWrong;

  /// The API could not be reached.
  ///
  /// In en, this message translates to:
  /// **'Salus could not be reached. Check your connection and try again.'**
  String get unreachable;

  /// Sign-in refused.
  ///
  /// In en, this message translates to:
  /// **'The e-mail or password is wrong.'**
  String get wrongCredentials;

  /// Wrong authenticator code.
  ///
  /// In en, this message translates to:
  /// **'The code is wrong. Check the time on your device and try again.'**
  String get wrongCode;

  /// The two-step sign-in expired.
  ///
  /// In en, this message translates to:
  /// **'The sign-in has ended. Enter your e-mail and password again.'**
  String get challengeEnded;

  /// Registration with a used e-mail.
  ///
  /// In en, this message translates to:
  /// **'This e-mail already has an account. Sign in instead.'**
  String get emailTaken;

  /// Field error.
  ///
  /// In en, this message translates to:
  /// **'Enter a valid e-mail address.'**
  String get invalidEmail;

  /// Field error.
  ///
  /// In en, this message translates to:
  /// **'Use 12 to 128 characters for the password.'**
  String get invalidPassword;

  /// Field error.
  ///
  /// In en, this message translates to:
  /// **'Enter your password.'**
  String get missingPassword;

  /// Field error: the authenticator code.
  ///
  /// In en, this message translates to:
  /// **'The code has 6 digits.'**
  String get invalidCode;

  /// A staff account signed in to the patient app.
  ///
  /// In en, this message translates to:
  /// **'This app is for patients. Clinic staff use the Salus web app.'**
  String get staffOnly;

  /// After registration when signing in did not finish.
  ///
  /// In en, this message translates to:
  /// **'Your account is ready. Sign in to continue.'**
  String get accountReady;

  /// Sign-out failed.
  ///
  /// In en, this message translates to:
  /// **'Could not sign out. Check your connection and try again.'**
  String get signOutFailed;

  /// Rate limited, no wait time known.
  ///
  /// In en, this message translates to:
  /// **'Too many attempts. Try again later.'**
  String get tooManyAttemptsLater;

  /// Rate limited; minutes to wait.
  ///
  /// In en, this message translates to:
  /// **'{minutes, plural, one{Too many attempts. Try again in {minutes} minute.} other{Too many attempts. Try again in {minutes} minutes.}}'**
  String tooManyAttempts(int minutes);

  /// Sign-in screen title.
  ///
  /// In en, this message translates to:
  /// **'Sign in to Salus'**
  String get signInTitle;

  /// Field label.
  ///
  /// In en, this message translates to:
  /// **'E-mail'**
  String get email;

  /// Field label.
  ///
  /// In en, this message translates to:
  /// **'Password'**
  String get password;

  /// Button.
  ///
  /// In en, this message translates to:
  /// **'Sign in'**
  String get signIn;

  /// Link on sign-in, and the registration screen title.
  ///
  /// In en, this message translates to:
  /// **'Create an account'**
  String get createAnAccount;

  /// Second sign-in step.
  ///
  /// In en, this message translates to:
  /// **'Enter the 6-digit code from your authenticator app.'**
  String get codePrompt;

  /// Field label: the authenticator code.
  ///
  /// In en, this message translates to:
  /// **'Code'**
  String get code;

  /// Link back to the first sign-in step.
  ///
  /// In en, this message translates to:
  /// **'Use another account'**
  String get useAnotherAccount;

  /// Helper under the new password field.
  ///
  /// In en, this message translates to:
  /// **'12 to 128 characters'**
  String get passwordHint;

  /// Registration button.
  ///
  /// In en, this message translates to:
  /// **'Create account'**
  String get createAccount;

  /// Link from registration to sign-in.
  ///
  /// In en, this message translates to:
  /// **'I already have an account'**
  String get haveAccount;

  /// Home screen; the e-mail address.
  ///
  /// In en, this message translates to:
  /// **'Signed in as {email}'**
  String signedInAs(String email);

  /// Button and profile screen title.
  ///
  /// In en, this message translates to:
  /// **'Complete your profile'**
  String get completeProfile;

  /// Home screen reminder.
  ///
  /// In en, this message translates to:
  /// **'Complete your profile before you book.'**
  String get completeProfileFirst;

  /// Home button.
  ///
  /// In en, this message translates to:
  /// **'Book an appointment'**
  String get bookAppointment;

  /// Home button and screen title.
  ///
  /// In en, this message translates to:
  /// **'My appointments'**
  String get myAppointments;

  /// Home button and screen title.
  ///
  /// In en, this message translates to:
  /// **'My consents'**
  String get myConsents;

  /// Home button and screen title.
  ///
  /// In en, this message translates to:
  /// **'My profile'**
  String get myProfile;

  /// Home link.
  ///
  /// In en, this message translates to:
  /// **'Sign out'**
  String get signOut;

  /// Profile screen, when creating it.
  ///
  /// In en, this message translates to:
  /// **'Clinics you book with see these details. The phone number is optional; the practitioner uses it only to reach you about an appointment.'**
  String get profileIntro;

  /// Field label.
  ///
  /// In en, this message translates to:
  /// **'Given name'**
  String get givenName;

  /// Field label.
  ///
  /// In en, this message translates to:
  /// **'Family name'**
  String get familyName;

  /// Field label.
  ///
  /// In en, this message translates to:
  /// **'Date of birth'**
  String get dateOfBirth;

  /// Helper under the date of birth: the format typed (year, month, day).
  ///
  /// In en, this message translates to:
  /// **'YYYY-MM-DD'**
  String get dateFormatHint;

  /// Field label.
  ///
  /// In en, this message translates to:
  /// **'Phone (optional)'**
  String get phoneOptional;

  /// Helper under the phone field. example is a phone number in international form, written by the app.
  ///
  /// In en, this message translates to:
  /// **'With the country code, for example {example}'**
  String phoneHint(String example);

  /// Button.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get save;

  /// Button: leave it for later (profile, consent, camera explanation M18).
  ///
  /// In en, this message translates to:
  /// **'Not now'**
  String get notNow;

  /// Field error.
  ///
  /// In en, this message translates to:
  /// **'Enter your given name, up to 100 characters.'**
  String get invalidGivenName;

  /// Field error.
  ///
  /// In en, this message translates to:
  /// **'Enter your family name, up to 100 characters.'**
  String get invalidFamilyName;

  /// Error from the API.
  ///
  /// In en, this message translates to:
  /// **'Enter your given and family name, up to 100 characters each.'**
  String get invalidName;

  /// Field error.
  ///
  /// In en, this message translates to:
  /// **'Enter your date of birth as YYYY-MM-DD.'**
  String get birthDateFormat;

  /// Field error.
  ///
  /// In en, this message translates to:
  /// **'Enter a date of birth in the past, from 1900 on.'**
  String get birthDateRange;

  /// Field error. example is a phone number in international form, written by the app.
  ///
  /// In en, this message translates to:
  /// **'Enter the phone number with its country code, for example {example}.'**
  String invalidPhone(String example);

  /// Save conflict.
  ///
  /// In en, this message translates to:
  /// **'Your profile was changed in the meantime. Check it and save again.'**
  String get profileChanged;

  /// Screen title.
  ///
  /// In en, this message translates to:
  /// **'Choose a clinic'**
  String get chooseClinic;

  /// Empty list.
  ///
  /// In en, this message translates to:
  /// **'No clinics are listed yet.'**
  String get noClinics;

  /// Empty list.
  ///
  /// In en, this message translates to:
  /// **'This clinic has no practitioners listed yet.'**
  String get noPractitioners;

  /// Above the list of practitioners.
  ///
  /// In en, this message translates to:
  /// **'Choose a practitioner.'**
  String get choosePractitioner;

  /// No schedule.
  ///
  /// In en, this message translates to:
  /// **'This practitioner has no open times yet.'**
  String get noOpenTimes;

  /// The time zone, an IANA name such as Europe/Sarajevo.
  ///
  /// In en, this message translates to:
  /// **'Times are in {zone}.'**
  String timesIn(String zone);

  /// Button.
  ///
  /// In en, this message translates to:
  /// **'Previous day'**
  String get previousDay;

  /// Button.
  ///
  /// In en, this message translates to:
  /// **'Next day'**
  String get nextDay;

  /// Empty day.
  ///
  /// In en, this message translates to:
  /// **'No free times on this day.'**
  String get noFreeTimes;

  /// Screen title.
  ///
  /// In en, this message translates to:
  /// **'Book this time'**
  String get bookThisTime;

  /// Link after the time was taken.
  ///
  /// In en, this message translates to:
  /// **'Choose another time'**
  String get chooseAnotherTime;

  /// Button: book the time.
  ///
  /// In en, this message translates to:
  /// **'Book'**
  String get book;

  /// An appointment's times, 24-hour, and the time zone.
  ///
  /// In en, this message translates to:
  /// **'{start} to {end} ({zone})'**
  String timeRange(String start, String end, String zone);

  /// Booking refused.
  ///
  /// In en, this message translates to:
  /// **'This time is no longer free. Choose another time.'**
  String get slotTaken;

  /// The patient did not agree.
  ///
  /// In en, this message translates to:
  /// **'Booking with this clinic needs your consent.'**
  String get consentNeeded;

  /// After booking.
  ///
  /// In en, this message translates to:
  /// **'Your appointment is booked.'**
  String get appointmentBooked;

  /// In place of a clinic name.
  ///
  /// In en, this message translates to:
  /// **'A clinic no longer listed'**
  String get unlistedClinic;

  /// Dialog title.
  ///
  /// In en, this message translates to:
  /// **'Cancel this appointment?'**
  String get cancelTitle;

  /// [legal] Dialog text: the effect of a cancellation.
  ///
  /// In en, this message translates to:
  /// **'Your appointment on {date} at {time} will be cancelled. You will get an e-mail to confirm it.'**
  String cancelBody(String date, String time);

  /// Dialog button: do not cancel.
  ///
  /// In en, this message translates to:
  /// **'Keep it'**
  String get keepIt;

  /// Button.
  ///
  /// In en, this message translates to:
  /// **'Cancel appointment'**
  String get cancelAppointment;

  /// Empty list.
  ///
  /// In en, this message translates to:
  /// **'You have no appointments yet.'**
  String get noAppointments;

  /// Heading.
  ///
  /// In en, this message translates to:
  /// **'Upcoming'**
  String get upcoming;

  /// Heading.
  ///
  /// In en, this message translates to:
  /// **'Past'**
  String get past;

  /// An appointment's status.
  ///
  /// In en, this message translates to:
  /// **'Booked'**
  String get statusBooked;

  /// An appointment's status.
  ///
  /// In en, this message translates to:
  /// **'Cancelled'**
  String get statusCancelled;

  /// An appointment's status.
  ///
  /// In en, this message translates to:
  /// **'Attended'**
  String get statusAttended;

  /// An appointment's status.
  ///
  /// In en, this message translates to:
  /// **'Not attended'**
  String get statusNotAttended;

  /// After cancelling.
  ///
  /// In en, this message translates to:
  /// **'Your appointment is cancelled.'**
  String get appointmentCancelled;

  /// [legal] Under a booked appointment: the cancellation rule.
  ///
  /// In en, this message translates to:
  /// **'Can be cancelled in the app until 24 hours before the start.'**
  String get cancelWindowHint;

  /// [legal] Cancellation refused: the rule.
  ///
  /// In en, this message translates to:
  /// **'This appointment starts in less than 24 hours and can no longer be cancelled in the app.'**
  String get cancelWindowClosed;

  /// Refused: the appointment changed.
  ///
  /// In en, this message translates to:
  /// **'This appointment is no longer booked.'**
  String get appointmentClosed;

  /// [legal] Consent screen title; the clinic's name.
  ///
  /// In en, this message translates to:
  /// **'Share your details with {clinic}'**
  String consentTitle(String clinic);

  /// [legal] Consent screen.
  ///
  /// In en, this message translates to:
  /// **'Before your first booking with {clinic}, we need your consent to share your details with it.'**
  String consentIntro(String clinic);

  /// [legal] Consent heading.
  ///
  /// In en, this message translates to:
  /// **'What is shared'**
  String get whatIsShared;

  /// [legal] Consent screen.
  ///
  /// In en, this message translates to:
  /// **'Your profile (your name, date of birth and, if you gave one, your phone number) and your appointments with this clinic.'**
  String get whatIsSharedBody;

  /// [legal] Consent heading.
  ///
  /// In en, this message translates to:
  /// **'Who sees it'**
  String get whoSeesIt;

  /// [legal] Consent screen.
  ///
  /// In en, this message translates to:
  /// **'{clinic}. The practitioner you book with sees your profile; the clinic\'s administrators see your name and appointment times.'**
  String whoSeesItBody(String clinic);

  /// [legal] Consent heading.
  ///
  /// In en, this message translates to:
  /// **'What for'**
  String get whatFor;

  /// [legal] Consent screen.
  ///
  /// In en, this message translates to:
  /// **'Only for your care at this clinic: to prepare for and hold your appointments.'**
  String get whatForBody;

  /// [legal] Consent heading.
  ///
  /// In en, this message translates to:
  /// **'Who is responsible'**
  String get whoIsResponsible;

  /// [legal] Consent screen.
  ///
  /// In en, this message translates to:
  /// **'{clinic} is responsible for your data. Salus processes it on the clinic\'s behalf.'**
  String whoIsResponsibleBody(String clinic);

  /// [legal] Link to the privacy notice.
  ///
  /// In en, this message translates to:
  /// **'Read the privacy notice'**
  String get readPrivacyNotice;

  /// The link could not be opened.
  ///
  /// In en, this message translates to:
  /// **'The privacy notice could not be opened.'**
  String get privacyNoticeFailed;

  /// [legal] Consent heading.
  ///
  /// In en, this message translates to:
  /// **'How to revoke'**
  String get howToRevoke;

  /// [legal] Consent screen: the effect of revoking.
  ///
  /// In en, this message translates to:
  /// **'You can revoke this consent at any time under My consents. The clinic then no longer sees your profile. It keeps your name and the details of appointments that have already taken place, as part of its care records. Your future appointments there are cancelled.'**
  String get howToRevokeBody;

  /// [legal] Consent button: gives the consent.
  ///
  /// In en, this message translates to:
  /// **'Agree and book'**
  String get agreeAndBook;

  /// [legal] Dialog title.
  ///
  /// In en, this message translates to:
  /// **'Revoke your consent?'**
  String get revokeTitle;

  /// [legal] Dialog text: the effect of revoking, including the end of history sharing with the clinic (SAL-42).
  ///
  /// In en, this message translates to:
  /// **'{clinic} will no longer see your profile. It keeps your name and the details of appointments that have already taken place, as part of its care records. Your future appointments with {clinic} will be cancelled. Any sharing of your history from other clinics with {clinic} ends too; consenting again does not restore it.'**
  String revokeBody(String clinic);

  /// Dialog button.
  ///
  /// In en, this message translates to:
  /// **'Keep consent'**
  String get keepConsent;

  /// [legal] Button and dialog button.
  ///
  /// In en, this message translates to:
  /// **'Revoke consent'**
  String get revokeConsent;

  /// Empty list.
  ///
  /// In en, this message translates to:
  /// **'You have not given any consents yet.'**
  String get noConsents;

  /// [legal] Above the consents.
  ///
  /// In en, this message translates to:
  /// **'You share your profile and appointments with these clinics for your care.'**
  String get consentsIntro;

  /// A consent's date.
  ///
  /// In en, this message translates to:
  /// **'Given on {date}'**
  String givenOn(String date);

  /// A consent's status.
  ///
  /// In en, this message translates to:
  /// **'Active'**
  String get consentActive;

  /// A consent's status and date.
  ///
  /// In en, this message translates to:
  /// **'Revoked on {date}'**
  String revokedOn(String date);

  /// [legal] After revoking: its effect.
  ///
  /// In en, this message translates to:
  /// **'Your consent is revoked. Your future appointments with this clinic will be cancelled, and you will get an e-mail for each.'**
  String get consentRevoked;

  /// Refused: already revoked.
  ///
  /// In en, this message translates to:
  /// **'This consent is already revoked.'**
  String get consentAlreadyRevoked;

  /// M1.
  ///
  /// In en, this message translates to:
  /// **'Join consultation'**
  String get joinConsultation;

  /// M2.
  ///
  /// In en, this message translates to:
  /// **'You can join from 10 minutes before the start.'**
  String get joinFrom;

  /// M3.
  ///
  /// In en, this message translates to:
  /// **'Check your camera and microphone'**
  String get checkTitle;

  /// M4.
  ///
  /// In en, this message translates to:
  /// **'The practitioner will see and hear you once you join.'**
  String get checkIntro;

  /// M5.
  ///
  /// In en, this message translates to:
  /// **'No sound detected. Check that your microphone is on and not muted.'**
  String get noSound;

  /// M6.
  ///
  /// In en, this message translates to:
  /// **'Join with camera off'**
  String get joinCameraOff;

  /// M6.
  ///
  /// In en, this message translates to:
  /// **'Join with microphone off'**
  String get joinMicrophoneOff;

  /// M7.
  ///
  /// In en, this message translates to:
  /// **'Join'**
  String get joinNow;

  /// M8.
  ///
  /// In en, this message translates to:
  /// **'Joining…'**
  String get joining;

  /// [legal] M9: a permission refused.
  ///
  /// In en, this message translates to:
  /// **'Salus does not have access to your camera. You can allow it in your device settings, or join without video.'**
  String get cameraDenied;

  /// [legal] M10: a permission refused.
  ///
  /// In en, this message translates to:
  /// **'Salus does not have access to your microphone. You can allow it in your device settings. Without it, the practitioner cannot hear you.'**
  String get microphoneDenied;

  /// M11.
  ///
  /// In en, this message translates to:
  /// **'Open settings'**
  String get openSettings;

  /// M12.
  ///
  /// In en, this message translates to:
  /// **'No camera found.'**
  String get noCamera;

  /// M12.
  ///
  /// In en, this message translates to:
  /// **'No microphone found.'**
  String get noMicrophone;

  /// M13.
  ///
  /// In en, this message translates to:
  /// **'Waiting for the practitioner'**
  String get waiting;

  /// M14; the practitioner's name.
  ///
  /// In en, this message translates to:
  /// **'{name} will join you here. Keep this screen open.'**
  String joinsYouHere(String name);

  /// M15.
  ///
  /// In en, this message translates to:
  /// **'Mute microphone'**
  String get mute;

  /// M15.
  ///
  /// In en, this message translates to:
  /// **'Unmute microphone'**
  String get unmute;

  /// M15.
  ///
  /// In en, this message translates to:
  /// **'Turn camera off'**
  String get cameraOff;

  /// M15.
  ///
  /// In en, this message translates to:
  /// **'Turn camera on'**
  String get cameraOn;

  /// M15; also leaving a questionnaire (questionnaire M15).
  ///
  /// In en, this message translates to:
  /// **'Leave'**
  String get leave;

  /// M16; the practitioner's name.
  ///
  /// In en, this message translates to:
  /// **'{name}\'s camera is off'**
  String peerCameraOff(String name);

  /// M17; the practitioner's name.
  ///
  /// In en, this message translates to:
  /// **'{name} is muted'**
  String peerMuted(String name);

  /// M18; the practitioner's name.
  ///
  /// In en, this message translates to:
  /// **'{name} has left the consultation. They can join again until the consultation ends.'**
  String peerLeft(String name);

  /// M19.
  ///
  /// In en, this message translates to:
  /// **'Connection lost. Reconnecting…'**
  String get reconnecting;

  /// M20.
  ///
  /// In en, this message translates to:
  /// **'The connection could not be restored.'**
  String get notRestored;

  /// M21.
  ///
  /// In en, this message translates to:
  /// **'Reconnected'**
  String get reconnected;

  /// M22.
  ///
  /// In en, this message translates to:
  /// **'The consultation has ended'**
  String get consultationOver;

  /// M23.
  ///
  /// In en, this message translates to:
  /// **'You can close this screen.'**
  String get closeScreen;

  /// M24.
  ///
  /// In en, this message translates to:
  /// **'The consultation reached its time limit and has ended.'**
  String get timeLimit;

  /// M25.
  ///
  /// In en, this message translates to:
  /// **'The practitioner did not join'**
  String get practitionerAbsent;

  /// M26.
  ///
  /// In en, this message translates to:
  /// **'The time to join this consultation has passed. The clinic can see that you joined. The clinic may contact you, or you can book a new appointment.'**
  String get timePassed;

  /// [legal] M27.
  ///
  /// In en, this message translates to:
  /// **'You withdrew your consent for this clinic, so you have left the consultation.'**
  String get withdrewConsent;

  /// M28.
  ///
  /// In en, this message translates to:
  /// **'This appointment was cancelled, so the consultation has ended.'**
  String get appointmentWasCancelled;

  /// M29.
  ///
  /// In en, this message translates to:
  /// **'This consultation opens 10 minutes before the start.'**
  String get windowNotOpen;

  /// M30.
  ///
  /// In en, this message translates to:
  /// **'This consultation has ended.'**
  String get consultationEnded;

  /// M32.
  ///
  /// In en, this message translates to:
  /// **'Joining needs your consent for this clinic.'**
  String get joinNeedsConsent;

  /// M33.
  ///
  /// In en, this message translates to:
  /// **'This consultation is not available.'**
  String get consultationNotAvailable;

  /// M34.
  ///
  /// In en, this message translates to:
  /// **'The video connection could not be started. Check your connection and try again.'**
  String get cannotConnect;

  /// [legal] M35: Android, before the system permission prompt. Changed by SAL-12 (no longer "not recorded" alone): draft, waiting for the owner and the lawyer (Q16).
  ///
  /// In en, this message translates to:
  /// **'To join the video consultation, Salus needs your camera and microphone. The consultation is not recorded as audio or video. If you agree, what is said is transcribed and translated.'**
  String get whyCameraAndMicrophone;

  /// M36: while the transcription agent is in the call. Proposed, waiting for the owner (consultation COPY.md).
  ///
  /// In en, this message translates to:
  /// **'Transcription is on'**
  String get transcriptionOn;

  /// SAL-12: the heading of the notice before the call. Draft, waiting for the owner and the lawyer (Q16).
  ///
  /// In en, this message translates to:
  /// **'Transcription and translation'**
  String get transcriptionTitle;

  /// [legal] SAL-12 notice: what is transcribed and translated. Draft, waiting for the owner and the lawyer (Q16).
  ///
  /// In en, this message translates to:
  /// **'In your video consultations with {clinic}, what you and the practitioner say can be transcribed as you speak and translated, so the practitioner can follow you in your language.'**
  String transcriptionIntro(String clinic);

  /// [legal] SAL-12 notice: the AI assistance for the practitioner. Draft, waiting for the owner and the lawyer (Q16).
  ///
  /// In en, this message translates to:
  /// **'From the transcript, the practitioner also gets AI assistance: suggestions during the consultation and drafts of documents. These are suggestions only; the practitioner checks them and makes every decision.'**
  String get transcriptionAssistance;

  /// [legal] SAL-12 notice: where the transcript is kept; no audio or video recording. Draft, waiting for the owner and the lawyer (Q16).
  ///
  /// In en, this message translates to:
  /// **'The transcript is kept with your medical record at {clinic}. The consultation is not recorded as audio or video.'**
  String transcriptionKept(String clinic);

  /// [legal] SAL-12 notice: what happens on decline. Draft, waiting for the owner and the lawyer (Q16).
  ///
  /// In en, this message translates to:
  /// **'If you do not agree, the consultation goes ahead as usual, without transcription, translation or AI assistance.'**
  String get transcriptionDecline;

  /// [legal] SAL-12 notice: how to withdraw; quotes the transcriptionOn badge and the myConsents screen. Draft, waiting for the owner and the lawyer (Q16).
  ///
  /// In en, this message translates to:
  /// **'You can withdraw your agreement at any time, also during the consultation: tap “Transcription is on”, or open My consents. Transcription stops at once; what was transcribed until then stays in your record.'**
  String get transcriptionWithdraw;

  /// [legal] SAL-12 notice: the button that gives the transcription consent. Draft, waiting for the owner and the lawyer (Q16).
  ///
  /// In en, this message translates to:
  /// **'Agree and continue'**
  String get transcriptionAgree;

  /// [legal] SAL-12 notice: the button that declines; the call goes on without the agent. Draft, waiting for the owner and the lawyer (Q16).
  ///
  /// In en, this message translates to:
  /// **'Continue without transcription'**
  String get transcriptionWithout;

  /// SAL-12: the dialog that withdraws the transcription consent (in the call and under My consents). Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Stop transcription?'**
  String get stopTranscriptionTitle;

  /// [legal] SAL-12: the effect of withdrawing the transcription consent. Draft, waiting for the owner and the lawyer (Q16).
  ///
  /// In en, this message translates to:
  /// **'Transcription, translation and AI assistance stop at once in your consultations with {clinic}. A consultation in progress goes on without them, and your appointments stay as they are. What was transcribed until now stays in your record.'**
  String stopTranscriptionBody(String clinic);

  /// SAL-12: the dialog button that keeps the transcription consent. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Keep transcription'**
  String get keepTranscription;

  /// [legal] SAL-12: the dialog button that withdraws the transcription consent, and the badge tooltip. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Stop transcription'**
  String get stopTranscription;

  /// [legal] SAL-12: after the transcription consent was withdrawn. Draft, waiting for the owner and the lawyer (Q16).
  ///
  /// In en, this message translates to:
  /// **'Transcription is off. Your consultations with this clinic go ahead without it.'**
  String get transcriptionStopped;

  /// SAL-12: My consents, the label of a care consent. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Sharing for care'**
  String get consentForCare;

  /// SAL-12: My consents, the label of a transcription consent. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Transcription and AI assistance'**
  String get consentForTranscription;

  /// M1.
  ///
  /// In en, this message translates to:
  /// **'To fill in'**
  String get toFillIn;

  /// M1.
  ///
  /// In en, this message translates to:
  /// **'In progress'**
  String get inProgress;

  /// M1: status.
  ///
  /// In en, this message translates to:
  /// **'Submitted'**
  String get submittedStatus;

  /// M1: status.
  ///
  /// In en, this message translates to:
  /// **'Closed'**
  String get closedStatus;

  /// M2.
  ///
  /// In en, this message translates to:
  /// **'Fill in'**
  String get fillIn;

  /// M2: continue filling in.
  ///
  /// In en, this message translates to:
  /// **'Continue'**
  String get continueFilling;

  /// M2.
  ///
  /// In en, this message translates to:
  /// **'View answers'**
  String get viewAnswers;

  /// M3.
  ///
  /// In en, this message translates to:
  /// **'Please submit it before your appointment starts.'**
  String get submitBeforeStart;

  /// M4.
  ///
  /// In en, this message translates to:
  /// **'The appointment has started, so this questionnaire can no longer be filled in.'**
  String get questionnaireClosed;

  /// M5; the clinic's name.
  ///
  /// In en, this message translates to:
  /// **'Your answers are saved as you go. {clinic} sees them only after you submit.'**
  String savedAsYouGo(String clinic);

  /// M6.
  ///
  /// In en, this message translates to:
  /// **'Saved'**
  String get saved;

  /// M6.
  ///
  /// In en, this message translates to:
  /// **'Saving…'**
  String get saving;

  /// M6.
  ///
  /// In en, this message translates to:
  /// **'Not saved. Check your connection. Saving will try again.'**
  String get notSaved;

  /// M7.
  ///
  /// In en, this message translates to:
  /// **'Required'**
  String get requiredQuestion;

  /// M8.
  ///
  /// In en, this message translates to:
  /// **'More information'**
  String get moreInformation;

  /// M9.
  ///
  /// In en, this message translates to:
  /// **'Your answer'**
  String get yourAnswer;

  /// M9.
  ///
  /// In en, this message translates to:
  /// **'Choose a date'**
  String get chooseDate;

  /// M9.
  ///
  /// In en, this message translates to:
  /// **'Select one.'**
  String get selectOne;

  /// M9.
  ///
  /// In en, this message translates to:
  /// **'Select all that apply.'**
  String get selectAll;

  /// M9.
  ///
  /// In en, this message translates to:
  /// **'Yes'**
  String get yes;

  /// M9.
  ///
  /// In en, this message translates to:
  /// **'No'**
  String get no;

  /// M10.
  ///
  /// In en, this message translates to:
  /// **'This question is required.'**
  String get questionRequired;

  /// M10.
  ///
  /// In en, this message translates to:
  /// **'Enter a whole number.'**
  String get enterWholeNumber;

  /// M10.
  ///
  /// In en, this message translates to:
  /// **'Enter a number.'**
  String get enterNumber;

  /// M10.
  ///
  /// In en, this message translates to:
  /// **'This number is too large.'**
  String get numberTooLarge;

  /// M10; max is 255 or 4000.
  ///
  /// In en, this message translates to:
  /// **'Enter up to {max} characters.'**
  String upToCharacters(int max);

  /// M10.
  ///
  /// In en, this message translates to:
  /// **'Enter a valid date.'**
  String get enterValidDate;

  /// M11.
  ///
  /// In en, this message translates to:
  /// **'Submit'**
  String get submit;

  /// M11.
  ///
  /// In en, this message translates to:
  /// **'Submit your answers?'**
  String get submitTitle;

  /// M11; the clinic's name.
  ///
  /// In en, this message translates to:
  /// **'You cannot change them after you submit. Your practitioner at {clinic} will see them.'**
  String submitBody(String clinic);

  /// M11: dialog button.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get cancel;

  /// M12.
  ///
  /// In en, this message translates to:
  /// **'Some required questions are not answered.'**
  String get requiredMissing;

  /// M13.
  ///
  /// In en, this message translates to:
  /// **'Answers submitted'**
  String get answersSubmitted;

  /// M13.
  ///
  /// In en, this message translates to:
  /// **'Your practitioner can see them before your appointment. You can view them from My appointments.'**
  String get answersSubmittedBody;

  /// M14.
  ///
  /// In en, this message translates to:
  /// **'Your answers'**
  String get yourAnswers;

  /// M14; when is dateAndTime.
  ///
  /// In en, this message translates to:
  /// **'Submitted {when}'**
  String submittedAt(String when);

  /// A long date and a 24-hour time.
  ///
  /// In en, this message translates to:
  /// **'{date}, {time}'**
  String dateAndTime(String date, String time);

  /// M14.
  ///
  /// In en, this message translates to:
  /// **'Not answered'**
  String get notAnswered;

  /// M15.
  ///
  /// In en, this message translates to:
  /// **'Your last answer is not saved yet. Leave anyway?'**
  String get leaveUnsaved;

  /// M15.
  ///
  /// In en, this message translates to:
  /// **'Stay'**
  String get stay;

  /// M16.
  ///
  /// In en, this message translates to:
  /// **'Filling in this questionnaire needs your consent for this clinic.'**
  String get fillingNeedsConsent;

  /// M16.
  ///
  /// In en, this message translates to:
  /// **'The appointment has started, so this questionnaire can no longer be changed.'**
  String get pastDue;

  /// M16.
  ///
  /// In en, this message translates to:
  /// **'The clinic withdrew this questionnaire. You do not need to fill it in.'**
  String get withdrawnByClinic;

  /// M16.
  ///
  /// In en, this message translates to:
  /// **'You have already submitted this questionnaire.'**
  String get alreadySubmitted;

  /// M16.
  ///
  /// In en, this message translates to:
  /// **'These answers were changed on another device. The latest answers are shown.'**
  String get changedElsewhere;

  /// M16.
  ///
  /// In en, this message translates to:
  /// **'This questionnaire is not available.'**
  String get questionnaireNotAvailable;

  /// M17: the date picker's confirm button.
  ///
  /// In en, this message translates to:
  /// **'Done'**
  String get done;

  /// M1.
  ///
  /// In en, this message translates to:
  /// **'Add file'**
  String get addFile;

  /// M2.
  ///
  /// In en, this message translates to:
  /// **'Photos up to 20 MB.'**
  String get photosUpTo;

  /// M2.
  ///
  /// In en, this message translates to:
  /// **'Documents (PDF) up to 20 MB.'**
  String get documentsUpTo;

  /// M2.
  ///
  /// In en, this message translates to:
  /// **'Videos up to 200 MB.'**
  String get videosUpTo;

  /// M2: an item that takes one file.
  ///
  /// In en, this message translates to:
  /// **'1 file.'**
  String get oneFileHint;

  /// M2: an item that takes 2 or more files.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, one{Up to {n} file.} other{Up to {n} files.}}'**
  String upToFilesHint(int n);

  /// M3.
  ///
  /// In en, this message translates to:
  /// **'Take a photo'**
  String get takePhoto;

  /// M3.
  ///
  /// In en, this message translates to:
  /// **'Record a video'**
  String get recordVideo;

  /// M3.
  ///
  /// In en, this message translates to:
  /// **'Choose from photos'**
  String get chooseFromPhotos;

  /// M3.
  ///
  /// In en, this message translates to:
  /// **'Choose a document'**
  String get chooseDocument;

  /// M4.
  ///
  /// In en, this message translates to:
  /// **'Preparing…'**
  String get preparing;

  /// M4; n from 0 to 100.
  ///
  /// In en, this message translates to:
  /// **'Uploading… {n}%'**
  String uploadingPercent(int n);

  /// M4.
  ///
  /// In en, this message translates to:
  /// **'Checking the file…'**
  String get checkingFile;

  /// M4.
  ///
  /// In en, this message translates to:
  /// **'Added'**
  String get added;

  /// M5.
  ///
  /// In en, this message translates to:
  /// **'Not uploaded. Check your connection.'**
  String get notUploaded;

  /// M5, M6.
  ///
  /// In en, this message translates to:
  /// **'Remove'**
  String get remove;

  /// M6.
  ///
  /// In en, this message translates to:
  /// **'Remove file'**
  String get removeFile;

  /// M6.
  ///
  /// In en, this message translates to:
  /// **'Remove this file?'**
  String get removeFileTitle;

  /// M6.
  ///
  /// In en, this message translates to:
  /// **'It is deleted from your answers.'**
  String get removeFileBody;

  /// M7.
  ///
  /// In en, this message translates to:
  /// **'This file did not pass the safety check and was deleted. Choose another file.'**
  String get fileRejected;

  /// M8; max is 20 or 200.
  ///
  /// In en, this message translates to:
  /// **'This file is larger than {max} MB.'**
  String fileTooLarge(int max);

  /// M8.
  ///
  /// In en, this message translates to:
  /// **'This kind of file cannot be added to this question.'**
  String get fileTypeRefused;

  /// M8.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, one{You can add up to {n} file to this question.} other{You can add up to {n} files to this question.}}'**
  String tooManyFiles(int n);

  /// M8.
  ///
  /// In en, this message translates to:
  /// **'The files in this questionnaire have reached 500 MB. Remove a file to add another.'**
  String get filesFull;

  /// M8.
  ///
  /// In en, this message translates to:
  /// **'The file could not be uploaded. Try again.'**
  String get fileNotUploaded;

  /// M9.
  ///
  /// In en, this message translates to:
  /// **'This file could not be prepared for upload. Choose another file.'**
  String get cannotPrepareFile;

  /// M10.
  ///
  /// In en, this message translates to:
  /// **'Some files are still being checked. You can submit once they are ready.'**
  String get filesBeingChecked;

  /// M11.
  ///
  /// In en, this message translates to:
  /// **'A file is still uploading. Leave anyway? The upload stops.'**
  String get leaveUploading;

  /// M12.
  ///
  /// In en, this message translates to:
  /// **'{n, plural, one{1 file} other{{n} files}}'**
  String fileCount(int n);

  /// M13.
  ///
  /// In en, this message translates to:
  /// **'Photo'**
  String get photoKind;

  /// M13.
  ///
  /// In en, this message translates to:
  /// **'Document'**
  String get documentKind;

  /// M13.
  ///
  /// In en, this message translates to:
  /// **'Video'**
  String get videoKind;

  /// M13.
  ///
  /// In en, this message translates to:
  /// **'{n} KB'**
  String kilobytes(int n);

  /// M13; mb has one decimal, written for the language.
  ///
  /// In en, this message translates to:
  /// **'{mb} MB'**
  String megabytes(String mb);

  /// M12, M13: "Photo, 2.4 MB".
  ///
  /// In en, this message translates to:
  /// **'{kind}, {size}'**
  String kindAndSize(String kind, String size);

  /// M14: screen reader; the file's name.
  ///
  /// In en, this message translates to:
  /// **'Uploading {title}, {n} percent'**
  String uploadingLabel(String title, int n);

  /// M14: screen reader.
  ///
  /// In en, this message translates to:
  /// **'{kind}, {title}, {status}'**
  String fileLabel(String kind, String title, String status);

  /// [legal] M18: Android, before the system permission prompt.
  ///
  /// In en, this message translates to:
  /// **'Allow camera access'**
  String get allowCamera;

  /// [legal] M18: Android, before the system permission prompt.
  ///
  /// In en, this message translates to:
  /// **'Salus needs the camera to take the photo or video you add to this questionnaire.'**
  String get whyCameraForFiles;

  /// M18; also the sign-in code step.
  ///
  /// In en, this message translates to:
  /// **'Continue'**
  String get continueLabel;

  /// [legal] M19: a permission refused.
  ///
  /// In en, this message translates to:
  /// **'Salus does not have access to your camera. You can allow it in your device settings, or choose a file instead.'**
  String get cameraDeniedForFiles;

  /// [legal] M20: a permission refused.
  ///
  /// In en, this message translates to:
  /// **'Salus does not have access to your photos. You can allow it in your device settings.'**
  String get photosDenied;

  /// M21: the name of a photo taken with the camera.
  ///
  /// In en, this message translates to:
  /// **'Photo, {date}, {time}'**
  String cameraPhotoTitle(String date, String time);

  /// M22: the name of a video recorded with the camera.
  ///
  /// In en, this message translates to:
  /// **'Video, {date}, {time}'**
  String cameraVideoTitle(String date, String time);

  /// Specialty, SNOMED CT 394814009.
  ///
  /// In en, this message translates to:
  /// **'General practice'**
  String get specialty394814009;

  /// Specialty, SNOMED CT 419192003.
  ///
  /// In en, this message translates to:
  /// **'Internal medicine'**
  String get specialty419192003;

  /// Specialty, SNOMED CT 394579002.
  ///
  /// In en, this message translates to:
  /// **'Cardiology'**
  String get specialty394579002;

  /// Specialty, SNOMED CT 394582007.
  ///
  /// In en, this message translates to:
  /// **'Dermatology'**
  String get specialty394582007;

  /// Specialty, SNOMED CT 394586005.
  ///
  /// In en, this message translates to:
  /// **'Gynaecology'**
  String get specialty394586005;

  /// Specialty, SNOMED CT 394537008.
  ///
  /// In en, this message translates to:
  /// **'Paediatrics'**
  String get specialty394537008;

  /// Specialty, SNOMED CT 394587001.
  ///
  /// In en, this message translates to:
  /// **'Psychiatry'**
  String get specialty394587001;

  /// Specialty, SNOMED CT 394591006.
  ///
  /// In en, this message translates to:
  /// **'Neurology'**
  String get specialty394591006;

  /// Specialty, SNOMED CT 418960008 (otolaryngology).
  ///
  /// In en, this message translates to:
  /// **'ENT'**
  String get specialty418960008;

  /// Specialty, SNOMED CT 394594003.
  ///
  /// In en, this message translates to:
  /// **'Ophthalmology'**
  String get specialty394594003;

  /// Specialty, SNOMED CT 394801008.
  ///
  /// In en, this message translates to:
  /// **'Orthopaedics'**
  String get specialty394801008;

  /// Specialty, SNOMED CT 394583002.
  ///
  /// In en, this message translates to:
  /// **'Endocrinology'**
  String get specialty394583002;

  /// docs/i18n COPY.md M1: the questionnaire is not written or translated in the app's language; the language's own name (L1).
  ///
  /// In en, this message translates to:
  /// **'This questionnaire is not available in {language}.'**
  String questionnaireNotInLanguage(String language);

  /// docs/i18n COPY.md M2: the questionnaire's languages, each by its own name (L1), separated by commas.
  ///
  /// In en, this message translates to:
  /// **'It is available in: {languages}. Choose the one you understand best.'**
  String questionnaireLanguages(String languages);

  /// docs/i18n COPY.md M3: the button that starts the questionnaire in the chosen language; its own name (L1).
  ///
  /// In en, this message translates to:
  /// **'Continue in {language}'**
  String continueInLanguage(String language);

  /// docs/i18n COPY.md M4: the API refused the language (language_unavailable).
  ///
  /// In en, this message translates to:
  /// **'This language is no longer available for this questionnaire. Choose another one.'**
  String get languageUnavailable;

  /// A practitioner's specialty and clinic on the free times screen.
  ///
  /// In en, this message translates to:
  /// **'{specialty}, {clinic}'**
  String specialtyAtClinic(String specialty, String clinic);

  /// SAL-17: the button under an appointment that has started, and the title of its documents screen.
  ///
  /// In en, this message translates to:
  /// **'Documents'**
  String get documents;

  /// SAL-17: the documents screen when the appointment has no signed document.
  ///
  /// In en, this message translates to:
  /// **'There are no signed documents for this appointment yet. A document appears here once your practitioner signs it.'**
  String get noDocuments;

  /// SAL-17: under a signed document's title: the day it was signed, as a long date in the clinic's zone.
  ///
  /// In en, this message translates to:
  /// **'Signed on {date}'**
  String documentSignedOn(String date);

  /// SAL-17: under a signed document written in another language than the app's: that language by its own name.
  ///
  /// In en, this message translates to:
  /// **'Language: {language}'**
  String documentLanguage(String language);

  /// SAL-17: button: the signed document's PDF to the device's share sheet (open, save, send).
  ///
  /// In en, this message translates to:
  /// **'Save or share the PDF'**
  String get sharePdf;

  /// SAL-17: the button to the transcript copy, and its screen's title.
  ///
  /// In en, this message translates to:
  /// **'Transcript'**
  String get transcript;

  /// SAL-17: at the top of the transcript copy.
  ///
  /// In en, this message translates to:
  /// **'The consultation as it was transcribed and translated automatically. It may contain mistakes.'**
  String get transcriptNote;

  /// SAL-17: the transcript copy when nothing was stored.
  ///
  /// In en, this message translates to:
  /// **'Nothing was transcribed in this consultation.'**
  String get transcriptEmpty;

  /// SAL-17: the transcript: the patient's own lines.
  ///
  /// In en, this message translates to:
  /// **'You'**
  String get speakerYou;

  /// SAL-17: the transcript: the practitioner's lines.
  ///
  /// In en, this message translates to:
  /// **'Practitioner'**
  String get speakerPractitioner;

  /// SAL-17: button: the transcript copy to the device's share sheet as text.
  ///
  /// In en, this message translates to:
  /// **'Share the transcript'**
  String get shareTranscript;

  /// SAL-17: under a transcript line: its translation.
  ///
  /// In en, this message translates to:
  /// **'Translation: {text}'**
  String transcriptTranslation(String text);

  /// [legal] SAL-13: under the plain-language explanation of a medical term: it informs, it diagnoses nothing.
  ///
  /// In en, this message translates to:
  /// **'Informational, not a diagnosis'**
  String get termExplanationNote;

  /// SAL-13: beside an explanation of a medical term a doctor has not reviewed yet.
  ///
  /// In en, this message translates to:
  /// **'Not reviewed yet'**
  String get termNotReviewed;

  /// SAL-13: button: closes a medical term's explanation.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get termClose;

  /// Home button and the conversation list title (SAL-31).
  ///
  /// In en, this message translates to:
  /// **'Messages'**
  String get chatsTitle;

  /// The conversation list without a conversation.
  ///
  /// In en, this message translates to:
  /// **'You have no conversations yet.'**
  String get chatsEmpty;

  /// The unread badge of a conversation, read out.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, one{{count} unread message} other{{count} unread messages}}'**
  String chatUnread(int count);

  /// A conversation's title when the doctor's name is not known.
  ///
  /// In en, this message translates to:
  /// **'Your doctor'**
  String get chatDoctorUnknown;

  /// The composer field.
  ///
  /// In en, this message translates to:
  /// **'Write a message'**
  String get chatMessageHint;

  /// The composer: send button tooltip.
  ///
  /// In en, this message translates to:
  /// **'Send'**
  String get chatSend;

  /// The composer: add a file from the camera or the files, tooltip.
  ///
  /// In en, this message translates to:
  /// **'Add a file'**
  String get chatAttach;

  /// A message that did not reach the API; the text stays in the composer.
  ///
  /// In en, this message translates to:
  /// **'The message was not sent. Check your connection and try again.'**
  String get chatSendFailed;

  /// An attachment the clinic's scan refused.
  ///
  /// In en, this message translates to:
  /// **'This file was not accepted.'**
  String get chatFileRejected;

  /// The composer: remove a file not yet sent, tooltip.
  ///
  /// In en, this message translates to:
  /// **'Remove the file'**
  String get chatFileRemove;

  /// An upload refused because the patient already has the most files waiting to be sent (SAL-30 too_many_files); count is the limit.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, one{You already have {count} file waiting to be sent. Files not sent are removed after 24 hours; try again later.} other{You already have {count} files waiting to be sent. Files not sent are removed after 24 hours; try again later.}}'**
  String chatTooManyFiles(int count);

  /// An attachment that could not be read.
  ///
  /// In en, this message translates to:
  /// **'The file could not be opened. Try again.'**
  String get chatOpenFailed;

  /// The live connection ended because the session ended (4401).
  ///
  /// In en, this message translates to:
  /// **'Your session has ended. Sign in again to see new messages.'**
  String get chatEndedSession;

  /// The live connection ended because access ended (4403, 4404).
  ///
  /// In en, this message translates to:
  /// **'New messages can no longer be shown in this conversation.'**
  String get chatEnded;

  /// The clinic's chat hours: button tooltip and dialog title.
  ///
  /// In en, this message translates to:
  /// **'Chat hours'**
  String get chatHours;

  /// One weekday of the chat hours; day is the weekday name, start and end are HH:MM.
  ///
  /// In en, this message translates to:
  /// **'{day}: {start}–{end}'**
  String chatHoursDay(String day, String start, String end);

  /// The clinic has no chat hours.
  ///
  /// In en, this message translates to:
  /// **'The clinic has not set chat hours.'**
  String get chatHoursNone;

  /// The chat hours dialog: the clinic's IANA time zone the hours are in (shown as given, not converted).
  ///
  /// In en, this message translates to:
  /// **'Times are in the clinic\'s time zone, {zone}.'**
  String chatHoursZone(String zone);

  /// The clinic's expected response time, in the chat hours dialog.
  ///
  /// In en, this message translates to:
  /// **'{hours, plural, one{The clinic usually answers within {hours} hour.} other{The clinic usually answers within {hours} hours.}}'**
  String chatResponseTime(int hours);

  /// The notice above the composer outside the chat hours (SAL-29 open_now false).
  ///
  /// In en, this message translates to:
  /// **'{hours, plural, one{The clinic is outside its chat hours. You can still write; it usually answers within {hours} hour.} other{The clinic is outside its chat hours. You can still write; it usually answers within {hours} hours.}}'**
  String chatOutOfHours(int hours);

  /// System message participant_invited; reason is second_opinion, specialist_review or handover.
  ///
  /// In en, this message translates to:
  /// **'{reason, select, second_opinion{A doctor from another clinic was invited to this conversation for a second opinion.} specialist_review{A doctor from another clinic was invited to this conversation for a specialist review.} handover{A doctor from another clinic was invited to this conversation to take over the care.} other{A doctor from another clinic was invited to this conversation.}}'**
  String chatSystemInvited(String reason);

  /// System message participant_added.
  ///
  /// In en, this message translates to:
  /// **'{reason, select, second_opinion{A doctor joined the conversation for a second opinion.} specialist_review{A doctor joined the conversation for a specialist review.} handover{A doctor joined the conversation to take over the care.} other{A doctor joined the conversation.}}'**
  String chatSystemAdded(String reason);

  /// System message participant_removed.
  ///
  /// In en, this message translates to:
  /// **'A doctor no longer takes part in the conversation.'**
  String get chatSystemRemoved;

  /// System message consent_granted: the patient agreed.
  ///
  /// In en, this message translates to:
  /// **'You allowed a doctor from another clinic to join the conversation.'**
  String get chatSystemGranted;

  /// System message consent_declined: the patient declined.
  ///
  /// In en, this message translates to:
  /// **'You declined a doctor from another clinic.'**
  String get chatSystemDeclined;

  /// A system message the app does not know.
  ///
  /// In en, this message translates to:
  /// **'The conversation was updated.'**
  String get chatSystemOther;

  /// [legal] The consent prompt: its heading.
  ///
  /// In en, this message translates to:
  /// **'A doctor from another clinic asks to join'**
  String get chatInviteTitle;

  /// [legal] The consent prompt: who asks and why.
  ///
  /// In en, this message translates to:
  /// **'{reason, select, second_opinion{Your clinic asks a doctor from another clinic to join this conversation for a second opinion.} specialist_review{Your clinic asks a doctor from another clinic to join this conversation for a specialist review.} handover{Your clinic asks a doctor from another clinic to join this conversation to take over your care.} other{Your clinic asks a doctor from another clinic to join this conversation.}}'**
  String chatInviteBody(String reason);

  /// [legal] The consent prompt: the invited doctor's name and clinic, when known.
  ///
  /// In en, this message translates to:
  /// **'Doctor: {doctor}, {clinic}'**
  String chatInviteWho(String doctor, String clinic);

  /// [legal] The consent prompt while the invited doctor's name and clinic are not known: the grant waits for them.
  ///
  /// In en, this message translates to:
  /// **'The invited doctor\'s name and clinic could not be loaded. Access can be granted only once they are shown.'**
  String get chatInviteUnresolved;

  /// [legal] The consent prompt: what the doctor would see; history is full, from or unknown.
  ///
  /// In en, this message translates to:
  /// **'{history, select, full{If you agree, the doctor sees the whole conversation, including the earlier messages, and can write in it.} from{If you agree, the doctor sees the conversation from a message your clinic chose onwards, and can write in it.} other{If you agree, the doctor sees the conversation and can write in it.}}'**
  String chatInviteSees(String history);

  /// [legal] The consent prompt: declining and revoking.
  ///
  /// In en, this message translates to:
  /// **'You may decline. If you agree, you can revoke the doctor\'s access later in this conversation; it ends at once.'**
  String get chatInviteChoice;

  /// [legal] The consent prompt: grant button.
  ///
  /// In en, this message translates to:
  /// **'Grant access'**
  String get chatGrant;

  /// [legal] The consent prompt: decline button.
  ///
  /// In en, this message translates to:
  /// **'Decline'**
  String get chatDecline;

  /// [legal] After a grant.
  ///
  /// In en, this message translates to:
  /// **'Access granted. You can revoke it in this conversation at any time.'**
  String get chatGranted;

  /// [legal] After a decline.
  ///
  /// In en, this message translates to:
  /// **'Declined. The doctor does not join this conversation.'**
  String get chatDeclined;

  /// [legal] The chat view's menu: revoke a granted invitation.
  ///
  /// In en, this message translates to:
  /// **'Revoke the access of the doctor from another clinic'**
  String get chatRevoke;

  /// [legal] The chat view's line for a doctor of another clinic the patient granted access, beside the revoke button.
  ///
  /// In en, this message translates to:
  /// **'Access granted to {doctor}, {clinic}.'**
  String chatGrantedTo(String doctor, String clinic);

  /// [legal] The granted doctor's line while their name and clinic are not known.
  ///
  /// In en, this message translates to:
  /// **'Access granted to a doctor from another clinic.'**
  String get chatGrantedToUnknown;

  /// [legal] The revocation dialog: its effect.
  ///
  /// In en, this message translates to:
  /// **'The doctor from the other clinic loses access to this conversation at once. The messages already written stay in it.'**
  String get chatRevokeBody;

  /// [legal] The revocation dialog: confirm button.
  ///
  /// In en, this message translates to:
  /// **'Revoke access'**
  String get chatRevokeConfirm;

  /// [legal] After a revocation.
  ///
  /// In en, this message translates to:
  /// **'Access revoked.'**
  String get chatRevoked;

  /// SAL-41: Profile button and screen title.
  ///
  /// In en, this message translates to:
  /// **'Allergies'**
  String get allergies;

  /// SAL-41: Profile button and screen title.
  ///
  /// In en, this message translates to:
  /// **'Current medications'**
  String get currentMedications;

  /// [legal] SAL-41: Allergies screen: who sees the profile's allergies.
  ///
  /// In en, this message translates to:
  /// **'Clinics you have given your consent to see the allergies in your profile when you have an appointment with them.'**
  String get allergiesIntro;

  /// [legal] SAL-41: Medications screen: who sees the profile's medications.
  ///
  /// In en, this message translates to:
  /// **'Clinics you have given your consent to see the medications in your profile when you have an appointment with them.'**
  String get medicationsIntro;

  /// SAL-41: Allergies screen, empty.
  ///
  /// In en, this message translates to:
  /// **'You have not added any allergies.'**
  String get noAllergies;

  /// SAL-41: Medications screen, empty.
  ///
  /// In en, this message translates to:
  /// **'You have not added any medications.'**
  String get noMedications;

  /// SAL-41: Button and form title.
  ///
  /// In en, this message translates to:
  /// **'Add an allergy'**
  String get addAllergy;

  /// SAL-41: Button and form title.
  ///
  /// In en, this message translates to:
  /// **'Add a medication'**
  String get addMedication;

  /// SAL-41: Form title when changing an allergy.
  ///
  /// In en, this message translates to:
  /// **'Change allergy'**
  String get changeAllergy;

  /// SAL-41: Form title when changing a medication.
  ///
  /// In en, this message translates to:
  /// **'Change medication'**
  String get changeMedication;

  /// SAL-41: Button on the patient's own allergy or medication: opens the form.
  ///
  /// In en, this message translates to:
  /// **'Change'**
  String get change;

  /// SAL-41: Allergy form: what the patient is allergic to.
  ///
  /// In en, this message translates to:
  /// **'Substance or food'**
  String get allergySubstance;

  /// SAL-41: Medication form: the medication's name.
  ///
  /// In en, this message translates to:
  /// **'Medication'**
  String get medicationName;

  /// SAL-41: Under the substance or medication field, which suggests terms.
  ///
  /// In en, this message translates to:
  /// **'Type to search, or write it in your own words.'**
  String get termSearchHint;

  /// SAL-41: Allergy form: the allergy's category.
  ///
  /// In en, this message translates to:
  /// **'Type (optional)'**
  String get allergyCategory;

  /// SAL-41: A drop-down's empty choice.
  ///
  /// In en, this message translates to:
  /// **'Not specified'**
  String get notSpecified;

  /// SAL-41: Allergy category.
  ///
  /// In en, this message translates to:
  /// **'Food'**
  String get categoryFood;

  /// SAL-41: Allergy category.
  ///
  /// In en, this message translates to:
  /// **'Medication'**
  String get categoryMedication;

  /// SAL-41: Allergy category (for example pollen or dust).
  ///
  /// In en, this message translates to:
  /// **'Environment'**
  String get categoryEnvironment;

  /// SAL-41: Allergy category (FHIR biologic: for example a vaccine or a blood product).
  ///
  /// In en, this message translates to:
  /// **'Biological product'**
  String get categoryBiologic;

  /// SAL-41: Allergy form: the reaction, up to 200 characters.
  ///
  /// In en, this message translates to:
  /// **'Reaction (optional)'**
  String get reactionOptional;

  /// SAL-41: Allergy form: the reaction's severity; only with a reaction.
  ///
  /// In en, this message translates to:
  /// **'Severity'**
  String get severity;

  /// SAL-41: Reaction severity.
  ///
  /// In en, this message translates to:
  /// **'Mild'**
  String get severityMild;

  /// SAL-41: Reaction severity.
  ///
  /// In en, this message translates to:
  /// **'Moderate'**
  String get severityModerate;

  /// SAL-41: Reaction severity.
  ///
  /// In en, this message translates to:
  /// **'Severe'**
  String get severitySevere;

  /// SAL-41: Allergy and medication form: the status drop-down.
  ///
  /// In en, this message translates to:
  /// **'Status'**
  String get entryStatus;

  /// SAL-41: Allergy clinical status.
  ///
  /// In en, this message translates to:
  /// **'Active'**
  String get allergyActive;

  /// SAL-41: Allergy clinical status.
  ///
  /// In en, this message translates to:
  /// **'Inactive'**
  String get allergyInactive;

  /// SAL-41: Allergy clinical status.
  ///
  /// In en, this message translates to:
  /// **'Resolved'**
  String get allergyResolved;

  /// SAL-41: Allergy card; the reaction as written.
  ///
  /// In en, this message translates to:
  /// **'Reaction: {reaction}'**
  String reactionLine(String reaction);

  /// SAL-41: Allergy card; the reaction as written and its severity (severityMild...).
  ///
  /// In en, this message translates to:
  /// **'Reaction: {reaction} ({severity})'**
  String reactionWithSeverity(String reaction, String severity);

  /// SAL-41: Allergy or medication card: a clinician confirmed it.
  ///
  /// In en, this message translates to:
  /// **'Confirmed by a clinician'**
  String get confirmedByClinician;

  /// SAL-41: Allergy or medication card: a clinic's entry, not the patient's.
  ///
  /// In en, this message translates to:
  /// **'Recorded by a clinic during one of your visits'**
  String get recordedByClinic;

  /// SAL-41: Form of a confirmed entry.
  ///
  /// In en, this message translates to:
  /// **'If you change this entry, it is no longer confirmed by a clinician.'**
  String get confirmedChangeNote;

  /// SAL-41: Substance or medication field: empty or too long.
  ///
  /// In en, this message translates to:
  /// **'Enter a name, up to 200 characters.'**
  String get nameRequired;

  /// SAL-41: Button on a clinic's entry, and the confirmation dialog's button.
  ///
  /// In en, this message translates to:
  /// **'Add to my profile'**
  String get addToProfile;

  /// SAL-41: Dialog title before a clinic's entry is added to the profile.
  ///
  /// In en, this message translates to:
  /// **'Add to your profile?'**
  String get addToProfileTitle;

  /// [legal] SAL-41: Dialog text: the effect of adding a clinic's entry to the profile.
  ///
  /// In en, this message translates to:
  /// **'Every clinic you have given your consent to will then see it, without the name of the clinic that recorded it.'**
  String get addToProfileBody;

  /// SAL-41: After a clinic's entry was added to the profile.
  ///
  /// In en, this message translates to:
  /// **'It is now in your profile.'**
  String get addedToProfile;

  /// SAL-41: Dialog title before an allergy or medication is removed.
  ///
  /// In en, this message translates to:
  /// **'Remove this entry?'**
  String get removeEntryTitle;

  /// [legal] SAL-41: Dialog text: the effect of a removal.
  ///
  /// In en, this message translates to:
  /// **'It will no longer be shown to you or to any clinic. If a clinic recorded or confirmed it, it stays in your medical record, marked as removed.'**
  String get removeEntryBody;

  /// SAL-41: After a removal.
  ///
  /// In en, this message translates to:
  /// **'The entry is removed.'**
  String get entryRemoved;

  /// SAL-41: After an allergy or medication was added or changed.
  ///
  /// In en, this message translates to:
  /// **'Saved to your profile.'**
  String get entrySaved;

  /// SAL-41: Write refused: the entry changed since it was read (412).
  ///
  /// In en, this message translates to:
  /// **'This entry was changed in the meantime. The latest version is shown.'**
  String get entryChanged;

  /// SAL-41: Write refused: invalid input (400).
  ///
  /// In en, this message translates to:
  /// **'Some details are not valid. Check them and try again.'**
  String get entryInvalid;

  /// SAL-41: Write refused (403).
  ///
  /// In en, this message translates to:
  /// **'You cannot change this entry.'**
  String get entryNotAllowed;

  /// SAL-41: Write refused: no such entry (404).
  ///
  /// In en, this message translates to:
  /// **'This entry is no longer available.'**
  String get entryNotFound;

  /// SAL-41: Medication form heading.
  ///
  /// In en, this message translates to:
  /// **'Dose (optional)'**
  String get doseHeading;

  /// SAL-41: Medication form: the dose's number.
  ///
  /// In en, this message translates to:
  /// **'Amount'**
  String get doseAmount;

  /// SAL-41: Medication form: the dose's unit.
  ///
  /// In en, this message translates to:
  /// **'Unit'**
  String get doseUnit;

  /// SAL-41: Dose unit choice.
  ///
  /// In en, this message translates to:
  /// **'milligrams (mg)'**
  String get unitMg;

  /// SAL-41: Dose unit choice.
  ///
  /// In en, this message translates to:
  /// **'grams (g)'**
  String get unitG;

  /// SAL-41: Dose unit choice.
  ///
  /// In en, this message translates to:
  /// **'micrograms (µg)'**
  String get unitUg;

  /// SAL-41: Dose unit choice.
  ///
  /// In en, this message translates to:
  /// **'millilitres (mL)'**
  String get unitMl;

  /// SAL-41: Dose unit choice.
  ///
  /// In en, this message translates to:
  /// **'international units (IU)'**
  String get unitIu;

  /// SAL-41: Dose unit choice.
  ///
  /// In en, this message translates to:
  /// **'tablets'**
  String get unitTablet;

  /// SAL-41: Dose unit choice.
  ///
  /// In en, this message translates to:
  /// **'capsules'**
  String get unitCapsule;

  /// SAL-41: Dose unit choice.
  ///
  /// In en, this message translates to:
  /// **'drops'**
  String get unitDrop;

  /// SAL-41: Dose unit choice.
  ///
  /// In en, this message translates to:
  /// **'puffs (inhaler)'**
  String get unitPuff;

  /// SAL-41: Dose unit choice.
  ///
  /// In en, this message translates to:
  /// **'patches'**
  String get unitPatch;

  /// SAL-41: Medication form: only one part of the dose.
  ///
  /// In en, this message translates to:
  /// **'Enter both the amount and the unit, or neither.'**
  String get doseBoth;

  /// SAL-41: A number field: 0 or less.
  ///
  /// In en, this message translates to:
  /// **'Enter a number above 0.'**
  String get numberAboveZero;

  /// SAL-41: Medication form heading.
  ///
  /// In en, this message translates to:
  /// **'How often (optional)'**
  String get howOftenHeading;

  /// SAL-41: Medication form: how many times per period.
  ///
  /// In en, this message translates to:
  /// **'Times'**
  String get timesLabel;

  /// SAL-41: Medication form: the period's number.
  ///
  /// In en, this message translates to:
  /// **'Every'**
  String get everyLabel;

  /// SAL-41: Medication form: the period's unit.
  ///
  /// In en, this message translates to:
  /// **'Unit of time'**
  String get periodUnitLabel;

  /// SAL-41: Unit of time choice.
  ///
  /// In en, this message translates to:
  /// **'hours'**
  String get periodHours;

  /// SAL-41: Unit of time choice.
  ///
  /// In en, this message translates to:
  /// **'days'**
  String get periodDays;

  /// SAL-41: Unit of time choice.
  ///
  /// In en, this message translates to:
  /// **'weeks'**
  String get periodWeeks;

  /// SAL-41: Unit of time choice.
  ///
  /// In en, this message translates to:
  /// **'months'**
  String get periodMonths;

  /// SAL-41: Medication form: how often only partly filled in.
  ///
  /// In en, this message translates to:
  /// **'Fill in all three, or leave them empty.'**
  String get howOftenAll;

  /// SAL-41: Medication form checkbox and card line.
  ///
  /// In en, this message translates to:
  /// **'As needed'**
  String get asNeeded;

  /// SAL-41: Medication status.
  ///
  /// In en, this message translates to:
  /// **'Taking it now'**
  String get medicationActive;

  /// SAL-41: Medication status.
  ///
  /// In en, this message translates to:
  /// **'Stopped'**
  String get medicationStopped;

  /// SAL-41: A dose; value is the number written for the language.
  ///
  /// In en, this message translates to:
  /// **'{value} mg'**
  String doseMg(String value);

  /// SAL-41: A dose.
  ///
  /// In en, this message translates to:
  /// **'{value} g'**
  String doseG(String value);

  /// SAL-41: A dose.
  ///
  /// In en, this message translates to:
  /// **'{value} µg'**
  String doseUg(String value);

  /// SAL-41: A dose.
  ///
  /// In en, this message translates to:
  /// **'{value} mL'**
  String doseMl(String value);

  /// SAL-41: A dose.
  ///
  /// In en, this message translates to:
  /// **'{value} IU'**
  String doseIu(String value);

  /// SAL-41: A dose; count is the number, value the number written for the language.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, one{{value} tablet} other{{value} tablets}}'**
  String doseTablet(num count, String value);

  /// SAL-41: A dose.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, one{{value} capsule} other{{value} capsules}}'**
  String doseCapsule(num count, String value);

  /// SAL-41: A dose.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, one{{value} drop} other{{value} drops}}'**
  String doseDrop(num count, String value);

  /// SAL-41: A dose (inhaler).
  ///
  /// In en, this message translates to:
  /// **'{count, plural, one{{value} puff} other{{value} puffs}}'**
  String dosePuff(num count, String value);

  /// SAL-41: A dose.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, one{{value} patch} other{{value} patches}}'**
  String dosePatch(num count, String value);

  /// SAL-41: Medication card; the dose (doseMg...).
  ///
  /// In en, this message translates to:
  /// **'Dose: {dose}'**
  String doseLine(String dose);

  /// SAL-41: Medication card: how many times per hour.
  ///
  /// In en, this message translates to:
  /// **'{frequency}× an hour'**
  String howOftenPerHour(int frequency);

  /// SAL-41: Medication card: how many times per day.
  ///
  /// In en, this message translates to:
  /// **'{frequency}× a day'**
  String howOftenPerDay(int frequency);

  /// SAL-41: Medication card: how many times per week.
  ///
  /// In en, this message translates to:
  /// **'{frequency}× a week'**
  String howOftenPerWeek(int frequency);

  /// SAL-41: Medication card: how many times per month.
  ///
  /// In en, this message translates to:
  /// **'{frequency}× a month'**
  String howOftenPerMonth(int frequency);

  /// SAL-41: Medication card: times per a period other than 1; every is the period written for the language.
  ///
  /// In en, this message translates to:
  /// **'{period, plural, one{{frequency}× every {every} hour} other{{frequency}× every {every} hours}}'**
  String howOftenEveryHours(num period, int frequency, String every);

  /// SAL-41: As howOftenEveryHours.
  ///
  /// In en, this message translates to:
  /// **'{period, plural, one{{frequency}× every {every} day} other{{frequency}× every {every} days}}'**
  String howOftenEveryDays(num period, int frequency, String every);

  /// SAL-41: As howOftenEveryHours.
  ///
  /// In en, this message translates to:
  /// **'{period, plural, one{{frequency}× every {every} week} other{{frequency}× every {every} weeks}}'**
  String howOftenEveryWeeks(num period, int frequency, String every);

  /// SAL-41: As howOftenEveryHours.
  ///
  /// In en, this message translates to:
  /// **'{period, plural, one{{frequency}× every {every} month} other{{frequency}× every {every} months}}'**
  String howOftenEveryMonths(num period, int frequency, String every);

  /// SAL-39: the home screen button and the title of the patient's history across every clinic.
  ///
  /// In en, this message translates to:
  /// **'My history'**
  String get myHistory;

  /// SAL-39: the history before it holds anything.
  ///
  /// In en, this message translates to:
  /// **'There is nothing in your history yet. Your visits, documents and answers appear here.'**
  String get historyEmpty;

  /// SAL-39: the history when the filters chosen leave nothing.
  ///
  /// In en, this message translates to:
  /// **'Nothing in your history matches these filters.'**
  String get historyNoMatch;

  /// SAL-39: the label of the history's clinic filter.
  ///
  /// In en, this message translates to:
  /// **'Clinic'**
  String get historyClinic;

  /// SAL-39: the clinic filter's choice that does not narrow.
  ///
  /// In en, this message translates to:
  /// **'All clinics'**
  String get historyAllClinics;

  /// SAL-39: the type filter for visits (consultations).
  ///
  /// In en, this message translates to:
  /// **'Visits'**
  String get historyVisits;

  /// SAL-39: the title of a visit in the history.
  ///
  /// In en, this message translates to:
  /// **'Visit'**
  String get historyVisit;

  /// SAL-39: the type filter for completed questionnaire answers, and the title of one in the history.
  ///
  /// In en, this message translates to:
  /// **'Questionnaire answers'**
  String get historyAnswers;

  /// SAL-39: the type filter for the files attached to questionnaire answers.
  ///
  /// In en, this message translates to:
  /// **'Files'**
  String get historyFiles;

  /// SAL-39: the title of a file in the history that has no name.
  ///
  /// In en, this message translates to:
  /// **'File'**
  String get historyFile;

  /// SAL-39: the date filter's button while no dates are chosen.
  ///
  /// In en, this message translates to:
  /// **'Any date'**
  String get historyAnyDate;

  /// SAL-39: the date filter's button with the days chosen, both long dates, the last included.
  ///
  /// In en, this message translates to:
  /// **'{from} to {to}'**
  String historyDateRange(String from, String to);

  /// SAL-39: removes the date filter.
  ///
  /// In en, this message translates to:
  /// **'Clear dates'**
  String get historyClearDates;

  /// SAL-39: loads the next page of the history (it also loads as the list scrolls).
  ///
  /// In en, this message translates to:
  /// **'Show more'**
  String get historyShowMore;

  /// SAL-39: under the history when its next page could not be read; Try again follows.
  ///
  /// In en, this message translates to:
  /// **'More of your history could not be loaded.'**
  String get historyMoreFailed;

  /// SAL-39: an item of the history could not be opened.
  ///
  /// In en, this message translates to:
  /// **'This item could not be opened. Try again.'**
  String get historyOpenFailed;

  /// [legal] SAL-42: booking, the switch that shares visits from other clinics with this clinic; off by default. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Share my history from other clinics with this clinic'**
  String get sharingToggle;

  /// [legal] SAL-42: above the visits to tick, at booking and in Shared history: exactly what the clinic will see, who sees it, and that a shared visit stays shared as it grows (owner, 2026-11-13). Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Tick the previous visits {clinic} may see. For each visit you tick, the doctors at {clinic} you have an appointment with see what is shown here, with the visit\'s documents, questionnaire answers and files, the allergies and medications recorded in it, and anything added to it later. Visits you do not tick stay hidden from {clinic}.'**
  String sharingIntro(String clinic);

  /// [legal] SAL-42: under the sharing intro: how to change or stop it, and that it takes effect at once. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'You can add or remove visits, or stop sharing, at any time under Shared history. Each change takes effect at once.'**
  String get sharingChangeLater;

  /// SAL-42: the preview when the patient has no finished visit at another clinic. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'You have no previous visits at other clinics to share.'**
  String get sharingNoVisits;

  /// SAL-42: button: tick every visit. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Select all'**
  String get sharingSelectAll;

  /// SAL-42: button: untick every visit. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Select none'**
  String get sharingSelectNone;

  /// SAL-42: button on My consents and the title of the screen that manages sharing per clinic. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Shared history'**
  String get sharedHistory;

  /// SAL-42: Shared history, above the list of clinics. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'The clinics you have consented to, and how many of your visits from other clinics each one sees.'**
  String get sharedHistoryIntro;

  /// SAL-42: Shared history without an active care consent. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'You have not consented to any clinic yet.'**
  String get sharedHistoryNone;

  /// SAL-42: Shared history, under a clinic: how many visits are shared with it. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =0{Nothing shared} one{{count} visit shared} other{{count} visits shared}}'**
  String sharedVisits(int count);

  /// [legal] SAL-42: after saving the visits shared with a clinic. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Your choice is saved and takes effect at once.'**
  String get sharingSaved;

  /// SAL-42: button: stop sharing history with the clinic (also the dialog's confirm button). Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Stop sharing'**
  String get sharingStop;

  /// [legal] SAL-42: dialog title before stopping the sharing. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Stop sharing your history?'**
  String get sharingStopTitle;

  /// [legal] SAL-42: dialog text: the effect of stopping the sharing. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'{clinic} will no longer see any of the visits from other clinics you shared with it. This takes effect at once.'**
  String sharingStopBody(String clinic);

  /// SAL-42: dialog button: keep the sharing. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Keep sharing'**
  String get sharingKeep;

  /// [legal] SAL-42: after the sharing is stopped. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'You no longer share your history with this clinic.'**
  String get sharingStopped;

  /// SAL-42: the sharing changed on another device since it was read; it is shown again. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Your shared visits changed in the meantime. Check them and save again.'**
  String get sharingChanged;

  /// SAL-42: after a booking whose sharing failed. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Your appointment is booked, but your history could not be shared. You can share it under Shared history.'**
  String get bookedNotShared;

  /// SAL-42: My consents, the label of a history consent. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Visits shared from other clinics'**
  String get consentForHistory;

  /// SAL-33: home button and screen title of the notification settings. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Reminders'**
  String get reminders;

  /// SAL-33: notification settings, the sentence under the title. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Choose how your clinics remind you of your appointments.'**
  String get remindersIntro;

  /// SAL-33: notification settings, the push switch. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Push notifications'**
  String get remindersPush;

  /// SAL-33: notification settings, the e-mail switch. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'E-mail'**
  String get remindersEmail;

  /// SAL-33: notification settings, the SMS switch. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'SMS'**
  String get remindersSms;

  /// [legal] SAL-33: notification settings, under the SMS switch: the SMS opt-in, what goes by SMS, to which number, where, and never advertising. Draft, waiting for the owner and the lawyer.
  ///
  /// In en, this message translates to:
  /// **'Reminders and waitlist offers by SMS to the phone number in your profile, only at clinics that offer SMS. Never used for advertising.'**
  String get remindersSmsHint;

  /// SAL-33: notification settings, the quiet hours switch. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Quiet hours'**
  String get quietHours;

  /// SAL-33: notification settings, under the quiet hours switch: push and SMS wait, e-mail is not held. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Push notifications and SMS wait until quiet hours end. If they would then arrive less than 30 minutes before the appointment, they are skipped. E-mails are not held.'**
  String get quietHoursHint;

  /// SAL-33: notification settings, the start of the quiet hours (a 24-hour time beside it). Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'From'**
  String get quietHoursFrom;

  /// SAL-33: notification settings, the end of the quiet hours (a 24-hour time beside it). Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Until'**
  String get quietHoursUntil;

  /// SAL-33: notification settings saved. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Your choices are saved.'**
  String get remindersSaved;

  /// SAL-33: notification settings refused, 409 phone_required. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'SMS needs a phone number in your profile. Add one in My profile, then try again.'**
  String get remindersPhoneRequired;

  /// SAL-33: notification settings refused, 400. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'These settings could not be saved. Check them and try again.'**
  String get remindersInvalid;

  /// SAL-34: button on an unconfirmed upcoming appointment; the patient confirms they will come. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Confirm I will come'**
  String get confirmAttendance;

  /// SAL-34: under an upcoming appointment the patient confirmed. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Confirmed'**
  String get appointmentConfirmedStatus;

  /// SAL-34: notice after confirming. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Your appointment is confirmed.'**
  String get appointmentConfirmedNotice;

  /// SAL-34: button opening the reschedule picker. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Change the time'**
  String get changeTime;

  /// SAL-34: until when the patient may change the time, in the clinic's zone. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'The time can be changed until {date} at {time}.'**
  String rescheduleUntil(String date, String time);

  /// SAL-34: how many more times the time may be changed (the clinic's limit). Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Time changes left: {count}'**
  String reschedulesLeft(int count);

  /// SAL-34: title of the reschedule picker. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Choose a new time'**
  String get rescheduleTitle;

  /// SAL-34: the appointment's current time on the picker. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Booked now: {date} at {time}'**
  String rescheduleCurrent(String date, String time);

  /// SAL-34: dialog title before moving the appointment. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Move your appointment?'**
  String get rescheduleConfirmTitle;

  /// [legal] SAL-34: dialog text, the effect of a reschedule. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Your appointment will move to {date} at {time}. Your current time will be released.'**
  String rescheduleConfirmBody(String date, String time);

  /// SAL-34: dialog button that moves the appointment. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Move appointment'**
  String get moveAppointment;

  /// SAL-34: notice after a reschedule. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'Your appointment is moved to the new time.'**
  String get appointmentMoved;

  /// SAL-34: reschedule refused, inside the clinic's notice. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'The time of this appointment can no longer be changed in the app. Please contact the clinic.'**
  String get rescheduleWindowClosed;

  /// SAL-34: reschedule refused, the clinic's limit reached. Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'This appointment\'s time has been changed as often as the clinic allows.'**
  String get rescheduleLimit;

  /// SAL-34: reschedule refused, the clinic's rules never allow it (another role, the join window open, self-service off). Draft, waiting for the owner.
  ///
  /// In en, this message translates to:
  /// **'The time of this appointment cannot be changed in the app.'**
  String get notReschedulable;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) => <String>[
    'ar',
    'bs',
    'de',
    'en',
    'fr',
    'hr',
    'it',
    'sr',
    'tr',
  ].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when language+script codes are specified.
  switch (locale.languageCode) {
    case 'sr':
      {
        switch (locale.scriptCode) {
          case 'Cyrl':
            return AppLocalizationsSrCyrl();
          case 'Latn':
            return AppLocalizationsSrLatn();
        }
        break;
      }
  }

  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'ar':
      return AppLocalizationsAr();
    case 'bs':
      return AppLocalizationsBs();
    case 'de':
      return AppLocalizationsDe();
    case 'en':
      return AppLocalizationsEn();
    case 'fr':
      return AppLocalizationsFr();
    case 'hr':
      return AppLocalizationsHr();
    case 'it':
      return AppLocalizationsIt();
    case 'sr':
      return AppLocalizationsSr();
    case 'tr':
      return AppLocalizationsTr();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
