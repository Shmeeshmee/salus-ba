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

  /// [legal] Dialog text: the effect of revoking.
  ///
  /// In en, this message translates to:
  /// **'{clinic} will no longer see your profile. It keeps your name and the details of appointments that have already taken place, as part of its care records. Your future appointments with {clinic} will be cancelled.'**
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

  /// [legal] M35: Android, before the system permission prompt.
  ///
  /// In en, this message translates to:
  /// **'To join the video consultation, Salus needs your camera and microphone. The consultation is not recorded.'**
  String get whyCameraAndMicrophone;

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
