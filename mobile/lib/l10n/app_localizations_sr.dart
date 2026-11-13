// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Serbian (`sr`).
class AppLocalizationsSr extends AppLocalizations {
  AppLocalizationsSr([String locale = 'sr']) : super(locale);

  @override
  String get tryAgain => 'Покушајте поново';

  @override
  String get language => 'Језик';

  @override
  String get useDeviceLanguage => 'Користи језик уређаја';

  @override
  String get languageSaved => 'Језик је сачуван.';

  @override
  String get somethingWrong => 'Дошло је до грешке. Покушајте поново.';

  @override
  String get unreachable =>
      'Salus тренутно није доступан. Провјерите интернет везу и покушајте поново.';

  @override
  String get wrongCredentials => 'Адреса е-поште или лозинка није тачна.';

  @override
  String get wrongCode =>
      'Код није тачан. Провјерите вријеме на уређају и покушајте поново.';

  @override
  String get challengeEnded =>
      'Пријава је истекла. Поново унесите адресу е-поште и лозинку.';

  @override
  String get emailTaken =>
      'За ову адресу е-поште већ постоји налог. Умјесто тога се пријавите.';

  @override
  String get invalidEmail => 'Унесите исправну адресу е-поште.';

  @override
  String get invalidPassword => 'Лозинка мора имати од 12 до 128 знакова.';

  @override
  String get missingPassword => 'Унесите лозинку.';

  @override
  String get invalidCode => 'Код има 6 цифара.';

  @override
  String get staffOnly =>
      'Ова апликација је намијењена пацијентима. Особље клинике користи веб-апликацију Salus.';

  @override
  String get accountReady =>
      'Ваш налог је спреман. Пријавите се да бисте наставили.';

  @override
  String get signOutFailed =>
      'Одјава није успјела. Провјерите интернет везу и покушајте поново.';

  @override
  String get tooManyAttemptsLater =>
      'Превише покушаја. Покушајте поново касније.';

  @override
  String tooManyAttempts(int minutes) {
    String _temp0 = intl.Intl.pluralLogic(
      minutes,
      locale: localeName,
      other: 'Превише покушаја. Покушајте поново за $minutes минута.',
      few: 'Превише покушаја. Покушајте поново за $minutes минута.',
      one: 'Превише покушаја. Покушајте поново за $minutes минут.',
    );
    return '$_temp0';
  }

  @override
  String get signInTitle => 'Пријава у Salus';

  @override
  String get email => 'Е-пошта';

  @override
  String get password => 'Лозинка';

  @override
  String get signIn => 'Пријавите се';

  @override
  String get createAnAccount => 'Направите налог';

  @override
  String get codePrompt =>
      'Унесите шестоцифрени код из апликације за аутентификацију.';

  @override
  String get code => 'Код';

  @override
  String get useAnotherAccount => 'Пријавите се другим налогом';

  @override
  String get passwordHint => 'Од 12 до 128 знакова';

  @override
  String get createAccount => 'Направите налог';

  @override
  String get haveAccount => 'Већ имам налог';

  @override
  String signedInAs(String email) {
    return 'Пријављени сте као $email';
  }

  @override
  String get completeProfile => 'Попуните свој профил';

  @override
  String get completeProfileFirst => 'Прије заказивања попуните свој профил.';

  @override
  String get bookAppointment => 'Закажите термин';

  @override
  String get myAppointments => 'Моји термини';

  @override
  String get myConsents => 'Моје сагласности';

  @override
  String get myProfile => 'Мој профил';

  @override
  String get signOut => 'Одјавите се';

  @override
  String get profileIntro =>
      'Клинике код којих закажете термин виде ове податке. Број телефона није обавезан; љекар га користи само да би вас контактирао у вези с термином.';

  @override
  String get givenName => 'Име';

  @override
  String get familyName => 'Презиме';

  @override
  String get dateOfBirth => 'Датум рођења';

  @override
  String get dateFormatHint => 'ГГГГ-ММ-ДД';

  @override
  String get phoneOptional => 'Телефон (није обавезно)';

  @override
  String phoneHint(String example) {
    return 'Са позивним бројем земље, на примјер $example';
  }

  @override
  String get save => 'Сачувајте';

  @override
  String get notNow => 'Не сада';

  @override
  String get invalidGivenName => 'Унесите име, највише 100 знакова.';

  @override
  String get invalidFamilyName => 'Унесите презиме, највише 100 знакова.';

  @override
  String get invalidName => 'Унесите име и презиме, свако највише 100 знакова.';

  @override
  String get birthDateFormat => 'Унесите датум рођења у облику ГГГГ-ММ-ДД.';

  @override
  String get birthDateRange =>
      'Унесите датум рођења у прошлости, од 1900. године надаље.';

  @override
  String invalidPhone(String example) {
    return 'Унесите број телефона са позивним бројем земље, на примјер $example.';
  }

  @override
  String get profileChanged =>
      'Ваш профил је у међувремену измијењен. Провјерите га и поново га сачувајте.';

  @override
  String get chooseClinic => 'Изаберите клинику';

  @override
  String get noClinics => 'Још нема клиника на списку.';

  @override
  String get noPractitioners => 'Ова клиника још нема љекара на списку.';

  @override
  String get choosePractitioner => 'Изаберите љекара.';

  @override
  String get noOpenTimes => 'Овај љекар још нема објављених термина.';

  @override
  String timesIn(String zone) {
    return 'Вријеме је приказано у временској зони $zone.';
  }

  @override
  String get previousDay => 'Претходни дан';

  @override
  String get nextDay => 'Сљедећи дан';

  @override
  String get noFreeTimes => 'Овог дана нема слободних термина.';

  @override
  String get bookThisTime => 'Закажите овај термин';

  @override
  String get chooseAnotherTime => 'Изаберите други термин';

  @override
  String get book => 'Закажите';

  @override
  String timeRange(String start, String end, String zone) {
    return '$start до $end ($zone)';
  }

  @override
  String get slotTaken =>
      'Овај термин више није слободан. Изаберите други термин.';

  @override
  String get consentNeeded =>
      'За заказивање код ове клинике потребна је ваша сагласност.';

  @override
  String get appointmentBooked => 'Ваш термин је заказан.';

  @override
  String get unlistedClinic => 'Клиника која више није на списку';

  @override
  String get cancelTitle => 'Отказати овај термин?';

  @override
  String cancelBody(String date, String time) {
    return 'Ваш термин заказан за $date у $time биће отказан. Потврду отказивања ћете добити е-поштом.';
  }

  @override
  String get keepIt => 'Задржите термин';

  @override
  String get cancelAppointment => 'Откажите термин';

  @override
  String get noAppointments => 'Још немате заказаних термина.';

  @override
  String get upcoming => 'Предстојећи';

  @override
  String get past => 'Прошли';

  @override
  String get statusBooked => 'Заказан';

  @override
  String get statusCancelled => 'Отказан';

  @override
  String get statusAttended => 'Обављен';

  @override
  String get statusNotAttended => 'Пропуштен';

  @override
  String get appointmentCancelled => 'Ваш термин је отказан.';

  @override
  String get cancelWindowHint =>
      'Термин можете отказати у апликацији најкасније 24 сата прије почетка.';

  @override
  String get cancelWindowClosed =>
      'Овај термин почиње за мање од 24 сата и више га није могуће отказати у апликацији.';

  @override
  String get appointmentClosed => 'Овај термин више није заказан.';

  @override
  String consentTitle(String clinic) {
    return 'Дијељење ваших података са клиником $clinic';
  }

  @override
  String consentIntro(String clinic) {
    return 'Прије вашег првог заказивања код клинике $clinic потребна нам је ваша сагласност да с њом подијелимо ваше податке.';
  }

  @override
  String get whatIsShared => 'Шта се дијели';

  @override
  String get whatIsSharedBody =>
      'Ваш профил (име, датум рођења и број телефона, ако сте га навели) и ваши термини код ове клинике.';

  @override
  String get whoSeesIt => 'Ко види податке';

  @override
  String whoSeesItBody(String clinic) {
    return '$clinic. Љекар код којег закажете термин види ваш профил; администратори клинике виде ваше име и вријеме ваших термина.';
  }

  @override
  String get whatFor => 'У коју сврху';

  @override
  String get whatForBody =>
      'Само за ваше лијечење у овој клиници: за припрему и одржавање ваших термина.';

  @override
  String get whoIsResponsible => 'Ко је одговоран';

  @override
  String whoIsResponsibleBody(String clinic) {
    return 'За ваше податке одговорна је клиника $clinic. Salus их обрађује у име клинике.';
  }

  @override
  String get readPrivacyNotice => 'Прочитајте обавјештење о приватности';

  @override
  String get privacyNoticeFailed =>
      'Обавјештење о приватности није могуће отворити.';

  @override
  String get howToRevoke => 'Како повући сагласност';

  @override
  String get howToRevokeBody =>
      'Ову сагласност можете повући у било ком тренутку у дијелу Моје сагласности. Клиника тада више не види ваш профил. Ваше име и податке о терминима који су већ одржани задржава као дио своје медицинске документације. Ваши будући термини код те клинике се отказују.';

  @override
  String get agreeAndBook => 'Прихватам и заказујем';

  @override
  String get revokeTitle => 'Повући сагласност?';

  @override
  String revokeBody(String clinic) {
    return 'Клиника $clinic више неће видјети ваш профил. Ваше име и податке о терминима који су већ одржани задржава као дио своје медицинске документације. Ваши будући термини код клинике $clinic биће отказани. Престаје и дијељење ваше историје из других клиника са клиником $clinic; поновни пристанак га не враћа.';
  }

  @override
  String get keepConsent => 'Задржите сагласност';

  @override
  String get revokeConsent => 'Повуците сагласност';

  @override
  String get noConsents => 'Још нисте дали ниједну сагласност.';

  @override
  String get consentsIntro =>
      'Са овим клиникама дијелите свој профил и термине ради свог лијечења.';

  @override
  String givenOn(String date) {
    return 'Дата $date';
  }

  @override
  String get consentActive => 'Активна';

  @override
  String revokedOn(String date) {
    return 'Повучена $date';
  }

  @override
  String get consentRevoked =>
      'Ваша сагласност је повучена. Ваши будући термини код ове клинике биће отказани и за сваки ћете добити обавјештење е-поштом.';

  @override
  String get consentAlreadyRevoked => 'Ова сагласност је већ повучена.';

  @override
  String get joinConsultation => 'Укључите се у консултацију';

  @override
  String get joinFrom => 'Можете се укључити од 10 минута прије почетка.';

  @override
  String get checkTitle => 'Провјерите камеру и микрофон';

  @override
  String get checkIntro => 'Љекар ће вас видјети и чути чим се укључите.';

  @override
  String get noSound =>
      'Звук се не чује. Провјерите да ли је микрофон укључен и да није утишан.';

  @override
  String get joinCameraOff => 'Укључите се без камере';

  @override
  String get joinMicrophoneOff => 'Укључите се без микрофона';

  @override
  String get joinNow => 'Укључите се';

  @override
  String get joining => 'Укључивање…';

  @override
  String get cameraDenied =>
      'Salus нема приступ вашој камери. Приступ можете дозволити у подешавањима уређаја или се укључити без видеа.';

  @override
  String get microphoneDenied =>
      'Salus нема приступ вашем микрофону. Приступ можете дозволити у подешавањима уређаја. Без микрофона вас љекар не може чути.';

  @override
  String get openSettings => 'Отворите подешавања';

  @override
  String get noCamera => 'Камера није пронађена.';

  @override
  String get noMicrophone => 'Микрофон није пронађен.';

  @override
  String get waiting => 'Чека се љекар';

  @override
  String joinsYouHere(String name) {
    return '$name ће вам се придружити овдје. Не затварајте овај екран.';
  }

  @override
  String get mute => 'Искључите микрофон';

  @override
  String get unmute => 'Укључите микрофон';

  @override
  String get cameraOff => 'Искључите камеру';

  @override
  String get cameraOn => 'Укључите камеру';

  @override
  String get leave => 'Изађите';

  @override
  String peerCameraOff(String name) {
    return '$name: камера је искључена';
  }

  @override
  String peerMuted(String name) {
    return '$name: микрофон је искључен';
  }

  @override
  String peerLeft(String name) {
    return '$name више није у консултацији. Може се поново укључити док се консултација не заврши.';
  }

  @override
  String get reconnecting => 'Веза је прекинута. Поновно повезивање…';

  @override
  String get notRestored => 'Везу није било могуће поново успоставити.';

  @override
  String get reconnected => 'Веза је поново успостављена';

  @override
  String get consultationOver => 'Консултација је завршена';

  @override
  String get closeScreen => 'Можете затворити овај екран.';

  @override
  String get timeLimit =>
      'Предвиђено вријеме за консултацију је истекло и консултација је завршена.';

  @override
  String get practitionerAbsent => 'Љекар се није укључио';

  @override
  String get timePassed =>
      'Вријеме за укључивање у ову консултацију је истекло. Клиника може видјети да сте се ви укључили. Клиника вас може контактирати, а можете и заказати нови термин.';

  @override
  String get withdrewConsent =>
      'Повукли сте сагласност за ову клинику, па сте искључени из консултације.';

  @override
  String get appointmentWasCancelled =>
      'Овај термин је отказан, па је консултација завршена.';

  @override
  String get windowNotOpen =>
      'Ова консултација се отвара 10 минута прије почетка.';

  @override
  String get consultationEnded => 'Ова консултација је завршена.';

  @override
  String get joinNeedsConsent =>
      'За укључивање је потребна ваша сагласност за ову клинику.';

  @override
  String get consultationNotAvailable => 'Ова консултација није доступна.';

  @override
  String get cannotConnect =>
      'Видео-везу није било могуће успоставити. Провјерите интернет везу и покушајте поново.';

  @override
  String get whyCameraAndMicrophone =>
      'Да бисте се укључили у видео-консултацију, Salus треба приступ вашој камери и микрофону. Консултација се не снима ни као звук ни као видео. Ако се сагласите, оно што се каже транскрибује се и преводи.';

  @override
  String get transcriptionOn => 'Транскрипција је укључена';

  @override
  String get transcriptionTitle => 'Транскрипција и превод';

  @override
  String transcriptionIntro(String clinic) {
    return 'У вашим видео-консултацијама с клиником $clinic оно што ви и љекар кажете може да се транскрибује док говорите и да се преведе, како би вас љекар могао пратити на вашем језику.';
  }

  @override
  String get transcriptionAssistance =>
      'На основу транскрипта љекар добија и помоћ вјештачке интелигенције (AI): приједлоге током консултације и нацрте докумената. То су само приједлози; љекар их провјерава и доноси сваку одлуку.';

  @override
  String transcriptionKept(String clinic) {
    return 'Транскрипт се чува уз вашу медицинску документацију у клиници $clinic. Консултација се не снима ни као звук ни као видео.';
  }

  @override
  String get transcriptionDecline =>
      'Ако се не сагласите, консултација тече као и обично, без транскрипције, превода и помоћи вјештачке интелигенције.';

  @override
  String get transcriptionWithdraw =>
      'Сагласност можете повући у било ком тренутку, и током консултације: додирните „Транскрипција је укључена” или отворите Моје сагласности. Транскрипција престаје одмах; оно што је до тада транскрибовано остаје у вашој документацији.';

  @override
  String get transcriptionAgree => 'Прихватите и наставите';

  @override
  String get transcriptionWithout => 'Наставите без транскрипције';

  @override
  String get stopTranscriptionTitle => 'Зауставити транскрипцију?';

  @override
  String stopTranscriptionBody(String clinic) {
    return 'Транскрипција, превод и помоћ вјештачке интелигенције одмах престају у вашим консултацијама с клиником $clinic. Консултација која је у току наставља се без њих, а ваши термини остају какви јесу. Оно што је до сада транскрибовано остаје у вашој документацији.';
  }

  @override
  String get keepTranscription => 'Задржите транскрипцију';

  @override
  String get stopTranscription => 'Зауставите транскрипцију';

  @override
  String get transcriptionStopped =>
      'Транскрипција је искључена. Ваше консултације с овом клиником теку без ње.';

  @override
  String get consentForCare => 'Дијељење ради лијечења';

  @override
  String get consentForTranscription =>
      'Транскрипција и помоћ вјештачке интелигенције';

  @override
  String get toFillIn => 'За попуњавање';

  @override
  String get inProgress => 'У току';

  @override
  String get submittedStatus => 'Послат';

  @override
  String get closedStatus => 'Затворен';

  @override
  String get fillIn => 'Попуните';

  @override
  String get continueFilling => 'Наставите';

  @override
  String get viewAnswers => 'Погледајте одговоре';

  @override
  String get submitBeforeStart =>
      'Молимо вас да га пошаљете прије почетка термина.';

  @override
  String get questionnaireClosed =>
      'Термин је почео, па овај упитник више није могуће попунити.';

  @override
  String savedAsYouGo(String clinic) {
    return 'Ваши одговори се чувају док попуњавате. Клиника $clinic их види тек када их пошаљете.';
  }

  @override
  String get saved => 'Сачувано';

  @override
  String get saving => 'Чување…';

  @override
  String get notSaved =>
      'Није сачувано. Провјерите интернет везу. Чување ће бити поново покушано.';

  @override
  String get requiredQuestion => 'Обавезно';

  @override
  String get moreInformation => 'Више информација';

  @override
  String get yourAnswer => 'Ваш одговор';

  @override
  String get chooseDate => 'Изаберите датум';

  @override
  String get selectOne => 'Изаберите један одговор.';

  @override
  String get selectAll => 'Изаберите све што се односи на вас.';

  @override
  String get yes => 'Да';

  @override
  String get no => 'Не';

  @override
  String get questionRequired => 'Ово питање је обавезно.';

  @override
  String get enterWholeNumber => 'Унесите цијели број.';

  @override
  String get enterNumber => 'Унесите број.';

  @override
  String get numberTooLarge => 'Овај број је превелик.';

  @override
  String upToCharacters(int max) {
    return 'Унесите највише $max знакова.';
  }

  @override
  String get enterValidDate => 'Унесите исправан датум.';

  @override
  String get submit => 'Пошаљите';

  @override
  String get submitTitle => 'Послати одговоре?';

  @override
  String submitBody(String clinic) {
    return 'Након слања их више не можете мијењати. Видјеће их ваш љекар у клиници $clinic.';
  }

  @override
  String get cancel => 'Одустаните';

  @override
  String get requiredMissing => 'Нисте одговорили на сва обавезна питања.';

  @override
  String get answersSubmitted => 'Одговори су послати';

  @override
  String get answersSubmittedBody =>
      'Ваш љекар их може видјети прије термина. Можете их погледати у дијелу Моји термини.';

  @override
  String get yourAnswers => 'Ваши одговори';

  @override
  String submittedAt(String when) {
    return 'Послато $when';
  }

  @override
  String dateAndTime(String date, String time) {
    return '$date, $time';
  }

  @override
  String get notAnswered => 'Без одговора';

  @override
  String get leaveUnsaved =>
      'Ваш посљедњи одговор још није сачуван. Желите ли ипак да изађете?';

  @override
  String get stay => 'Останите';

  @override
  String get fillingNeedsConsent =>
      'За попуњавање овог упитника потребна је ваша сагласност за ову клинику.';

  @override
  String get pastDue =>
      'Термин је почео, па овај упитник више није могуће мијењати.';

  @override
  String get withdrawnByClinic =>
      'Клиника је повукла овај упитник. Не морате га попуњавати.';

  @override
  String get alreadySubmitted => 'Већ сте послали овај упитник.';

  @override
  String get changedElsewhere =>
      'Ови одговори су измијењени на другом уређају. Приказани су најновији одговори.';

  @override
  String get questionnaireNotAvailable => 'Овај упитник није доступан.';

  @override
  String get done => 'Готово';

  @override
  String get addFile => 'Додајте датотеку';

  @override
  String get photosUpTo => 'Фотографије до 20 MB.';

  @override
  String get documentsUpTo => 'Документи (PDF) до 20 MB.';

  @override
  String get videosUpTo => 'Видео-снимци до 200 MB.';

  @override
  String get oneFileHint => '1 датотека.';

  @override
  String upToFilesHint(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'До $n датотека.',
      few: 'До $n датотеке.',
      one: 'До $n датотеке.',
    );
    return '$_temp0';
  }

  @override
  String get takePhoto => 'Снимите фотографију';

  @override
  String get recordVideo => 'Снимите видео';

  @override
  String get chooseFromPhotos => 'Изаберите из фотографија';

  @override
  String get chooseDocument => 'Изаберите документ';

  @override
  String get preparing => 'Припрема…';

  @override
  String uploadingPercent(int n) {
    return 'Слање… $n%';
  }

  @override
  String get checkingFile => 'Провјера датотеке…';

  @override
  String get added => 'Додато';

  @override
  String get notUploaded => 'Није послато. Провјерите интернет везу.';

  @override
  String get remove => 'Уклоните';

  @override
  String get removeFile => 'Уклоните датотеку';

  @override
  String get removeFileTitle => 'Уклонити ову датотеку?';

  @override
  String get removeFileBody => 'Биће обрисана из ваших одговора.';

  @override
  String get fileRejected =>
      'Ова датотека није прошла сигурносну провјеру и обрисана је. Изаберите другу датотеку.';

  @override
  String fileTooLarge(int max) {
    return 'Ова датотека је већа од $max MB.';
  }

  @override
  String get fileTypeRefused =>
      'Ова врста датотеке не може се приложити уз ово питање.';

  @override
  String tooManyFiles(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Уз ово питање можете приложити највише $n датотека.',
      few: 'Уз ово питање можете приложити највише $n датотеке.',
      one: 'Уз ово питање можете приложити највише $n датотеку.',
    );
    return '$_temp0';
  }

  @override
  String get filesFull =>
      'Датотеке у овом упитнику достигле су укупно 500 MB. Уклоните неку датотеку да бисте додали нову.';

  @override
  String get fileNotUploaded =>
      'Датотеку није било могуће послати. Покушајте поново.';

  @override
  String get cannotPrepareFile =>
      'Ову датотеку није било могуће припремити за слање. Изаберите другу датотеку.';

  @override
  String get filesBeingChecked =>
      'Неке датотеке се још провјеравају. Упитник можете послати када буду спремне.';

  @override
  String get leaveUploading =>
      'Датотека се још шаље. Желите ли ипак да изађете? Слање ће бити прекинуто.';

  @override
  String fileCount(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n датотека',
      few: '$n датотеке',
      one: '$n датотека',
    );
    return '$_temp0';
  }

  @override
  String get photoKind => 'Фотографија';

  @override
  String get documentKind => 'Документ';

  @override
  String get videoKind => 'Видео';

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
    return 'Слање датотеке $title, $n посто';
  }

  @override
  String fileLabel(String kind, String title, String status) {
    return '$kind, $title, $status';
  }

  @override
  String get allowCamera => 'Дозволите приступ камери';

  @override
  String get whyCameraForFiles =>
      'Salus треба приступ камери да бисте снимили фотографију или видео који додајете у овај упитник.';

  @override
  String get continueLabel => 'Наставите';

  @override
  String get cameraDeniedForFiles =>
      'Salus нема приступ вашој камери. Приступ можете дозволити у подешавањима уређаја или умјесто тога изабрати датотеку.';

  @override
  String get photosDenied =>
      'Salus нема приступ вашим фотографијама. Приступ можете дозволити у подешавањима уређаја.';

  @override
  String cameraPhotoTitle(String date, String time) {
    return 'Фотографија, $date, $time';
  }

  @override
  String cameraVideoTitle(String date, String time) {
    return 'Видео, $date, $time';
  }

  @override
  String get specialty394814009 => 'Породична медицина';

  @override
  String get specialty419192003 => 'Интерна медицина';

  @override
  String get specialty394579002 => 'Кардиологија';

  @override
  String get specialty394582007 => 'Дерматологија';

  @override
  String get specialty394586005 => 'Гинекологија';

  @override
  String get specialty394537008 => 'Педијатрија';

  @override
  String get specialty394587001 => 'Психијатрија';

  @override
  String get specialty394591006 => 'Неурологија';

  @override
  String get specialty418960008 => 'Оториноларингологија (ОРЛ)';

  @override
  String get specialty394594003 => 'Офталмологија';

  @override
  String get specialty394801008 => 'Ортопедија';

  @override
  String get specialty394583002 => 'Ендокринологија';

  @override
  String questionnaireNotInLanguage(String language) {
    return 'Овај упитник није доступан на језику: $language.';
  }

  @override
  String questionnaireLanguages(String languages) {
    return 'Доступан је на овим језицима: $languages. Изаберите онај који најбоље разумијете.';
  }

  @override
  String continueInLanguage(String language) {
    return 'Наставите на језику: $language';
  }

  @override
  String get languageUnavailable =>
      'Овај језик више није доступан за овај упитник. Изаберите други.';

  @override
  String specialtyAtClinic(String specialty, String clinic) {
    return '$specialty, $clinic';
  }

  @override
  String get documents => 'Документи';

  @override
  String get noDocuments =>
      'За овај термин још нема потписаних докумената. Документ се појављује овдје када га ваш љекар потпише.';

  @override
  String documentSignedOn(String date) {
    return 'Потписано $date';
  }

  @override
  String documentLanguage(String language) {
    return 'Језик: $language';
  }

  @override
  String get sharePdf => 'Сачувај или подијели PDF';

  @override
  String get transcript => 'Транскрипт';

  @override
  String get transcriptNote =>
      'Консултација онако како је аутоматски транскрибована и преведена. Може да садржи грешке.';

  @override
  String get transcriptEmpty =>
      'У овој консултацији ништа није транскрибовано.';

  @override
  String get speakerYou => 'Ви';

  @override
  String get speakerPractitioner => 'Љекар';

  @override
  String get shareTranscript => 'Подијели транскрипт';

  @override
  String transcriptTranslation(String text) {
    return 'Пријевод: $text';
  }

  @override
  String get termExplanationNote => 'Информативно, није дијагноза';

  @override
  String get termNotReviewed => 'Још није провјерено';

  @override
  String get termClose => 'Затвори';

  @override
  String get chatsTitle => 'Поруке';

  @override
  String get chatsEmpty => 'Још немате разговора.';

  @override
  String chatUnread(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count непрочитаних порука',
      few: '$count непрочитане поруке',
      one: '$count непрочитана порука',
    );
    return '$_temp0';
  }

  @override
  String get chatDoctorUnknown => 'Ваш љекар';

  @override
  String get chatMessageHint => 'Напишите поруку';

  @override
  String get chatSend => 'Пошаљи';

  @override
  String get chatAttach => 'Додај датотеку';

  @override
  String get chatSendFailed =>
      'Порука није послата. Провјерите интернет везу и покушајте поново.';

  @override
  String get chatFileRejected => 'Ова датотека није прихваћена.';

  @override
  String get chatFileRemove => 'Уклони датотеку';

  @override
  String chatTooManyFiles(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'Већ имате $count датотека које чекају слање. Непослате датотеке се уклањају након 24 сата; покушајте поново касније.',
      few:
          'Већ имате $count датотеке које чекају слање. Непослате датотеке се уклањају након 24 сата; покушајте поново касније.',
      one:
          'Већ имате $count датотеку која чека слање. Непослате датотеке се уклањају након 24 сата; покушајте поново касније.',
    );
    return '$_temp0';
  }

  @override
  String get chatOpenFailed =>
      'Датотека није могла да се отвори. Покушајте поново.';

  @override
  String get chatEndedSession =>
      'Ваша сесија је истекла. Поново се пријавите да бисте видјели нове поруке.';

  @override
  String get chatEnded =>
      'Нове поруке у овом разговору више не могу да се прикажу.';

  @override
  String get chatHours => 'Радно вријеме за поруке';

  @override
  String chatHoursDay(String day, String start, String end) {
    return '$day: $start–$end';
  }

  @override
  String get chatHoursNone => 'Клиника није одредила радно вријеме за поруке.';

  @override
  String chatHoursZone(String zone) {
    return 'Времена су у временској зони клинике, $zone.';
  }

  @override
  String chatResponseTime(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other: 'Клиника обично одговара у року од $hours сати.',
      few: 'Клиника обично одговара у року од $hours сата.',
      one: 'Клиника обично одговара у року од $hours сата.',
    );
    return '$_temp0';
  }

  @override
  String chatOutOfHours(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other:
          'Клиника је тренутно ван радног времена за поруке. И даље можете да пишете; обично одговара у року од $hours сати.',
      few:
          'Клиника је тренутно ван радног времена за поруке. И даље можете да пишете; обично одговара у року од $hours сата.',
      one:
          'Клиника је тренутно ван радног времена за поруке. И даље можете да пишете; обично одговара у року од $hours сата.',
    );
    return '$_temp0';
  }

  @override
  String chatSystemInvited(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Љекар из друге клинике је позван у овај разговор ради другог мишљења.',
      'specialist_review': 'Љекар из друге клинике је позван у овај разговор ради специјалистичког прегледа.',
      'handover': 'Љекар из друге клинике је позван у овај разговор да преузме лијечење.',
      'other': 'Љекар из друге клинике је позван у овај разговор.',
    });
    return '$_temp0';
  }

  @override
  String chatSystemAdded(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Љекар се придружио разговору ради другог мишљења.',
      'specialist_review':
          'Љекар се придружио разговору ради специјалистичког прегледа.',
      'handover': 'Љекар се придружио разговору да преузме лијечење.',
      'other': 'Љекар се придружио разговору.',
    });
    return '$_temp0';
  }

  @override
  String get chatSystemRemoved => 'Љекар више не учествује у разговору.';

  @override
  String get chatSystemGranted =>
      'Дозволили сте љекару из друге клинике да се придружи разговору.';

  @override
  String get chatSystemDeclined => 'Одбили сте љекара из друге клинике.';

  @override
  String get chatSystemOther => 'Разговор је ажуриран.';

  @override
  String get chatInviteTitle =>
      'Љекар из друге клинике тражи приступ разговору';

  @override
  String chatInviteBody(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Ваша клиника тражи да се љекар из друге клинике придружи овом разговору ради другог мишљења.',
      'specialist_review': 'Ваша клиника тражи да се љекар из друге клинике придружи овом разговору ради специјалистичког прегледа.',
      'handover': 'Ваша клиника тражи да се љекар из друге клинике придружи овом разговору да преузме ваше лијечење.',
      'other': 'Ваша клиника тражи да се љекар из друге клинике придружи овом разговору.',
    });
    return '$_temp0';
  }

  @override
  String chatInviteWho(String doctor, String clinic) {
    return 'Љекар: $doctor, $clinic';
  }

  @override
  String get chatInviteUnresolved =>
      'Име и клиника позваног љекара нису могли да се учитају. Приступ можете да одобрите тек када буду приказани.';

  @override
  String chatInviteSees(String history) {
    String _temp0 = intl.Intl.selectLogic(history, {
      'full': 'Ако пристанете, љекар види цијели разговор, укључујући раније поруке, и може да пише у њему.',
      'from': 'Ако пристанете, љекар види разговор од поруке коју је изабрала ваша клиника надаље и може да пише у њему.',
      'other': 'Ако пристанете, љекар види разговор и може да пише у њему.',
    });
    return '$_temp0';
  }

  @override
  String get chatInviteChoice =>
      'Можете да одбијете. Ако пристанете, приступ љекара можете касније да опозовете у овом разговору; престаје одмах.';

  @override
  String get chatGrant => 'Одобри приступ';

  @override
  String get chatDecline => 'Одбиј';

  @override
  String get chatGranted =>
      'Приступ је одобрен. Можете да га опозовете у овом разговору у било ком тренутку.';

  @override
  String get chatDeclined => 'Одбијено. Љекар се не придружује овом разговору.';

  @override
  String get chatRevoke => 'Опозови приступ љекара из друге клинике';

  @override
  String chatGrantedTo(String doctor, String clinic) {
    return 'Приступ је одобрен: $doctor, $clinic.';
  }

  @override
  String get chatGrantedToUnknown =>
      'Приступ је одобрен љекару из друге клинике.';

  @override
  String get chatRevokeBody =>
      'Љекар из друге клинике одмах губи приступ овом разговору. Већ написане поруке остају у њему.';

  @override
  String get chatRevokeConfirm => 'Опозови приступ';

  @override
  String get chatRevoked => 'Приступ је опозван.';

  @override
  String get allergies => 'Алергије';

  @override
  String get currentMedications => 'Тренутни лијекови';

  @override
  String get allergiesIntro =>
      'Клинике којима сте дали сагласност виде алергије из вашег профила када имате термин код њих.';

  @override
  String get medicationsIntro =>
      'Клинике којима сте дали сагласност виде лијекове из вашег профила када имате термин код њих.';

  @override
  String get noAllergies => 'Нисте додали ниједну алергију.';

  @override
  String get noMedications => 'Нисте додали ниједан лијек.';

  @override
  String get addAllergy => 'Додајте алергију';

  @override
  String get addMedication => 'Додајте лијек';

  @override
  String get changeAllergy => 'Измјена алергије';

  @override
  String get changeMedication => 'Измјена лијека';

  @override
  String get change => 'Измијените';

  @override
  String get allergySubstance => 'Супстанца или намирница';

  @override
  String get medicationName => 'Лијек';

  @override
  String get termSearchHint =>
      'Почните да куцате за претрагу или упишите својим ријечима.';

  @override
  String get allergyCategory => 'Врста (није обавезно)';

  @override
  String get notSpecified => 'Није наведено';

  @override
  String get categoryFood => 'Храна';

  @override
  String get categoryMedication => 'Лијек';

  @override
  String get categoryEnvironment => 'Окружење';

  @override
  String get categoryBiologic => 'Биолошки препарат';

  @override
  String get reactionOptional => 'Реакција (није обавезно)';

  @override
  String get severity => 'Тежина';

  @override
  String get severityMild => 'Блага';

  @override
  String get severityModerate => 'Умјерена';

  @override
  String get severitySevere => 'Тешка';

  @override
  String get entryStatus => 'Статус';

  @override
  String get allergyActive => 'Активна';

  @override
  String get allergyInactive => 'Неактивна';

  @override
  String get allergyResolved => 'Прошла';

  @override
  String reactionLine(String reaction) {
    return 'Реакција: $reaction';
  }

  @override
  String reactionWithSeverity(String reaction, String severity) {
    return 'Реакција: $reaction ($severity)';
  }

  @override
  String get confirmedByClinician => 'Потврдио љекар';

  @override
  String get recordedByClinic => 'Унијела клиника током једне од ваших посјета';

  @override
  String get confirmedChangeNote =>
      'Ако измијените овај унос, више неће бити потврђен од љекара.';

  @override
  String get nameRequired => 'Унесите назив, највише 200 знакова.';

  @override
  String get addToProfile => 'Додајте у мој профил';

  @override
  String get addToProfileTitle => 'Додати у ваш профил?';

  @override
  String get addToProfileBody =>
      'Тада ће га видјети свака клиника којој сте дали сагласност, без назива клинике која га је унијела.';

  @override
  String get addedToProfile => 'Сада је у вашем профилу.';

  @override
  String get removeEntryTitle => 'Уклонити овај унос?';

  @override
  String get removeEntryBody =>
      'Више се неће приказивати ни вама ни било којој клиници. Ако га је клиника унијела или потврдила, остаје у вашем медицинском картону, означен као уклоњен.';

  @override
  String get entryRemoved => 'Унос је уклоњен.';

  @override
  String get entrySaved => 'Сачувано у вашем профилу.';

  @override
  String get entryChanged =>
      'Овај унос је у међувремену измијењен. Приказана је најновија верзија.';

  @override
  String get entryInvalid =>
      'Неки подаци нису исправни. Провјерите их и покушајте поново.';

  @override
  String get entryNotAllowed => 'Не можете измијенити овај унос.';

  @override
  String get entryNotFound => 'Овај унос више није доступан.';

  @override
  String get doseHeading => 'Доза (није обавезно)';

  @override
  String get doseAmount => 'Количина';

  @override
  String get doseUnit => 'Јединица';

  @override
  String get unitMg => 'милиграми (мг)';

  @override
  String get unitG => 'грами (г)';

  @override
  String get unitUg => 'микрограми (µг)';

  @override
  String get unitMl => 'милилитри (мл)';

  @override
  String get unitIu => 'интернационалне јединице (ИЈ)';

  @override
  String get unitTablet => 'таблете';

  @override
  String get unitCapsule => 'капсуле';

  @override
  String get unitDrop => 'капи';

  @override
  String get unitPuff => 'удисаји (инхалатор)';

  @override
  String get unitPatch => 'фластери';

  @override
  String get doseBoth => 'Унесите и количину и јединицу или ништа од тога.';

  @override
  String get numberAboveZero => 'Унесите број већи од 0.';

  @override
  String get howOftenHeading => 'Колико често (није обавезно)';

  @override
  String get timesLabel => 'Пута';

  @override
  String get everyLabel => 'Сваких';

  @override
  String get periodUnitLabel => 'Јединица времена';

  @override
  String get periodHours => 'сати';

  @override
  String get periodDays => 'дана';

  @override
  String get periodWeeks => 'недјеља';

  @override
  String get periodMonths => 'мјесеци';

  @override
  String get howOftenAll => 'Попуните сва три поља или их оставите празна.';

  @override
  String get asNeeded => 'По потреби';

  @override
  String get medicationActive => 'Тренутно узимам';

  @override
  String get medicationStopped => 'Више не узимам';

  @override
  String doseMg(String value) {
    return '$value мг';
  }

  @override
  String doseG(String value) {
    return '$value г';
  }

  @override
  String doseUg(String value) {
    return '$value µг';
  }

  @override
  String doseMl(String value) {
    return '$value мл';
  }

  @override
  String doseIu(String value) {
    return '$value ИЈ';
  }

  @override
  String doseTablet(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value таблета',
      few: '$value таблете',
      one: '$value таблета',
    );
    return '$_temp0';
  }

  @override
  String doseCapsule(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value капсула',
      few: '$value капсуле',
      one: '$value капсула',
    );
    return '$_temp0';
  }

  @override
  String doseDrop(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value капи',
      few: '$value капи',
      one: '$value кап',
    );
    return '$_temp0';
  }

  @override
  String dosePuff(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value удисаја',
      few: '$value удисаја',
      one: '$value удисај',
    );
    return '$_temp0';
  }

  @override
  String dosePatch(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value фластера',
      few: '$value фластера',
      one: '$value фластер',
    );
    return '$_temp0';
  }

  @override
  String doseLine(String dose) {
    return 'Доза: $dose';
  }

  @override
  String howOftenPerHour(int frequency) {
    return '$frequency× на сат';
  }

  @override
  String howOftenPerDay(int frequency) {
    return '$frequency× дневно';
  }

  @override
  String howOftenPerWeek(int frequency) {
    return '$frequency× недјељно';
  }

  @override
  String howOftenPerMonth(int frequency) {
    return '$frequency× мјесечно';
  }

  @override
  String howOftenEveryHours(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× сваких $every сати',
      few: '$frequency× свака $every сата',
      one: '$frequency× сваки $every сат',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryDays(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× сваких $every дана',
      few: '$frequency× свака $every дана',
      one: '$frequency× сваки $every дан',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryWeeks(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× сваких $every недјеља',
      few: '$frequency× сваке $every недјеље',
      one: '$frequency× сваке $every недјеље',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryMonths(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× сваких $every мјесеци',
      few: '$frequency× свака $every мјесеца',
      one: '$frequency× сваки $every мјесец',
    );
    return '$_temp0';
  }

  @override
  String get myHistory => 'Моја историја';

  @override
  String get historyEmpty =>
      'Ваша историја је још празна. Овдје се појављују ваше посјете, документи и одговори.';

  @override
  String get historyNoMatch =>
      'Ништа у вашој историји не одговара овим филтерима.';

  @override
  String get historyClinic => 'Клиника';

  @override
  String get historyAllClinics => 'Све клинике';

  @override
  String get historyVisits => 'Посјете';

  @override
  String get historyVisit => 'Посјета';

  @override
  String get historyAnswers => 'Одговори на упитник';

  @override
  String get historyFiles => 'Датотеке';

  @override
  String get historyFile => 'Датотека';

  @override
  String get historyAnyDate => 'Било који датум';

  @override
  String historyDateRange(String from, String to) {
    return 'од $from до $to';
  }

  @override
  String get historyClearDates => 'Поништи датуме';

  @override
  String get historyShowMore => 'Прикажи више';

  @override
  String get historyMoreFailed =>
      'Остатак ваше историје није се могао учитати.';

  @override
  String get historyOpenFailed =>
      'Ова ставка се није могла отворити. Покушајте поново.';

  @override
  String get sharingToggle =>
      'Подијели моју историју из других клиника са овом клиником';

  @override
  String sharingIntro(String clinic) {
    return 'Означите раније посјете које $clinic смије да види. За сваку означену посјету љекари клинике $clinic код којих имате термин виде оно што је овдје приказано, са документима посјете, одговорима на упитнике и датотекама, алергијама и лијековима евидентираним у њој, као и свиме што се посјети касније дода. Посјете које не означите остају скривене од $clinic.';
  }

  @override
  String get sharingChangeLater =>
      'Посјете можете да додате или уклоните, или да престанете да дијелите, у било ком тренутку у дијелу Подијељена историја. Свака промјена важи одмах.';

  @override
  String get sharingNoVisits =>
      'Немате ранијих посјета у другим клиникама које бисте могли да подијелите.';

  @override
  String get sharingSelectAll => 'Означите све';

  @override
  String get sharingSelectNone => 'Поништите све';

  @override
  String get sharedHistory => 'Подијељена историја';

  @override
  String get sharedHistoryIntro =>
      'Клинике којима сте дали сагласност и колико ваших посјета из других клиника свака од њих види.';

  @override
  String get sharedHistoryNone => 'Још нисте дали сагласност ниједној клиници.';

  @override
  String sharedVisits(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count подијељених посјета',
      few: '$count подијељене посјете',
      one: '$count подијељена посјета',
      zero: 'Ништа није подијељено',
    );
    return '$_temp0';
  }

  @override
  String get sharingSaved => 'Ваш избор је сачуван и важи одмах.';

  @override
  String get sharingStop => 'Престаните да дијелите';

  @override
  String get sharingStopTitle => 'Престати са дијељењем ваше историје?';

  @override
  String sharingStopBody(String clinic) {
    return '$clinic више неће видјети ниједну посјету из других клиника коју сте са њом подијелили. Ово важи одмах.';
  }

  @override
  String get sharingKeep => 'Наставите да дијелите';

  @override
  String get sharingStopped =>
      'Више не дијелите своју историју са овом клиником.';

  @override
  String get sharingChanged =>
      'Ваше подијељене посјете су се у међувремену промијениле. Провјерите их и поново сачувајте.';

  @override
  String get bookedNotShared =>
      'Ваш термин је заказан, али ваша историја није могла да буде подијељена. Можете да је подијелите у дијелу Подијељена историја.';

  @override
  String get consentForHistory => 'Посјете подијељене из других клиника';

  @override
  String get waitlistTitle => 'Моја листа чекања';

  @override
  String get waitlistJoinButton =>
      'Нема одговарајућег термина? Пријавите се на листу чекања';

  @override
  String get waitlistJoinTitle => 'Пријава на листу чекања';

  @override
  String get waitlistExplained =>
      'Када се ослободи термин код овог љекара, нуди се пацијентима на чекању, једном по једном. Добићете обавјештење и термин можете да закажете у апликацији док је резервисан за вас.';

  @override
  String get waitlistHowLong => 'Колико дуго?';

  @override
  String waitlistDays(int days) {
    String _temp0 = intl.Intl.pluralLogic(
      days,
      locale: localeName,
      other: '$days дана',
      few: '$days дана',
      one: '$days дан',
    );
    return '$_temp0';
  }

  @override
  String get waitlistWhen => 'У које доба дана?';

  @override
  String get waitlistAnyTime => 'Било када';

  @override
  String get waitlistMornings => 'Ујутру, од 08:00 до 12:00';

  @override
  String get waitlistAfternoons => 'Послијеподне, од 12:00 до 17:00';

  @override
  String get waitlistJoin => 'Пријави се на листу чекања';

  @override
  String get waitlistJoined => 'На листи чекања сте.';

  @override
  String get waitlistEmpty => 'Нисте ни на једној листи чекања.';

  @override
  String get waitlistAPractitioner => 'Љекар';

  @override
  String waitlistUntil(String date) {
    return 'Чекате до $date';
  }

  @override
  String get waitlistOfferTitle => 'Ослободио се термин за вас';

  @override
  String waitlistHeldUntil(String time) {
    return 'Резервисан за вас до $time';
  }

  @override
  String get waitlistAcceptEffect =>
      'Прихватањем заказујете овај термин. Када плаћање буде доступно у апликацији Salus, прихватањем ћете га и платити.';

  @override
  String get waitlistAccept => 'Прихвати и закажи';

  @override
  String get waitlistDecline => 'Одбиј';

  @override
  String get waitlistBooked =>
      'Ваш термин је заказан. Наћи ћете га у одјељку Моји термини.';

  @override
  String get waitlistDeclined => 'Одбили сте термин. Остајете на листи чекања.';

  @override
  String get waitlistWithdraw => 'Напусти листу чекања';

  @override
  String get waitlistWithdrawn => 'Напустили сте листу чекања.';

  @override
  String get waitlistOff => 'Ова установа нема листу чекања.';

  @override
  String get waitlistTooMany =>
      'Већ сте пет пута на листи чекања ове установе. Прво напустите једну од пријава.';

  @override
  String get waitlistOfferExpired =>
      'Овај термин више није резервисан за вас. Остајете на листи чекања.';
}

/// The translations for Serbian, using the Cyrillic script (`sr_Cyrl`).
class AppLocalizationsSrCyrl extends AppLocalizationsSr {
  AppLocalizationsSrCyrl() : super('sr_Cyrl');

  @override
  String get tryAgain => 'Покушајте поново';

  @override
  String get language => 'Језик';

  @override
  String get useDeviceLanguage => 'Користи језик уређаја';

  @override
  String get languageSaved => 'Језик је сачуван.';

  @override
  String get somethingWrong => 'Дошло је до грешке. Покушајте поново.';

  @override
  String get unreachable =>
      'Salus тренутно није доступан. Провјерите интернет везу и покушајте поново.';

  @override
  String get wrongCredentials => 'Адреса е-поште или лозинка није тачна.';

  @override
  String get wrongCode =>
      'Код није тачан. Провјерите вријеме на уређају и покушајте поново.';

  @override
  String get challengeEnded =>
      'Пријава је истекла. Поново унесите адресу е-поште и лозинку.';

  @override
  String get emailTaken =>
      'За ову адресу е-поште већ постоји налог. Умјесто тога се пријавите.';

  @override
  String get invalidEmail => 'Унесите исправну адресу е-поште.';

  @override
  String get invalidPassword => 'Лозинка мора имати од 12 до 128 знакова.';

  @override
  String get missingPassword => 'Унесите лозинку.';

  @override
  String get invalidCode => 'Код има 6 цифара.';

  @override
  String get staffOnly =>
      'Ова апликација је намијењена пацијентима. Особље клинике користи веб-апликацију Salus.';

  @override
  String get accountReady =>
      'Ваш налог је спреман. Пријавите се да бисте наставили.';

  @override
  String get signOutFailed =>
      'Одјава није успјела. Провјерите интернет везу и покушајте поново.';

  @override
  String get tooManyAttemptsLater =>
      'Превише покушаја. Покушајте поново касније.';

  @override
  String tooManyAttempts(int minutes) {
    String _temp0 = intl.Intl.pluralLogic(
      minutes,
      locale: localeName,
      other: 'Превише покушаја. Покушајте поново за $minutes минута.',
      few: 'Превише покушаја. Покушајте поново за $minutes минута.',
      one: 'Превише покушаја. Покушајте поново за $minutes минут.',
    );
    return '$_temp0';
  }

  @override
  String get signInTitle => 'Пријава у Salus';

  @override
  String get email => 'Е-пошта';

  @override
  String get password => 'Лозинка';

  @override
  String get signIn => 'Пријавите се';

  @override
  String get createAnAccount => 'Направите налог';

  @override
  String get codePrompt =>
      'Унесите шестоцифрени код из апликације за аутентификацију.';

  @override
  String get code => 'Код';

  @override
  String get useAnotherAccount => 'Пријавите се другим налогом';

  @override
  String get passwordHint => 'Од 12 до 128 знакова';

  @override
  String get createAccount => 'Направите налог';

  @override
  String get haveAccount => 'Већ имам налог';

  @override
  String signedInAs(String email) {
    return 'Пријављени сте као $email';
  }

  @override
  String get completeProfile => 'Попуните свој профил';

  @override
  String get completeProfileFirst => 'Прије заказивања попуните свој профил.';

  @override
  String get bookAppointment => 'Закажите термин';

  @override
  String get myAppointments => 'Моји термини';

  @override
  String get myConsents => 'Моје сагласности';

  @override
  String get myProfile => 'Мој профил';

  @override
  String get signOut => 'Одјавите се';

  @override
  String get profileIntro =>
      'Клинике код којих закажете термин виде ове податке. Број телефона није обавезан; љекар га користи само да би вас контактирао у вези с термином.';

  @override
  String get givenName => 'Име';

  @override
  String get familyName => 'Презиме';

  @override
  String get dateOfBirth => 'Датум рођења';

  @override
  String get dateFormatHint => 'ГГГГ-ММ-ДД';

  @override
  String get phoneOptional => 'Телефон (није обавезно)';

  @override
  String phoneHint(String example) {
    return 'Са позивним бројем земље, на примјер $example';
  }

  @override
  String get save => 'Сачувајте';

  @override
  String get notNow => 'Не сада';

  @override
  String get invalidGivenName => 'Унесите име, највише 100 знакова.';

  @override
  String get invalidFamilyName => 'Унесите презиме, највише 100 знакова.';

  @override
  String get invalidName => 'Унесите име и презиме, свако највише 100 знакова.';

  @override
  String get birthDateFormat => 'Унесите датум рођења у облику ГГГГ-ММ-ДД.';

  @override
  String get birthDateRange =>
      'Унесите датум рођења у прошлости, од 1900. године надаље.';

  @override
  String invalidPhone(String example) {
    return 'Унесите број телефона са позивним бројем земље, на примјер $example.';
  }

  @override
  String get profileChanged =>
      'Ваш профил је у међувремену измијењен. Провјерите га и поново га сачувајте.';

  @override
  String get chooseClinic => 'Изаберите клинику';

  @override
  String get noClinics => 'Још нема клиника на списку.';

  @override
  String get noPractitioners => 'Ова клиника још нема љекара на списку.';

  @override
  String get choosePractitioner => 'Изаберите љекара.';

  @override
  String get noOpenTimes => 'Овај љекар још нема објављених термина.';

  @override
  String timesIn(String zone) {
    return 'Вријеме је приказано у временској зони $zone.';
  }

  @override
  String get previousDay => 'Претходни дан';

  @override
  String get nextDay => 'Сљедећи дан';

  @override
  String get noFreeTimes => 'Овог дана нема слободних термина.';

  @override
  String get bookThisTime => 'Закажите овај термин';

  @override
  String get chooseAnotherTime => 'Изаберите други термин';

  @override
  String get book => 'Закажите';

  @override
  String timeRange(String start, String end, String zone) {
    return '$start до $end ($zone)';
  }

  @override
  String get slotTaken =>
      'Овај термин више није слободан. Изаберите други термин.';

  @override
  String get consentNeeded =>
      'За заказивање код ове клинике потребна је ваша сагласност.';

  @override
  String get appointmentBooked => 'Ваш термин је заказан.';

  @override
  String get unlistedClinic => 'Клиника која више није на списку';

  @override
  String get cancelTitle => 'Отказати овај термин?';

  @override
  String cancelBody(String date, String time) {
    return 'Ваш термин заказан за $date у $time биће отказан. Потврду отказивања ћете добити е-поштом.';
  }

  @override
  String get keepIt => 'Задржите термин';

  @override
  String get cancelAppointment => 'Откажите термин';

  @override
  String get noAppointments => 'Још немате заказаних термина.';

  @override
  String get upcoming => 'Предстојећи';

  @override
  String get past => 'Прошли';

  @override
  String get statusBooked => 'Заказан';

  @override
  String get statusCancelled => 'Отказан';

  @override
  String get statusAttended => 'Обављен';

  @override
  String get statusNotAttended => 'Пропуштен';

  @override
  String get appointmentCancelled => 'Ваш термин је отказан.';

  @override
  String get cancelWindowHint =>
      'Термин можете отказати у апликацији најкасније 24 сата прије почетка.';

  @override
  String get cancelWindowClosed =>
      'Овај термин почиње за мање од 24 сата и више га није могуће отказати у апликацији.';

  @override
  String get appointmentClosed => 'Овај термин више није заказан.';

  @override
  String consentTitle(String clinic) {
    return 'Дијељење ваших података са клиником $clinic';
  }

  @override
  String consentIntro(String clinic) {
    return 'Прије вашег првог заказивања код клинике $clinic потребна нам је ваша сагласност да с њом подијелимо ваше податке.';
  }

  @override
  String get whatIsShared => 'Шта се дијели';

  @override
  String get whatIsSharedBody =>
      'Ваш профил (име, датум рођења и број телефона, ако сте га навели) и ваши термини код ове клинике.';

  @override
  String get whoSeesIt => 'Ко види податке';

  @override
  String whoSeesItBody(String clinic) {
    return '$clinic. Љекар код којег закажете термин види ваш профил; администратори клинике виде ваше име и вријеме ваших термина.';
  }

  @override
  String get whatFor => 'У коју сврху';

  @override
  String get whatForBody =>
      'Само за ваше лијечење у овој клиници: за припрему и одржавање ваших термина.';

  @override
  String get whoIsResponsible => 'Ко је одговоран';

  @override
  String whoIsResponsibleBody(String clinic) {
    return 'За ваше податке одговорна је клиника $clinic. Salus их обрађује у име клинике.';
  }

  @override
  String get readPrivacyNotice => 'Прочитајте обавјештење о приватности';

  @override
  String get privacyNoticeFailed =>
      'Обавјештење о приватности није могуће отворити.';

  @override
  String get howToRevoke => 'Како повући сагласност';

  @override
  String get howToRevokeBody =>
      'Ову сагласност можете повући у било ком тренутку у дијелу Моје сагласности. Клиника тада више не види ваш профил. Ваше име и податке о терминима који су већ одржани задржава као дио своје медицинске документације. Ваши будући термини код те клинике се отказују.';

  @override
  String get agreeAndBook => 'Прихватам и заказујем';

  @override
  String get revokeTitle => 'Повући сагласност?';

  @override
  String revokeBody(String clinic) {
    return 'Клиника $clinic више неће видјети ваш профил. Ваше име и податке о терминима који су већ одржани задржава као дио своје медицинске документације. Ваши будући термини код клинике $clinic биће отказани. Престаје и дијељење ваше историје из других клиника са клиником $clinic; поновни пристанак га не враћа.';
  }

  @override
  String get keepConsent => 'Задржите сагласност';

  @override
  String get revokeConsent => 'Повуците сагласност';

  @override
  String get noConsents => 'Још нисте дали ниједну сагласност.';

  @override
  String get consentsIntro =>
      'Са овим клиникама дијелите свој профил и термине ради свог лијечења.';

  @override
  String givenOn(String date) {
    return 'Дата $date';
  }

  @override
  String get consentActive => 'Активна';

  @override
  String revokedOn(String date) {
    return 'Повучена $date';
  }

  @override
  String get consentRevoked =>
      'Ваша сагласност је повучена. Ваши будући термини код ове клинике биће отказани и за сваки ћете добити обавјештење е-поштом.';

  @override
  String get consentAlreadyRevoked => 'Ова сагласност је већ повучена.';

  @override
  String get joinConsultation => 'Укључите се у консултацију';

  @override
  String get joinFrom => 'Можете се укључити од 10 минута прије почетка.';

  @override
  String get checkTitle => 'Провјерите камеру и микрофон';

  @override
  String get checkIntro => 'Љекар ће вас видјети и чути чим се укључите.';

  @override
  String get noSound =>
      'Звук се не чује. Провјерите да ли је микрофон укључен и да није утишан.';

  @override
  String get joinCameraOff => 'Укључите се без камере';

  @override
  String get joinMicrophoneOff => 'Укључите се без микрофона';

  @override
  String get joinNow => 'Укључите се';

  @override
  String get joining => 'Укључивање…';

  @override
  String get cameraDenied =>
      'Salus нема приступ вашој камери. Приступ можете дозволити у подешавањима уређаја или се укључити без видеа.';

  @override
  String get microphoneDenied =>
      'Salus нема приступ вашем микрофону. Приступ можете дозволити у подешавањима уређаја. Без микрофона вас љекар не може чути.';

  @override
  String get openSettings => 'Отворите подешавања';

  @override
  String get noCamera => 'Камера није пронађена.';

  @override
  String get noMicrophone => 'Микрофон није пронађен.';

  @override
  String get waiting => 'Чека се љекар';

  @override
  String joinsYouHere(String name) {
    return '$name ће вам се придружити овдје. Не затварајте овај екран.';
  }

  @override
  String get mute => 'Искључите микрофон';

  @override
  String get unmute => 'Укључите микрофон';

  @override
  String get cameraOff => 'Искључите камеру';

  @override
  String get cameraOn => 'Укључите камеру';

  @override
  String get leave => 'Изађите';

  @override
  String peerCameraOff(String name) {
    return '$name: камера је искључена';
  }

  @override
  String peerMuted(String name) {
    return '$name: микрофон је искључен';
  }

  @override
  String peerLeft(String name) {
    return '$name више није у консултацији. Може се поново укључити док се консултација не заврши.';
  }

  @override
  String get reconnecting => 'Веза је прекинута. Поновно повезивање…';

  @override
  String get notRestored => 'Везу није било могуће поново успоставити.';

  @override
  String get reconnected => 'Веза је поново успостављена';

  @override
  String get consultationOver => 'Консултација је завршена';

  @override
  String get closeScreen => 'Можете затворити овај екран.';

  @override
  String get timeLimit =>
      'Предвиђено вријеме за консултацију је истекло и консултација је завршена.';

  @override
  String get practitionerAbsent => 'Љекар се није укључио';

  @override
  String get timePassed =>
      'Вријеме за укључивање у ову консултацију је истекло. Клиника може видјети да сте се ви укључили. Клиника вас може контактирати, а можете и заказати нови термин.';

  @override
  String get withdrewConsent =>
      'Повукли сте сагласност за ову клинику, па сте искључени из консултације.';

  @override
  String get appointmentWasCancelled =>
      'Овај термин је отказан, па је консултација завршена.';

  @override
  String get windowNotOpen =>
      'Ова консултација се отвара 10 минута прије почетка.';

  @override
  String get consultationEnded => 'Ова консултација је завршена.';

  @override
  String get joinNeedsConsent =>
      'За укључивање је потребна ваша сагласност за ову клинику.';

  @override
  String get consultationNotAvailable => 'Ова консултација није доступна.';

  @override
  String get cannotConnect =>
      'Видео-везу није било могуће успоставити. Провјерите интернет везу и покушајте поново.';

  @override
  String get whyCameraAndMicrophone =>
      'Да бисте се укључили у видео-консултацију, Salus треба приступ вашој камери и микрофону. Консултација се не снима ни као звук ни као видео. Ако се сагласите, оно што се каже транскрибује се и преводи.';

  @override
  String get transcriptionOn => 'Транскрипција је укључена';

  @override
  String get transcriptionTitle => 'Транскрипција и превод';

  @override
  String transcriptionIntro(String clinic) {
    return 'У вашим видео-консултацијама с клиником $clinic оно што ви и љекар кажете може да се транскрибује док говорите и да се преведе, како би вас љекар могао пратити на вашем језику.';
  }

  @override
  String get transcriptionAssistance =>
      'На основу транскрипта љекар добија и помоћ вјештачке интелигенције (AI): приједлоге током консултације и нацрте докумената. То су само приједлози; љекар их провјерава и доноси сваку одлуку.';

  @override
  String transcriptionKept(String clinic) {
    return 'Транскрипт се чува уз вашу медицинску документацију у клиници $clinic. Консултација се не снима ни као звук ни као видео.';
  }

  @override
  String get transcriptionDecline =>
      'Ако се не сагласите, консултација тече као и обично, без транскрипције, превода и помоћи вјештачке интелигенције.';

  @override
  String get transcriptionWithdraw =>
      'Сагласност можете повући у било ком тренутку, и током консултације: додирните „Транскрипција је укључена” или отворите Моје сагласности. Транскрипција престаје одмах; оно што је до тада транскрибовано остаје у вашој документацији.';

  @override
  String get transcriptionAgree => 'Прихватите и наставите';

  @override
  String get transcriptionWithout => 'Наставите без транскрипције';

  @override
  String get stopTranscriptionTitle => 'Зауставити транскрипцију?';

  @override
  String stopTranscriptionBody(String clinic) {
    return 'Транскрипција, превод и помоћ вјештачке интелигенције одмах престају у вашим консултацијама с клиником $clinic. Консултација која је у току наставља се без њих, а ваши термини остају какви јесу. Оно што је до сада транскрибовано остаје у вашој документацији.';
  }

  @override
  String get keepTranscription => 'Задржите транскрипцију';

  @override
  String get stopTranscription => 'Зауставите транскрипцију';

  @override
  String get transcriptionStopped =>
      'Транскрипција је искључена. Ваше консултације с овом клиником теку без ње.';

  @override
  String get consentForCare => 'Дијељење ради лијечења';

  @override
  String get consentForTranscription =>
      'Транскрипција и помоћ вјештачке интелигенције';

  @override
  String get toFillIn => 'За попуњавање';

  @override
  String get inProgress => 'У току';

  @override
  String get submittedStatus => 'Послат';

  @override
  String get closedStatus => 'Затворен';

  @override
  String get fillIn => 'Попуните';

  @override
  String get continueFilling => 'Наставите';

  @override
  String get viewAnswers => 'Погледајте одговоре';

  @override
  String get submitBeforeStart =>
      'Молимо вас да га пошаљете прије почетка термина.';

  @override
  String get questionnaireClosed =>
      'Термин је почео, па овај упитник више није могуће попунити.';

  @override
  String savedAsYouGo(String clinic) {
    return 'Ваши одговори се чувају док попуњавате. Клиника $clinic их види тек када их пошаљете.';
  }

  @override
  String get saved => 'Сачувано';

  @override
  String get saving => 'Чување…';

  @override
  String get notSaved =>
      'Није сачувано. Провјерите интернет везу. Чување ће бити поново покушано.';

  @override
  String get requiredQuestion => 'Обавезно';

  @override
  String get moreInformation => 'Више информација';

  @override
  String get yourAnswer => 'Ваш одговор';

  @override
  String get chooseDate => 'Изаберите датум';

  @override
  String get selectOne => 'Изаберите један одговор.';

  @override
  String get selectAll => 'Изаберите све што се односи на вас.';

  @override
  String get yes => 'Да';

  @override
  String get no => 'Не';

  @override
  String get questionRequired => 'Ово питање је обавезно.';

  @override
  String get enterWholeNumber => 'Унесите цијели број.';

  @override
  String get enterNumber => 'Унесите број.';

  @override
  String get numberTooLarge => 'Овај број је превелик.';

  @override
  String upToCharacters(int max) {
    return 'Унесите највише $max знакова.';
  }

  @override
  String get enterValidDate => 'Унесите исправан датум.';

  @override
  String get submit => 'Пошаљите';

  @override
  String get submitTitle => 'Послати одговоре?';

  @override
  String submitBody(String clinic) {
    return 'Након слања их више не можете мијењати. Видјеће их ваш љекар у клиници $clinic.';
  }

  @override
  String get cancel => 'Одустаните';

  @override
  String get requiredMissing => 'Нисте одговорили на сва обавезна питања.';

  @override
  String get answersSubmitted => 'Одговори су послати';

  @override
  String get answersSubmittedBody =>
      'Ваш љекар их може видјети прије термина. Можете их погледати у дијелу Моји термини.';

  @override
  String get yourAnswers => 'Ваши одговори';

  @override
  String submittedAt(String when) {
    return 'Послато $when';
  }

  @override
  String dateAndTime(String date, String time) {
    return '$date, $time';
  }

  @override
  String get notAnswered => 'Без одговора';

  @override
  String get leaveUnsaved =>
      'Ваш посљедњи одговор још није сачуван. Желите ли ипак да изађете?';

  @override
  String get stay => 'Останите';

  @override
  String get fillingNeedsConsent =>
      'За попуњавање овог упитника потребна је ваша сагласност за ову клинику.';

  @override
  String get pastDue =>
      'Термин је почео, па овај упитник више није могуће мијењати.';

  @override
  String get withdrawnByClinic =>
      'Клиника је повукла овај упитник. Не морате га попуњавати.';

  @override
  String get alreadySubmitted => 'Већ сте послали овај упитник.';

  @override
  String get changedElsewhere =>
      'Ови одговори су измијењени на другом уређају. Приказани су најновији одговори.';

  @override
  String get questionnaireNotAvailable => 'Овај упитник није доступан.';

  @override
  String get done => 'Готово';

  @override
  String get addFile => 'Додајте датотеку';

  @override
  String get photosUpTo => 'Фотографије до 20 MB.';

  @override
  String get documentsUpTo => 'Документи (PDF) до 20 MB.';

  @override
  String get videosUpTo => 'Видео-снимци до 200 MB.';

  @override
  String get oneFileHint => '1 датотека.';

  @override
  String upToFilesHint(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'До $n датотека.',
      few: 'До $n датотеке.',
      one: 'До $n датотеке.',
    );
    return '$_temp0';
  }

  @override
  String get takePhoto => 'Снимите фотографију';

  @override
  String get recordVideo => 'Снимите видео';

  @override
  String get chooseFromPhotos => 'Изаберите из фотографија';

  @override
  String get chooseDocument => 'Изаберите документ';

  @override
  String get preparing => 'Припрема…';

  @override
  String uploadingPercent(int n) {
    return 'Слање… $n%';
  }

  @override
  String get checkingFile => 'Провјера датотеке…';

  @override
  String get added => 'Додато';

  @override
  String get notUploaded => 'Није послато. Провјерите интернет везу.';

  @override
  String get remove => 'Уклоните';

  @override
  String get removeFile => 'Уклоните датотеку';

  @override
  String get removeFileTitle => 'Уклонити ову датотеку?';

  @override
  String get removeFileBody => 'Биће обрисана из ваших одговора.';

  @override
  String get fileRejected =>
      'Ова датотека није прошла сигурносну провјеру и обрисана је. Изаберите другу датотеку.';

  @override
  String fileTooLarge(int max) {
    return 'Ова датотека је већа од $max MB.';
  }

  @override
  String get fileTypeRefused =>
      'Ова врста датотеке не може се приложити уз ово питање.';

  @override
  String tooManyFiles(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Уз ово питање можете приложити највише $n датотека.',
      few: 'Уз ово питање можете приложити највише $n датотеке.',
      one: 'Уз ово питање можете приложити највише $n датотеку.',
    );
    return '$_temp0';
  }

  @override
  String get filesFull =>
      'Датотеке у овом упитнику достигле су укупно 500 MB. Уклоните неку датотеку да бисте додали нову.';

  @override
  String get fileNotUploaded =>
      'Датотеку није било могуће послати. Покушајте поново.';

  @override
  String get cannotPrepareFile =>
      'Ову датотеку није било могуће припремити за слање. Изаберите другу датотеку.';

  @override
  String get filesBeingChecked =>
      'Неке датотеке се још провјеравају. Упитник можете послати када буду спремне.';

  @override
  String get leaveUploading =>
      'Датотека се још шаље. Желите ли ипак да изађете? Слање ће бити прекинуто.';

  @override
  String fileCount(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n датотека',
      few: '$n датотеке',
      one: '$n датотека',
    );
    return '$_temp0';
  }

  @override
  String get photoKind => 'Фотографија';

  @override
  String get documentKind => 'Документ';

  @override
  String get videoKind => 'Видео';

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
    return 'Слање датотеке $title, $n посто';
  }

  @override
  String fileLabel(String kind, String title, String status) {
    return '$kind, $title, $status';
  }

  @override
  String get allowCamera => 'Дозволите приступ камери';

  @override
  String get whyCameraForFiles =>
      'Salus треба приступ камери да бисте снимили фотографију или видео који додајете у овај упитник.';

  @override
  String get continueLabel => 'Наставите';

  @override
  String get cameraDeniedForFiles =>
      'Salus нема приступ вашој камери. Приступ можете дозволити у подешавањима уређаја или умјесто тога изабрати датотеку.';

  @override
  String get photosDenied =>
      'Salus нема приступ вашим фотографијама. Приступ можете дозволити у подешавањима уређаја.';

  @override
  String cameraPhotoTitle(String date, String time) {
    return 'Фотографија, $date, $time';
  }

  @override
  String cameraVideoTitle(String date, String time) {
    return 'Видео, $date, $time';
  }

  @override
  String get specialty394814009 => 'Породична медицина';

  @override
  String get specialty419192003 => 'Интерна медицина';

  @override
  String get specialty394579002 => 'Кардиологија';

  @override
  String get specialty394582007 => 'Дерматологија';

  @override
  String get specialty394586005 => 'Гинекологија';

  @override
  String get specialty394537008 => 'Педијатрија';

  @override
  String get specialty394587001 => 'Психијатрија';

  @override
  String get specialty394591006 => 'Неурологија';

  @override
  String get specialty418960008 => 'Оториноларингологија (ОРЛ)';

  @override
  String get specialty394594003 => 'Офталмологија';

  @override
  String get specialty394801008 => 'Ортопедија';

  @override
  String get specialty394583002 => 'Ендокринологија';

  @override
  String questionnaireNotInLanguage(String language) {
    return 'Овај упитник није доступан на језику: $language.';
  }

  @override
  String questionnaireLanguages(String languages) {
    return 'Доступан је на овим језицима: $languages. Изаберите онај који најбоље разумијете.';
  }

  @override
  String continueInLanguage(String language) {
    return 'Наставите на језику: $language';
  }

  @override
  String get languageUnavailable =>
      'Овај језик више није доступан за овај упитник. Изаберите други.';

  @override
  String specialtyAtClinic(String specialty, String clinic) {
    return '$specialty, $clinic';
  }

  @override
  String get documents => 'Документи';

  @override
  String get noDocuments =>
      'За овај термин још нема потписаних докумената. Документ се појављује овдје када га ваш љекар потпише.';

  @override
  String documentSignedOn(String date) {
    return 'Потписано $date';
  }

  @override
  String documentLanguage(String language) {
    return 'Језик: $language';
  }

  @override
  String get sharePdf => 'Сачувај или подијели PDF';

  @override
  String get transcript => 'Транскрипт';

  @override
  String get transcriptNote =>
      'Консултација онако како је аутоматски транскрибована и преведена. Може да садржи грешке.';

  @override
  String get transcriptEmpty =>
      'У овој консултацији ништа није транскрибовано.';

  @override
  String get speakerYou => 'Ви';

  @override
  String get speakerPractitioner => 'Љекар';

  @override
  String get shareTranscript => 'Подијели транскрипт';

  @override
  String transcriptTranslation(String text) {
    return 'Пријевод: $text';
  }

  @override
  String get termExplanationNote => 'Информативно, није дијагноза';

  @override
  String get termNotReviewed => 'Још није провјерено';

  @override
  String get termClose => 'Затвори';

  @override
  String get chatsTitle => 'Поруке';

  @override
  String get chatsEmpty => 'Још немате разговора.';

  @override
  String chatUnread(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count непрочитаних порука',
      few: '$count непрочитане поруке',
      one: '$count непрочитана порука',
    );
    return '$_temp0';
  }

  @override
  String get chatDoctorUnknown => 'Ваш љекар';

  @override
  String get chatMessageHint => 'Напишите поруку';

  @override
  String get chatSend => 'Пошаљи';

  @override
  String get chatAttach => 'Додај датотеку';

  @override
  String get chatSendFailed =>
      'Порука није послата. Провјерите интернет везу и покушајте поново.';

  @override
  String get chatFileRejected => 'Ова датотека није прихваћена.';

  @override
  String get chatFileRemove => 'Уклони датотеку';

  @override
  String chatTooManyFiles(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'Већ имате $count датотека које чекају слање. Непослате датотеке се уклањају након 24 сата; покушајте поново касније.',
      few:
          'Већ имате $count датотеке које чекају слање. Непослате датотеке се уклањају након 24 сата; покушајте поново касније.',
      one:
          'Већ имате $count датотеку која чека слање. Непослате датотеке се уклањају након 24 сата; покушајте поново касније.',
    );
    return '$_temp0';
  }

  @override
  String get chatOpenFailed =>
      'Датотека није могла да се отвори. Покушајте поново.';

  @override
  String get chatEndedSession =>
      'Ваша сесија је истекла. Поново се пријавите да бисте видјели нове поруке.';

  @override
  String get chatEnded =>
      'Нове поруке у овом разговору више не могу да се прикажу.';

  @override
  String get chatHours => 'Радно вријеме за поруке';

  @override
  String chatHoursDay(String day, String start, String end) {
    return '$day: $start–$end';
  }

  @override
  String get chatHoursNone => 'Клиника није одредила радно вријеме за поруке.';

  @override
  String chatHoursZone(String zone) {
    return 'Времена су у временској зони клинике, $zone.';
  }

  @override
  String chatResponseTime(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other: 'Клиника обично одговара у року од $hours сати.',
      few: 'Клиника обично одговара у року од $hours сата.',
      one: 'Клиника обично одговара у року од $hours сата.',
    );
    return '$_temp0';
  }

  @override
  String chatOutOfHours(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other:
          'Клиника је тренутно ван радног времена за поруке. И даље можете да пишете; обично одговара у року од $hours сати.',
      few:
          'Клиника је тренутно ван радног времена за поруке. И даље можете да пишете; обично одговара у року од $hours сата.',
      one:
          'Клиника је тренутно ван радног времена за поруке. И даље можете да пишете; обично одговара у року од $hours сата.',
    );
    return '$_temp0';
  }

  @override
  String chatSystemInvited(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Љекар из друге клинике је позван у овај разговор ради другог мишљења.',
      'specialist_review': 'Љекар из друге клинике је позван у овај разговор ради специјалистичког прегледа.',
      'handover': 'Љекар из друге клинике је позван у овај разговор да преузме лијечење.',
      'other': 'Љекар из друге клинике је позван у овај разговор.',
    });
    return '$_temp0';
  }

  @override
  String chatSystemAdded(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Љекар се придружио разговору ради другог мишљења.',
      'specialist_review':
          'Љекар се придружио разговору ради специјалистичког прегледа.',
      'handover': 'Љекар се придружио разговору да преузме лијечење.',
      'other': 'Љекар се придружио разговору.',
    });
    return '$_temp0';
  }

  @override
  String get chatSystemRemoved => 'Љекар више не учествује у разговору.';

  @override
  String get chatSystemGranted =>
      'Дозволили сте љекару из друге клинике да се придружи разговору.';

  @override
  String get chatSystemDeclined => 'Одбили сте љекара из друге клинике.';

  @override
  String get chatSystemOther => 'Разговор је ажуриран.';

  @override
  String get chatInviteTitle =>
      'Љекар из друге клинике тражи приступ разговору';

  @override
  String chatInviteBody(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Ваша клиника тражи да се љекар из друге клинике придружи овом разговору ради другог мишљења.',
      'specialist_review': 'Ваша клиника тражи да се љекар из друге клинике придружи овом разговору ради специјалистичког прегледа.',
      'handover': 'Ваша клиника тражи да се љекар из друге клинике придружи овом разговору да преузме ваше лијечење.',
      'other': 'Ваша клиника тражи да се љекар из друге клинике придружи овом разговору.',
    });
    return '$_temp0';
  }

  @override
  String chatInviteWho(String doctor, String clinic) {
    return 'Љекар: $doctor, $clinic';
  }

  @override
  String get chatInviteUnresolved =>
      'Име и клиника позваног љекара нису могли да се учитају. Приступ можете да одобрите тек када буду приказани.';

  @override
  String chatInviteSees(String history) {
    String _temp0 = intl.Intl.selectLogic(history, {
      'full': 'Ако пристанете, љекар види цијели разговор, укључујући раније поруке, и може да пише у њему.',
      'from': 'Ако пристанете, љекар види разговор од поруке коју је изабрала ваша клиника надаље и може да пише у њему.',
      'other': 'Ако пристанете, љекар види разговор и може да пише у њему.',
    });
    return '$_temp0';
  }

  @override
  String get chatInviteChoice =>
      'Можете да одбијете. Ако пристанете, приступ љекара можете касније да опозовете у овом разговору; престаје одмах.';

  @override
  String get chatGrant => 'Одобри приступ';

  @override
  String get chatDecline => 'Одбиј';

  @override
  String get chatGranted =>
      'Приступ је одобрен. Можете да га опозовете у овом разговору у било ком тренутку.';

  @override
  String get chatDeclined => 'Одбијено. Љекар се не придружује овом разговору.';

  @override
  String get chatRevoke => 'Опозови приступ љекара из друге клинике';

  @override
  String chatGrantedTo(String doctor, String clinic) {
    return 'Приступ је одобрен: $doctor, $clinic.';
  }

  @override
  String get chatGrantedToUnknown =>
      'Приступ је одобрен љекару из друге клинике.';

  @override
  String get chatRevokeBody =>
      'Љекар из друге клинике одмах губи приступ овом разговору. Већ написане поруке остају у њему.';

  @override
  String get chatRevokeConfirm => 'Опозови приступ';

  @override
  String get chatRevoked => 'Приступ је опозван.';

  @override
  String get allergies => 'Алергије';

  @override
  String get currentMedications => 'Тренутни лијекови';

  @override
  String get allergiesIntro =>
      'Клинике којима сте дали сагласност виде алергије из вашег профила када имате термин код њих.';

  @override
  String get medicationsIntro =>
      'Клинике којима сте дали сагласност виде лијекове из вашег профила када имате термин код њих.';

  @override
  String get noAllergies => 'Нисте додали ниједну алергију.';

  @override
  String get noMedications => 'Нисте додали ниједан лијек.';

  @override
  String get addAllergy => 'Додајте алергију';

  @override
  String get addMedication => 'Додајте лијек';

  @override
  String get changeAllergy => 'Измјена алергије';

  @override
  String get changeMedication => 'Измјена лијека';

  @override
  String get change => 'Измијените';

  @override
  String get allergySubstance => 'Супстанца или намирница';

  @override
  String get medicationName => 'Лијек';

  @override
  String get termSearchHint =>
      'Почните да куцате за претрагу или упишите својим ријечима.';

  @override
  String get allergyCategory => 'Врста (није обавезно)';

  @override
  String get notSpecified => 'Није наведено';

  @override
  String get categoryFood => 'Храна';

  @override
  String get categoryMedication => 'Лијек';

  @override
  String get categoryEnvironment => 'Окружење';

  @override
  String get categoryBiologic => 'Биолошки препарат';

  @override
  String get reactionOptional => 'Реакција (није обавезно)';

  @override
  String get severity => 'Тежина';

  @override
  String get severityMild => 'Блага';

  @override
  String get severityModerate => 'Умјерена';

  @override
  String get severitySevere => 'Тешка';

  @override
  String get entryStatus => 'Статус';

  @override
  String get allergyActive => 'Активна';

  @override
  String get allergyInactive => 'Неактивна';

  @override
  String get allergyResolved => 'Прошла';

  @override
  String reactionLine(String reaction) {
    return 'Реакција: $reaction';
  }

  @override
  String reactionWithSeverity(String reaction, String severity) {
    return 'Реакција: $reaction ($severity)';
  }

  @override
  String get confirmedByClinician => 'Потврдио љекар';

  @override
  String get recordedByClinic => 'Унијела клиника током једне од ваших посјета';

  @override
  String get confirmedChangeNote =>
      'Ако измијените овај унос, више неће бити потврђен од љекара.';

  @override
  String get nameRequired => 'Унесите назив, највише 200 знакова.';

  @override
  String get addToProfile => 'Додајте у мој профил';

  @override
  String get addToProfileTitle => 'Додати у ваш профил?';

  @override
  String get addToProfileBody =>
      'Тада ће га видјети свака клиника којој сте дали сагласност, без назива клинике која га је унијела.';

  @override
  String get addedToProfile => 'Сада је у вашем профилу.';

  @override
  String get removeEntryTitle => 'Уклонити овај унос?';

  @override
  String get removeEntryBody =>
      'Више се неће приказивати ни вама ни било којој клиници. Ако га је клиника унијела или потврдила, остаје у вашем медицинском картону, означен као уклоњен.';

  @override
  String get entryRemoved => 'Унос је уклоњен.';

  @override
  String get entrySaved => 'Сачувано у вашем профилу.';

  @override
  String get entryChanged =>
      'Овај унос је у међувремену измијењен. Приказана је најновија верзија.';

  @override
  String get entryInvalid =>
      'Неки подаци нису исправни. Провјерите их и покушајте поново.';

  @override
  String get entryNotAllowed => 'Не можете измијенити овај унос.';

  @override
  String get entryNotFound => 'Овај унос више није доступан.';

  @override
  String get doseHeading => 'Доза (није обавезно)';

  @override
  String get doseAmount => 'Количина';

  @override
  String get doseUnit => 'Јединица';

  @override
  String get unitMg => 'милиграми (мг)';

  @override
  String get unitG => 'грами (г)';

  @override
  String get unitUg => 'микрограми (µг)';

  @override
  String get unitMl => 'милилитри (мл)';

  @override
  String get unitIu => 'интернационалне јединице (ИЈ)';

  @override
  String get unitTablet => 'таблете';

  @override
  String get unitCapsule => 'капсуле';

  @override
  String get unitDrop => 'капи';

  @override
  String get unitPuff => 'удисаји (инхалатор)';

  @override
  String get unitPatch => 'фластери';

  @override
  String get doseBoth => 'Унесите и количину и јединицу или ништа од тога.';

  @override
  String get numberAboveZero => 'Унесите број већи од 0.';

  @override
  String get howOftenHeading => 'Колико често (није обавезно)';

  @override
  String get timesLabel => 'Пута';

  @override
  String get everyLabel => 'Сваких';

  @override
  String get periodUnitLabel => 'Јединица времена';

  @override
  String get periodHours => 'сати';

  @override
  String get periodDays => 'дана';

  @override
  String get periodWeeks => 'недјеља';

  @override
  String get periodMonths => 'мјесеци';

  @override
  String get howOftenAll => 'Попуните сва три поља или их оставите празна.';

  @override
  String get asNeeded => 'По потреби';

  @override
  String get medicationActive => 'Тренутно узимам';

  @override
  String get medicationStopped => 'Више не узимам';

  @override
  String doseMg(String value) {
    return '$value мг';
  }

  @override
  String doseG(String value) {
    return '$value г';
  }

  @override
  String doseUg(String value) {
    return '$value µг';
  }

  @override
  String doseMl(String value) {
    return '$value мл';
  }

  @override
  String doseIu(String value) {
    return '$value ИЈ';
  }

  @override
  String doseTablet(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value таблета',
      few: '$value таблете',
      one: '$value таблета',
    );
    return '$_temp0';
  }

  @override
  String doseCapsule(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value капсула',
      few: '$value капсуле',
      one: '$value капсула',
    );
    return '$_temp0';
  }

  @override
  String doseDrop(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value капи',
      few: '$value капи',
      one: '$value кап',
    );
    return '$_temp0';
  }

  @override
  String dosePuff(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value удисаја',
      few: '$value удисаја',
      one: '$value удисај',
    );
    return '$_temp0';
  }

  @override
  String dosePatch(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value фластера',
      few: '$value фластера',
      one: '$value фластер',
    );
    return '$_temp0';
  }

  @override
  String doseLine(String dose) {
    return 'Доза: $dose';
  }

  @override
  String howOftenPerHour(int frequency) {
    return '$frequency× на сат';
  }

  @override
  String howOftenPerDay(int frequency) {
    return '$frequency× дневно';
  }

  @override
  String howOftenPerWeek(int frequency) {
    return '$frequency× недјељно';
  }

  @override
  String howOftenPerMonth(int frequency) {
    return '$frequency× мјесечно';
  }

  @override
  String howOftenEveryHours(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× сваких $every сати',
      few: '$frequency× свака $every сата',
      one: '$frequency× сваки $every сат',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryDays(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× сваких $every дана',
      few: '$frequency× свака $every дана',
      one: '$frequency× сваки $every дан',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryWeeks(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× сваких $every недјеља',
      few: '$frequency× сваке $every недјеље',
      one: '$frequency× сваке $every недјеље',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryMonths(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× сваких $every мјесеци',
      few: '$frequency× свака $every мјесеца',
      one: '$frequency× сваки $every мјесец',
    );
    return '$_temp0';
  }

  @override
  String get myHistory => 'Моја историја';

  @override
  String get historyEmpty =>
      'Ваша историја је још празна. Овдје се појављују ваше посјете, документи и одговори.';

  @override
  String get historyNoMatch =>
      'Ништа у вашој историји не одговара овим филтерима.';

  @override
  String get historyClinic => 'Клиника';

  @override
  String get historyAllClinics => 'Све клинике';

  @override
  String get historyVisits => 'Посјете';

  @override
  String get historyVisit => 'Посјета';

  @override
  String get historyAnswers => 'Одговори на упитник';

  @override
  String get historyFiles => 'Датотеке';

  @override
  String get historyFile => 'Датотека';

  @override
  String get historyAnyDate => 'Било који датум';

  @override
  String historyDateRange(String from, String to) {
    return 'од $from до $to';
  }

  @override
  String get historyClearDates => 'Поништи датуме';

  @override
  String get historyShowMore => 'Прикажи више';

  @override
  String get historyMoreFailed =>
      'Остатак ваше историје није се могао учитати.';

  @override
  String get historyOpenFailed =>
      'Ова ставка се није могла отворити. Покушајте поново.';

  @override
  String get sharingToggle =>
      'Подијели моју историју из других клиника са овом клиником';

  @override
  String sharingIntro(String clinic) {
    return 'Означите раније посјете које $clinic смије да види. За сваку означену посјету љекари клинике $clinic код којих имате термин виде оно што је овдје приказано, са документима посјете, одговорима на упитнике и датотекама, алергијама и лијековима евидентираним у њој, као и свиме што се посјети касније дода. Посјете које не означите остају скривене од $clinic.';
  }

  @override
  String get sharingChangeLater =>
      'Посјете можете да додате или уклоните, или да престанете да дијелите, у било ком тренутку у дијелу Подијељена историја. Свака промјена важи одмах.';

  @override
  String get sharingNoVisits =>
      'Немате ранијих посјета у другим клиникама које бисте могли да подијелите.';

  @override
  String get sharingSelectAll => 'Означите све';

  @override
  String get sharingSelectNone => 'Поништите све';

  @override
  String get sharedHistory => 'Подијељена историја';

  @override
  String get sharedHistoryIntro =>
      'Клинике којима сте дали сагласност и колико ваших посјета из других клиника свака од њих види.';

  @override
  String get sharedHistoryNone => 'Још нисте дали сагласност ниједној клиници.';

  @override
  String sharedVisits(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count подијељених посјета',
      few: '$count подијељене посјете',
      one: '$count подијељена посјета',
      zero: 'Ништа није подијељено',
    );
    return '$_temp0';
  }

  @override
  String get sharingSaved => 'Ваш избор је сачуван и важи одмах.';

  @override
  String get sharingStop => 'Престаните да дијелите';

  @override
  String get sharingStopTitle => 'Престати са дијељењем ваше историје?';

  @override
  String sharingStopBody(String clinic) {
    return '$clinic више неће видјети ниједну посјету из других клиника коју сте са њом подијелили. Ово важи одмах.';
  }

  @override
  String get sharingKeep => 'Наставите да дијелите';

  @override
  String get sharingStopped =>
      'Више не дијелите своју историју са овом клиником.';

  @override
  String get sharingChanged =>
      'Ваше подијељене посјете су се у међувремену промијениле. Провјерите их и поново сачувајте.';

  @override
  String get bookedNotShared =>
      'Ваш термин је заказан, али ваша историја није могла да буде подијељена. Можете да је подијелите у дијелу Подијељена историја.';

  @override
  String get consentForHistory => 'Посјете подијељене из других клиника';

  @override
  String get waitlistTitle => 'Моја листа чекања';

  @override
  String get waitlistJoinButton =>
      'Нема одговарајућег термина? Пријавите се на листу чекања';

  @override
  String get waitlistJoinTitle => 'Пријава на листу чекања';

  @override
  String get waitlistExplained =>
      'Када се ослободи термин код овог љекара, нуди се пацијентима на чекању, једном по једном. Добићете обавјештење и термин можете да закажете у апликацији док је резервисан за вас.';

  @override
  String get waitlistHowLong => 'Колико дуго?';

  @override
  String waitlistDays(int days) {
    String _temp0 = intl.Intl.pluralLogic(
      days,
      locale: localeName,
      other: '$days дана',
      few: '$days дана',
      one: '$days дан',
    );
    return '$_temp0';
  }

  @override
  String get waitlistWhen => 'У које доба дана?';

  @override
  String get waitlistAnyTime => 'Било када';

  @override
  String get waitlistMornings => 'Ујутру, од 08:00 до 12:00';

  @override
  String get waitlistAfternoons => 'Послијеподне, од 12:00 до 17:00';

  @override
  String get waitlistJoin => 'Пријави се на листу чекања';

  @override
  String get waitlistJoined => 'На листи чекања сте.';

  @override
  String get waitlistEmpty => 'Нисте ни на једној листи чекања.';

  @override
  String get waitlistAPractitioner => 'Љекар';

  @override
  String waitlistUntil(String date) {
    return 'Чекате до $date';
  }

  @override
  String get waitlistOfferTitle => 'Ослободио се термин за вас';

  @override
  String waitlistHeldUntil(String time) {
    return 'Резервисан за вас до $time';
  }

  @override
  String get waitlistAcceptEffect =>
      'Прихватањем заказујете овај термин. Када плаћање буде доступно у апликацији Salus, прихватањем ћете га и платити.';

  @override
  String get waitlistAccept => 'Прихвати и закажи';

  @override
  String get waitlistDecline => 'Одбиј';

  @override
  String get waitlistBooked =>
      'Ваш термин је заказан. Наћи ћете га у одјељку Моји термини.';

  @override
  String get waitlistDeclined => 'Одбили сте термин. Остајете на листи чекања.';

  @override
  String get waitlistWithdraw => 'Напусти листу чекања';

  @override
  String get waitlistWithdrawn => 'Напустили сте листу чекања.';

  @override
  String get waitlistOff => 'Ова установа нема листу чекања.';

  @override
  String get waitlistTooMany =>
      'Већ сте пет пута на листи чекања ове установе. Прво напустите једну од пријава.';

  @override
  String get waitlistOfferExpired =>
      'Овај термин више није резервисан за вас. Остајете на листи чекања.';
}

/// The translations for Serbian, using the Latin script (`sr_Latn`).
class AppLocalizationsSrLatn extends AppLocalizationsSr {
  AppLocalizationsSrLatn() : super('sr_Latn');

  @override
  String get tryAgain => 'Pokušajte ponovo';

  @override
  String get language => 'Jezik';

  @override
  String get useDeviceLanguage => 'Koristi jezik uređaja';

  @override
  String get languageSaved => 'Jezik je sačuvan.';

  @override
  String get somethingWrong => 'Došlo je do greške. Pokušajte ponovo.';

  @override
  String get unreachable =>
      'Salus trenutno nije dostupan. Provjerite internet vezu i pokušajte ponovo.';

  @override
  String get wrongCredentials => 'Adresa e-pošte ili lozinka nije tačna.';

  @override
  String get wrongCode =>
      'Kod nije tačan. Provjerite vrijeme na uređaju i pokušajte ponovo.';

  @override
  String get challengeEnded =>
      'Prijava je istekla. Ponovo unesite adresu e-pošte i lozinku.';

  @override
  String get emailTaken =>
      'Za ovu adresu e-pošte već postoji nalog. Umjesto toga se prijavite.';

  @override
  String get invalidEmail => 'Unesite ispravnu adresu e-pošte.';

  @override
  String get invalidPassword => 'Lozinka mora imati od 12 do 128 znakova.';

  @override
  String get missingPassword => 'Unesite lozinku.';

  @override
  String get invalidCode => 'Kod ima 6 cifara.';

  @override
  String get staffOnly =>
      'Ova aplikacija je namijenjena pacijentima. Osoblje klinike koristi veb-aplikaciju Salus.';

  @override
  String get accountReady =>
      'Vaš nalog je spreman. Prijavite se da biste nastavili.';

  @override
  String get signOutFailed =>
      'Odjava nije uspjela. Provjerite internet vezu i pokušajte ponovo.';

  @override
  String get tooManyAttemptsLater =>
      'Previše pokušaja. Pokušajte ponovo kasnije.';

  @override
  String tooManyAttempts(int minutes) {
    String _temp0 = intl.Intl.pluralLogic(
      minutes,
      locale: localeName,
      other: 'Previše pokušaja. Pokušajte ponovo za $minutes minuta.',
      few: 'Previše pokušaja. Pokušajte ponovo za $minutes minuta.',
      one: 'Previše pokušaja. Pokušajte ponovo za $minutes minut.',
    );
    return '$_temp0';
  }

  @override
  String get signInTitle => 'Prijava u Salus';

  @override
  String get email => 'E-pošta';

  @override
  String get password => 'Lozinka';

  @override
  String get signIn => 'Prijavite se';

  @override
  String get createAnAccount => 'Napravite nalog';

  @override
  String get codePrompt =>
      'Unesite šestocifreni kod iz aplikacije za autentifikaciju.';

  @override
  String get code => 'Kod';

  @override
  String get useAnotherAccount => 'Prijavite se drugim nalogom';

  @override
  String get passwordHint => 'Od 12 do 128 znakova';

  @override
  String get createAccount => 'Napravite nalog';

  @override
  String get haveAccount => 'Već imam nalog';

  @override
  String signedInAs(String email) {
    return 'Prijavljeni ste kao $email';
  }

  @override
  String get completeProfile => 'Popunite svoj profil';

  @override
  String get completeProfileFirst => 'Prije zakazivanja popunite svoj profil.';

  @override
  String get bookAppointment => 'Zakažite termin';

  @override
  String get myAppointments => 'Moji termini';

  @override
  String get myConsents => 'Moje saglasnosti';

  @override
  String get myProfile => 'Moj profil';

  @override
  String get signOut => 'Odjavite se';

  @override
  String get profileIntro =>
      'Klinike kod kojih zakažete termin vide ove podatke. Broj telefona nije obavezan; ljekar ga koristi samo da bi vas kontaktirao u vezi s terminom.';

  @override
  String get givenName => 'Ime';

  @override
  String get familyName => 'Prezime';

  @override
  String get dateOfBirth => 'Datum rođenja';

  @override
  String get dateFormatHint => 'GGGG-MM-DD';

  @override
  String get phoneOptional => 'Telefon (nije obavezno)';

  @override
  String phoneHint(String example) {
    return 'Sa pozivnim brojem zemlje, na primjer $example';
  }

  @override
  String get save => 'Sačuvajte';

  @override
  String get notNow => 'Ne sada';

  @override
  String get invalidGivenName => 'Unesite ime, najviše 100 znakova.';

  @override
  String get invalidFamilyName => 'Unesite prezime, najviše 100 znakova.';

  @override
  String get invalidName => 'Unesite ime i prezime, svako najviše 100 znakova.';

  @override
  String get birthDateFormat => 'Unesite datum rođenja u obliku GGGG-MM-DD.';

  @override
  String get birthDateRange =>
      'Unesite datum rođenja u prošlosti, od 1900. godine nadalje.';

  @override
  String invalidPhone(String example) {
    return 'Unesite broj telefona sa pozivnim brojem zemlje, na primjer $example.';
  }

  @override
  String get profileChanged =>
      'Vaš profil je u međuvremenu izmijenjen. Provjerite ga i ponovo ga sačuvajte.';

  @override
  String get chooseClinic => 'Izaberite kliniku';

  @override
  String get noClinics => 'Još nema klinika na spisku.';

  @override
  String get noPractitioners => 'Ova klinika još nema ljekara na spisku.';

  @override
  String get choosePractitioner => 'Izaberite ljekara.';

  @override
  String get noOpenTimes => 'Ovaj ljekar još nema objavljenih termina.';

  @override
  String timesIn(String zone) {
    return 'Vrijeme je prikazano u vremenskoj zoni $zone.';
  }

  @override
  String get previousDay => 'Prethodni dan';

  @override
  String get nextDay => 'Sljedeći dan';

  @override
  String get noFreeTimes => 'Ovog dana nema slobodnih termina.';

  @override
  String get bookThisTime => 'Zakažite ovaj termin';

  @override
  String get chooseAnotherTime => 'Izaberite drugi termin';

  @override
  String get book => 'Zakažite';

  @override
  String timeRange(String start, String end, String zone) {
    return '$start do $end ($zone)';
  }

  @override
  String get slotTaken =>
      'Ovaj termin više nije slobodan. Izaberite drugi termin.';

  @override
  String get consentNeeded =>
      'Za zakazivanje kod ove klinike potrebna je vaša saglasnost.';

  @override
  String get appointmentBooked => 'Vaš termin je zakazan.';

  @override
  String get unlistedClinic => 'Klinika koja više nije na spisku';

  @override
  String get cancelTitle => 'Otkazati ovaj termin?';

  @override
  String cancelBody(String date, String time) {
    return 'Vaš termin zakazan za $date u $time biće otkazan. Potvrdu otkazivanja ćete dobiti e-poštom.';
  }

  @override
  String get keepIt => 'Zadržite termin';

  @override
  String get cancelAppointment => 'Otkažite termin';

  @override
  String get noAppointments => 'Još nemate zakazanih termina.';

  @override
  String get upcoming => 'Predstojeći';

  @override
  String get past => 'Prošli';

  @override
  String get statusBooked => 'Zakazan';

  @override
  String get statusCancelled => 'Otkazan';

  @override
  String get statusAttended => 'Obavljen';

  @override
  String get statusNotAttended => 'Propušten';

  @override
  String get appointmentCancelled => 'Vaš termin je otkazan.';

  @override
  String get cancelWindowHint =>
      'Termin možete otkazati u aplikaciji najkasnije 24 sata prije početka.';

  @override
  String get cancelWindowClosed =>
      'Ovaj termin počinje za manje od 24 sata i više ga nije moguće otkazati u aplikaciji.';

  @override
  String get appointmentClosed => 'Ovaj termin više nije zakazan.';

  @override
  String consentTitle(String clinic) {
    return 'Dijeljenje vaših podataka sa klinikom $clinic';
  }

  @override
  String consentIntro(String clinic) {
    return 'Prije vašeg prvog zakazivanja kod klinike $clinic potrebna nam je vaša saglasnost da s njom podijelimo vaše podatke.';
  }

  @override
  String get whatIsShared => 'Šta se dijeli';

  @override
  String get whatIsSharedBody =>
      'Vaš profil (ime, datum rođenja i broj telefona, ako ste ga naveli) i vaši termini kod ove klinike.';

  @override
  String get whoSeesIt => 'Ko vidi podatke';

  @override
  String whoSeesItBody(String clinic) {
    return '$clinic. Ljekar kod kojeg zakažete termin vidi vaš profil; administratori klinike vide vaše ime i vrijeme vaših termina.';
  }

  @override
  String get whatFor => 'U koju svrhu';

  @override
  String get whatForBody =>
      'Samo za vaše liječenje u ovoj klinici: za pripremu i održavanje vaših termina.';

  @override
  String get whoIsResponsible => 'Ko je odgovoran';

  @override
  String whoIsResponsibleBody(String clinic) {
    return 'Za vaše podatke odgovorna je klinika $clinic. Salus ih obrađuje u ime klinike.';
  }

  @override
  String get readPrivacyNotice => 'Pročitajte obavještenje o privatnosti';

  @override
  String get privacyNoticeFailed =>
      'Obavještenje o privatnosti nije moguće otvoriti.';

  @override
  String get howToRevoke => 'Kako povući saglasnost';

  @override
  String get howToRevokeBody =>
      'Ovu saglasnost možete povući u bilo kom trenutku u dijelu Moje saglasnosti. Klinika tada više ne vidi vaš profil. Vaše ime i podatke o terminima koji su već održani zadržava kao dio svoje medicinske dokumentacije. Vaši budući termini kod te klinike se otkazuju.';

  @override
  String get agreeAndBook => 'Prihvatam i zakazujem';

  @override
  String get revokeTitle => 'Povući saglasnost?';

  @override
  String revokeBody(String clinic) {
    return 'Klinika $clinic više neće vidjeti vaš profil. Vaše ime i podatke o terminima koji su već održani zadržava kao dio svoje medicinske dokumentacije. Vaši budući termini kod klinike $clinic biće otkazani. Prestaje i dijeljenje vaše istorije iz drugih klinika sa klinikom $clinic; ponovni pristanak ga ne vraća.';
  }

  @override
  String get keepConsent => 'Zadržite saglasnost';

  @override
  String get revokeConsent => 'Povucite saglasnost';

  @override
  String get noConsents => 'Još niste dali nijednu saglasnost.';

  @override
  String get consentsIntro =>
      'Sa ovim klinikama dijelite svoj profil i termine radi svog liječenja.';

  @override
  String givenOn(String date) {
    return 'Data $date';
  }

  @override
  String get consentActive => 'Aktivna';

  @override
  String revokedOn(String date) {
    return 'Povučena $date';
  }

  @override
  String get consentRevoked =>
      'Vaša saglasnost je povučena. Vaši budući termini kod ove klinike biće otkazani i za svaki ćete dobiti obavještenje e-poštom.';

  @override
  String get consentAlreadyRevoked => 'Ova saglasnost je već povučena.';

  @override
  String get joinConsultation => 'Uključite se u konsultaciju';

  @override
  String get joinFrom => 'Možete se uključiti od 10 minuta prije početka.';

  @override
  String get checkTitle => 'Provjerite kameru i mikrofon';

  @override
  String get checkIntro => 'Ljekar će vas vidjeti i čuti čim se uključite.';

  @override
  String get noSound =>
      'Zvuk se ne čuje. Provjerite da li je mikrofon uključen i da nije utišan.';

  @override
  String get joinCameraOff => 'Uključite se bez kamere';

  @override
  String get joinMicrophoneOff => 'Uključite se bez mikrofona';

  @override
  String get joinNow => 'Uključite se';

  @override
  String get joining => 'Uključivanje…';

  @override
  String get cameraDenied =>
      'Salus nema pristup vašoj kameri. Pristup možete dozvoliti u podešavanjima uređaja ili se uključiti bez videa.';

  @override
  String get microphoneDenied =>
      'Salus nema pristup vašem mikrofonu. Pristup možete dozvoliti u podešavanjima uređaja. Bez mikrofona vas ljekar ne može čuti.';

  @override
  String get openSettings => 'Otvorite podešavanja';

  @override
  String get noCamera => 'Kamera nije pronađena.';

  @override
  String get noMicrophone => 'Mikrofon nije pronađen.';

  @override
  String get waiting => 'Čeka se ljekar';

  @override
  String joinsYouHere(String name) {
    return '$name će vam se pridružiti ovdje. Ne zatvarajte ovaj ekran.';
  }

  @override
  String get mute => 'Isključite mikrofon';

  @override
  String get unmute => 'Uključite mikrofon';

  @override
  String get cameraOff => 'Isključite kameru';

  @override
  String get cameraOn => 'Uključite kameru';

  @override
  String get leave => 'Izađite';

  @override
  String peerCameraOff(String name) {
    return '$name: kamera je isključena';
  }

  @override
  String peerMuted(String name) {
    return '$name: mikrofon je isključen';
  }

  @override
  String peerLeft(String name) {
    return '$name više nije u konsultaciji. Može se ponovo uključiti dok se konsultacija ne završi.';
  }

  @override
  String get reconnecting => 'Veza je prekinuta. Ponovno povezivanje…';

  @override
  String get notRestored => 'Vezu nije bilo moguće ponovo uspostaviti.';

  @override
  String get reconnected => 'Veza je ponovo uspostavljena';

  @override
  String get consultationOver => 'Konsultacija je završena';

  @override
  String get closeScreen => 'Možete zatvoriti ovaj ekran.';

  @override
  String get timeLimit =>
      'Predviđeno vrijeme za konsultaciju je isteklo i konsultacija je završena.';

  @override
  String get practitionerAbsent => 'Ljekar se nije uključio';

  @override
  String get timePassed =>
      'Vrijeme za uključivanje u ovu konsultaciju je isteklo. Klinika može vidjeti da ste se vi uključili. Klinika vas može kontaktirati, a možete i zakazati novi termin.';

  @override
  String get withdrewConsent =>
      'Povukli ste saglasnost za ovu kliniku, pa ste isključeni iz konsultacije.';

  @override
  String get appointmentWasCancelled =>
      'Ovaj termin je otkazan, pa je konsultacija završena.';

  @override
  String get windowNotOpen =>
      'Ova konsultacija se otvara 10 minuta prije početka.';

  @override
  String get consultationEnded => 'Ova konsultacija je završena.';

  @override
  String get joinNeedsConsent =>
      'Za uključivanje je potrebna vaša saglasnost za ovu kliniku.';

  @override
  String get consultationNotAvailable => 'Ova konsultacija nije dostupna.';

  @override
  String get cannotConnect =>
      'Video-vezu nije bilo moguće uspostaviti. Provjerite internet vezu i pokušajte ponovo.';

  @override
  String get whyCameraAndMicrophone =>
      'Da biste se uključili u video-konsultaciju, Salus treba pristup vašoj kameri i mikrofonu. Konsultacija se ne snima ni kao zvuk ni kao video. Ako se saglasite, ono što se kaže transkribuje se i prevodi.';

  @override
  String get transcriptionOn => 'Transkripcija je uključena';

  @override
  String get transcriptionTitle => 'Transkripcija i prevod';

  @override
  String transcriptionIntro(String clinic) {
    return 'U vašim video-konsultacijama s klinikom $clinic ono što vi i ljekar kažete može da se transkribuje dok govorite i da se prevede, kako bi vas ljekar mogao pratiti na vašem jeziku.';
  }

  @override
  String get transcriptionAssistance =>
      'Na osnovu transkripta ljekar dobija i pomoć vještačke inteligencije (AI): prijedloge tokom konsultacije i nacrte dokumenata. To su samo prijedlozi; ljekar ih provjerava i donosi svaku odluku.';

  @override
  String transcriptionKept(String clinic) {
    return 'Transkript se čuva uz vašu medicinsku dokumentaciju u klinici $clinic. Konsultacija se ne snima ni kao zvuk ni kao video.';
  }

  @override
  String get transcriptionDecline =>
      'Ako se ne saglasite, konsultacija teče kao i obično, bez transkripcije, prevoda i pomoći vještačke inteligencije.';

  @override
  String get transcriptionWithdraw =>
      'Saglasnost možete povući u bilo kom trenutku, i tokom konsultacije: dodirnite „Transkripcija je uključena” ili otvorite Moje saglasnosti. Transkripcija prestaje odmah; ono što je do tada transkribovano ostaje u vašoj dokumentaciji.';

  @override
  String get transcriptionAgree => 'Prihvatite i nastavite';

  @override
  String get transcriptionWithout => 'Nastavite bez transkripcije';

  @override
  String get stopTranscriptionTitle => 'Zaustaviti transkripciju?';

  @override
  String stopTranscriptionBody(String clinic) {
    return 'Transkripcija, prevod i pomoć vještačke inteligencije odmah prestaju u vašim konsultacijama s klinikom $clinic. Konsultacija koja je u toku nastavlja se bez njih, a vaši termini ostaju kakvi jesu. Ono što je do sada transkribovano ostaje u vašoj dokumentaciji.';
  }

  @override
  String get keepTranscription => 'Zadržite transkripciju';

  @override
  String get stopTranscription => 'Zaustavite transkripciju';

  @override
  String get transcriptionStopped =>
      'Transkripcija je isključena. Vaše konsultacije s ovom klinikom teku bez nje.';

  @override
  String get consentForCare => 'Dijeljenje radi liječenja';

  @override
  String get consentForTranscription =>
      'Transkripcija i pomoć vještačke inteligencije';

  @override
  String get toFillIn => 'Za popunjavanje';

  @override
  String get inProgress => 'U toku';

  @override
  String get submittedStatus => 'Poslat';

  @override
  String get closedStatus => 'Zatvoren';

  @override
  String get fillIn => 'Popunite';

  @override
  String get continueFilling => 'Nastavite';

  @override
  String get viewAnswers => 'Pogledajte odgovore';

  @override
  String get submitBeforeStart =>
      'Molimo vas da ga pošaljete prije početka termina.';

  @override
  String get questionnaireClosed =>
      'Termin je počeo, pa ovaj upitnik više nije moguće popuniti.';

  @override
  String savedAsYouGo(String clinic) {
    return 'Vaši odgovori se čuvaju dok popunjavate. Klinika $clinic ih vidi tek kada ih pošaljete.';
  }

  @override
  String get saved => 'Sačuvano';

  @override
  String get saving => 'Čuvanje…';

  @override
  String get notSaved =>
      'Nije sačuvano. Provjerite internet vezu. Čuvanje će biti ponovo pokušano.';

  @override
  String get requiredQuestion => 'Obavezno';

  @override
  String get moreInformation => 'Više informacija';

  @override
  String get yourAnswer => 'Vaš odgovor';

  @override
  String get chooseDate => 'Izaberite datum';

  @override
  String get selectOne => 'Izaberite jedan odgovor.';

  @override
  String get selectAll => 'Izaberite sve što se odnosi na vas.';

  @override
  String get yes => 'Da';

  @override
  String get no => 'Ne';

  @override
  String get questionRequired => 'Ovo pitanje je obavezno.';

  @override
  String get enterWholeNumber => 'Unesite cijeli broj.';

  @override
  String get enterNumber => 'Unesite broj.';

  @override
  String get numberTooLarge => 'Ovaj broj je prevelik.';

  @override
  String upToCharacters(int max) {
    return 'Unesite najviše $max znakova.';
  }

  @override
  String get enterValidDate => 'Unesite ispravan datum.';

  @override
  String get submit => 'Pošaljite';

  @override
  String get submitTitle => 'Poslati odgovore?';

  @override
  String submitBody(String clinic) {
    return 'Nakon slanja ih više ne možete mijenjati. Vidjeće ih vaš ljekar u klinici $clinic.';
  }

  @override
  String get cancel => 'Odustanite';

  @override
  String get requiredMissing => 'Niste odgovorili na sva obavezna pitanja.';

  @override
  String get answersSubmitted => 'Odgovori su poslati';

  @override
  String get answersSubmittedBody =>
      'Vaš ljekar ih može vidjeti prije termina. Možete ih pogledati u dijelu Moji termini.';

  @override
  String get yourAnswers => 'Vaši odgovori';

  @override
  String submittedAt(String when) {
    return 'Poslato $when';
  }

  @override
  String dateAndTime(String date, String time) {
    return '$date, $time';
  }

  @override
  String get notAnswered => 'Bez odgovora';

  @override
  String get leaveUnsaved =>
      'Vaš posljednji odgovor još nije sačuvan. Želite li ipak da izađete?';

  @override
  String get stay => 'Ostanite';

  @override
  String get fillingNeedsConsent =>
      'Za popunjavanje ovog upitnika potrebna je vaša saglasnost za ovu kliniku.';

  @override
  String get pastDue =>
      'Termin je počeo, pa ovaj upitnik više nije moguće mijenjati.';

  @override
  String get withdrawnByClinic =>
      'Klinika je povukla ovaj upitnik. Ne morate ga popunjavati.';

  @override
  String get alreadySubmitted => 'Već ste poslali ovaj upitnik.';

  @override
  String get changedElsewhere =>
      'Ovi odgovori su izmijenjeni na drugom uređaju. Prikazani su najnoviji odgovori.';

  @override
  String get questionnaireNotAvailable => 'Ovaj upitnik nije dostupan.';

  @override
  String get done => 'Gotovo';

  @override
  String get addFile => 'Dodajte datoteku';

  @override
  String get photosUpTo => 'Fotografije do 20 MB.';

  @override
  String get documentsUpTo => 'Dokumenti (PDF) do 20 MB.';

  @override
  String get videosUpTo => 'Video-snimci do 200 MB.';

  @override
  String get oneFileHint => '1 datoteka.';

  @override
  String upToFilesHint(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Do $n datoteka.',
      few: 'Do $n datoteke.',
      one: 'Do $n datoteke.',
    );
    return '$_temp0';
  }

  @override
  String get takePhoto => 'Snimite fotografiju';

  @override
  String get recordVideo => 'Snimite video';

  @override
  String get chooseFromPhotos => 'Izaberite iz fotografija';

  @override
  String get chooseDocument => 'Izaberite dokument';

  @override
  String get preparing => 'Priprema…';

  @override
  String uploadingPercent(int n) {
    return 'Slanje… $n%';
  }

  @override
  String get checkingFile => 'Provjera datoteke…';

  @override
  String get added => 'Dodato';

  @override
  String get notUploaded => 'Nije poslato. Provjerite internet vezu.';

  @override
  String get remove => 'Uklonite';

  @override
  String get removeFile => 'Uklonite datoteku';

  @override
  String get removeFileTitle => 'Ukloniti ovu datoteku?';

  @override
  String get removeFileBody => 'Biće obrisana iz vaših odgovora.';

  @override
  String get fileRejected =>
      'Ova datoteka nije prošla sigurnosnu provjeru i obrisana je. Izaberite drugu datoteku.';

  @override
  String fileTooLarge(int max) {
    return 'Ova datoteka je veća od $max MB.';
  }

  @override
  String get fileTypeRefused =>
      'Ova vrsta datoteke ne može se priložiti uz ovo pitanje.';

  @override
  String tooManyFiles(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Uz ovo pitanje možete priložiti najviše $n datoteka.',
      few: 'Uz ovo pitanje možete priložiti najviše $n datoteke.',
      one: 'Uz ovo pitanje možete priložiti najviše $n datoteku.',
    );
    return '$_temp0';
  }

  @override
  String get filesFull =>
      'Datoteke u ovom upitniku dostigle su ukupno 500 MB. Uklonite neku datoteku da biste dodali novu.';

  @override
  String get fileNotUploaded =>
      'Datoteku nije bilo moguće poslati. Pokušajte ponovo.';

  @override
  String get cannotPrepareFile =>
      'Ovu datoteku nije bilo moguće pripremiti za slanje. Izaberite drugu datoteku.';

  @override
  String get filesBeingChecked =>
      'Neke datoteke se još provjeravaju. Upitnik možete poslati kada budu spremne.';

  @override
  String get leaveUploading =>
      'Datoteka se još šalje. Želite li ipak da izađete? Slanje će biti prekinuto.';

  @override
  String fileCount(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n datoteka',
      few: '$n datoteke',
      one: '$n datoteka',
    );
    return '$_temp0';
  }

  @override
  String get photoKind => 'Fotografija';

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
    return 'Slanje datoteke $title, $n posto';
  }

  @override
  String fileLabel(String kind, String title, String status) {
    return '$kind, $title, $status';
  }

  @override
  String get allowCamera => 'Dozvolite pristup kameri';

  @override
  String get whyCameraForFiles =>
      'Salus treba pristup kameri da biste snimili fotografiju ili video koji dodajete u ovaj upitnik.';

  @override
  String get continueLabel => 'Nastavite';

  @override
  String get cameraDeniedForFiles =>
      'Salus nema pristup vašoj kameri. Pristup možete dozvoliti u podešavanjima uređaja ili umjesto toga izabrati datoteku.';

  @override
  String get photosDenied =>
      'Salus nema pristup vašim fotografijama. Pristup možete dozvoliti u podešavanjima uređaja.';

  @override
  String cameraPhotoTitle(String date, String time) {
    return 'Fotografija, $date, $time';
  }

  @override
  String cameraVideoTitle(String date, String time) {
    return 'Video, $date, $time';
  }

  @override
  String get specialty394814009 => 'Porodična medicina';

  @override
  String get specialty419192003 => 'Interna medicina';

  @override
  String get specialty394579002 => 'Kardiologija';

  @override
  String get specialty394582007 => 'Dermatologija';

  @override
  String get specialty394586005 => 'Ginekologija';

  @override
  String get specialty394537008 => 'Pedijatrija';

  @override
  String get specialty394587001 => 'Psihijatrija';

  @override
  String get specialty394591006 => 'Neurologija';

  @override
  String get specialty418960008 => 'Otorinolaringologija (ORL)';

  @override
  String get specialty394594003 => 'Oftalmologija';

  @override
  String get specialty394801008 => 'Ortopedija';

  @override
  String get specialty394583002 => 'Endokrinologija';

  @override
  String questionnaireNotInLanguage(String language) {
    return 'Ovaj upitnik nije dostupan na jeziku: $language.';
  }

  @override
  String questionnaireLanguages(String languages) {
    return 'Dostupan je na ovim jezicima: $languages. Izaberite onaj koji najbolje razumijete.';
  }

  @override
  String continueInLanguage(String language) {
    return 'Nastavite na jeziku: $language';
  }

  @override
  String get languageUnavailable =>
      'Ovaj jezik više nije dostupan za ovaj upitnik. Izaberite drugi.';

  @override
  String specialtyAtClinic(String specialty, String clinic) {
    return '$specialty, $clinic';
  }

  @override
  String get documents => 'Dokumenti';

  @override
  String get noDocuments =>
      'Za ovaj termin još nema potpisanih dokumenata. Dokument se pojavljuje ovdje kada ga vaš ljekar potpiše.';

  @override
  String documentSignedOn(String date) {
    return 'Potpisano $date';
  }

  @override
  String documentLanguage(String language) {
    return 'Jezik: $language';
  }

  @override
  String get sharePdf => 'Sačuvaj ili podijeli PDF';

  @override
  String get transcript => 'Transkript';

  @override
  String get transcriptNote =>
      'Konsultacija onako kako je automatski transkribovana i prevedena. Može da sadrži greške.';

  @override
  String get transcriptEmpty =>
      'U ovoj konsultaciji ništa nije transkribovano.';

  @override
  String get speakerYou => 'Vi';

  @override
  String get speakerPractitioner => 'Ljekar';

  @override
  String get shareTranscript => 'Podijeli transkript';

  @override
  String transcriptTranslation(String text) {
    return 'Prijevod: $text';
  }

  @override
  String get termExplanationNote => 'Informativno, nije dijagnoza';

  @override
  String get termNotReviewed => 'Još nije provjereno';

  @override
  String get termClose => 'Zatvori';

  @override
  String get chatsTitle => 'Poruke';

  @override
  String get chatsEmpty => 'Još nemate razgovora.';

  @override
  String chatUnread(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count nepročitanih poruka',
      few: '$count nepročitane poruke',
      one: '$count nepročitana poruka',
    );
    return '$_temp0';
  }

  @override
  String get chatDoctorUnknown => 'Vaš ljekar';

  @override
  String get chatMessageHint => 'Napišite poruku';

  @override
  String get chatSend => 'Pošalji';

  @override
  String get chatAttach => 'Dodaj datoteku';

  @override
  String get chatSendFailed =>
      'Poruka nije poslata. Provjerite internet vezu i pokušajte ponovo.';

  @override
  String get chatFileRejected => 'Ova datoteka nije prihvaćena.';

  @override
  String get chatFileRemove => 'Ukloni datoteku';

  @override
  String chatTooManyFiles(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'Već imate $count datoteka koje čekaju slanje. Neposlate datoteke se uklanjaju nakon 24 sata; pokušajte ponovo kasnije.',
      few:
          'Već imate $count datoteke koje čekaju slanje. Neposlate datoteke se uklanjaju nakon 24 sata; pokušajte ponovo kasnije.',
      one:
          'Već imate $count datoteku koja čeka slanje. Neposlate datoteke se uklanjaju nakon 24 sata; pokušajte ponovo kasnije.',
    );
    return '$_temp0';
  }

  @override
  String get chatOpenFailed =>
      'Datoteka nije mogla da se otvori. Pokušajte ponovo.';

  @override
  String get chatEndedSession =>
      'Vaša sesija je istekla. Ponovo se prijavite da biste vidjeli nove poruke.';

  @override
  String get chatEnded =>
      'Nove poruke u ovom razgovoru više ne mogu da se prikažu.';

  @override
  String get chatHours => 'Radno vrijeme za poruke';

  @override
  String chatHoursDay(String day, String start, String end) {
    return '$day: $start–$end';
  }

  @override
  String get chatHoursNone => 'Klinika nije odredila radno vrijeme za poruke.';

  @override
  String chatHoursZone(String zone) {
    return 'Vremena su u vremenskoj zoni klinike, $zone.';
  }

  @override
  String chatResponseTime(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other: 'Klinika obično odgovara u roku od $hours sati.',
      few: 'Klinika obično odgovara u roku od $hours sata.',
      one: 'Klinika obično odgovara u roku od $hours sata.',
    );
    return '$_temp0';
  }

  @override
  String chatOutOfHours(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other:
          'Klinika je trenutno van radnog vremena za poruke. I dalje možete da pišete; obično odgovara u roku od $hours sati.',
      few:
          'Klinika je trenutno van radnog vremena za poruke. I dalje možete da pišete; obično odgovara u roku od $hours sata.',
      one:
          'Klinika je trenutno van radnog vremena za poruke. I dalje možete da pišete; obično odgovara u roku od $hours sata.',
    );
    return '$_temp0';
  }

  @override
  String chatSystemInvited(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Ljekar iz druge klinike je pozvan u ovaj razgovor radi drugog mišljenja.',
      'specialist_review': 'Ljekar iz druge klinike je pozvan u ovaj razgovor radi specijalističkog pregleda.',
      'handover': 'Ljekar iz druge klinike je pozvan u ovaj razgovor da preuzme liječenje.',
      'other': 'Ljekar iz druge klinike je pozvan u ovaj razgovor.',
    });
    return '$_temp0';
  }

  @override
  String chatSystemAdded(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Ljekar se pridružio razgovoru radi drugog mišljenja.',
      'specialist_review':
          'Ljekar se pridružio razgovoru radi specijalističkog pregleda.',
      'handover': 'Ljekar se pridružio razgovoru da preuzme liječenje.',
      'other': 'Ljekar se pridružio razgovoru.',
    });
    return '$_temp0';
  }

  @override
  String get chatSystemRemoved => 'Ljekar više ne učestvuje u razgovoru.';

  @override
  String get chatSystemGranted =>
      'Dozvolili ste ljekaru iz druge klinike da se pridruži razgovoru.';

  @override
  String get chatSystemDeclined => 'Odbili ste ljekara iz druge klinike.';

  @override
  String get chatSystemOther => 'Razgovor je ažuriran.';

  @override
  String get chatInviteTitle =>
      'Ljekar iz druge klinike traži pristup razgovoru';

  @override
  String chatInviteBody(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Vaša klinika traži da se ljekar iz druge klinike pridruži ovom razgovoru radi drugog mišljenja.',
      'specialist_review': 'Vaša klinika traži da se ljekar iz druge klinike pridruži ovom razgovoru radi specijalističkog pregleda.',
      'handover': 'Vaša klinika traži da se ljekar iz druge klinike pridruži ovom razgovoru da preuzme vaše liječenje.',
      'other': 'Vaša klinika traži da se ljekar iz druge klinike pridruži ovom razgovoru.',
    });
    return '$_temp0';
  }

  @override
  String chatInviteWho(String doctor, String clinic) {
    return 'Ljekar: $doctor, $clinic';
  }

  @override
  String get chatInviteUnresolved =>
      'Ime i klinika pozvanog ljekara nisu mogli da se učitaju. Pristup možete da odobrite tek kada budu prikazani.';

  @override
  String chatInviteSees(String history) {
    String _temp0 = intl.Intl.selectLogic(history, {
      'full': 'Ako pristanete, ljekar vidi cijeli razgovor, uključujući ranije poruke, i može da piše u njemu.',
      'from': 'Ako pristanete, ljekar vidi razgovor od poruke koju je izabrala vaša klinika nadalje i može da piše u njemu.',
      'other': 'Ako pristanete, ljekar vidi razgovor i može da piše u njemu.',
    });
    return '$_temp0';
  }

  @override
  String get chatInviteChoice =>
      'Možete da odbijete. Ako pristanete, pristup ljekara možete kasnije da opozovete u ovom razgovoru; prestaje odmah.';

  @override
  String get chatGrant => 'Odobri pristup';

  @override
  String get chatDecline => 'Odbij';

  @override
  String get chatGranted =>
      'Pristup je odobren. Možete da ga opozovete u ovom razgovoru u bilo kom trenutku.';

  @override
  String get chatDeclined =>
      'Odbijeno. Ljekar se ne pridružuje ovom razgovoru.';

  @override
  String get chatRevoke => 'Opozovi pristup ljekara iz druge klinike';

  @override
  String chatGrantedTo(String doctor, String clinic) {
    return 'Pristup je odobren: $doctor, $clinic.';
  }

  @override
  String get chatGrantedToUnknown =>
      'Pristup je odobren ljekaru iz druge klinike.';

  @override
  String get chatRevokeBody =>
      'Ljekar iz druge klinike odmah gubi pristup ovom razgovoru. Već napisane poruke ostaju u njemu.';

  @override
  String get chatRevokeConfirm => 'Opozovi pristup';

  @override
  String get chatRevoked => 'Pristup je opozvan.';

  @override
  String get allergies => 'Alergije';

  @override
  String get currentMedications => 'Trenutni lijekovi';

  @override
  String get allergiesIntro =>
      'Klinike kojima ste dali saglasnost vide alergije iz vašeg profila kada imate termin kod njih.';

  @override
  String get medicationsIntro =>
      'Klinike kojima ste dali saglasnost vide lijekove iz vašeg profila kada imate termin kod njih.';

  @override
  String get noAllergies => 'Niste dodali nijednu alergiju.';

  @override
  String get noMedications => 'Niste dodali nijedan lijek.';

  @override
  String get addAllergy => 'Dodajte alergiju';

  @override
  String get addMedication => 'Dodajte lijek';

  @override
  String get changeAllergy => 'Izmjena alergije';

  @override
  String get changeMedication => 'Izmjena lijeka';

  @override
  String get change => 'Izmijenite';

  @override
  String get allergySubstance => 'Supstanca ili namirnica';

  @override
  String get medicationName => 'Lijek';

  @override
  String get termSearchHint =>
      'Počnite da kucate za pretragu ili upišite svojim riječima.';

  @override
  String get allergyCategory => 'Vrsta (nije obavezno)';

  @override
  String get notSpecified => 'Nije navedeno';

  @override
  String get categoryFood => 'Hrana';

  @override
  String get categoryMedication => 'Lijek';

  @override
  String get categoryEnvironment => 'Okruženje';

  @override
  String get categoryBiologic => 'Biološki preparat';

  @override
  String get reactionOptional => 'Reakcija (nije obavezno)';

  @override
  String get severity => 'Težina';

  @override
  String get severityMild => 'Blaga';

  @override
  String get severityModerate => 'Umjerena';

  @override
  String get severitySevere => 'Teška';

  @override
  String get entryStatus => 'Status';

  @override
  String get allergyActive => 'Aktivna';

  @override
  String get allergyInactive => 'Neaktivna';

  @override
  String get allergyResolved => 'Prošla';

  @override
  String reactionLine(String reaction) {
    return 'Reakcija: $reaction';
  }

  @override
  String reactionWithSeverity(String reaction, String severity) {
    return 'Reakcija: $reaction ($severity)';
  }

  @override
  String get confirmedByClinician => 'Potvrdio ljekar';

  @override
  String get recordedByClinic => 'Unijela klinika tokom jedne od vaših posjeta';

  @override
  String get confirmedChangeNote =>
      'Ako izmijenite ovaj unos, više neće biti potvrđen od ljekara.';

  @override
  String get nameRequired => 'Unesite naziv, najviše 200 znakova.';

  @override
  String get addToProfile => 'Dodajte u moj profil';

  @override
  String get addToProfileTitle => 'Dodati u vaš profil?';

  @override
  String get addToProfileBody =>
      'Tada će ga vidjeti svaka klinika kojoj ste dali saglasnost, bez naziva klinike koja ga je unijela.';

  @override
  String get addedToProfile => 'Sada je u vašem profilu.';

  @override
  String get removeEntryTitle => 'Ukloniti ovaj unos?';

  @override
  String get removeEntryBody =>
      'Više se neće prikazivati ni vama ni bilo kojoj klinici. Ako ga je klinika unijela ili potvrdila, ostaje u vašem medicinskom kartonu, označen kao uklonjen.';

  @override
  String get entryRemoved => 'Unos je uklonjen.';

  @override
  String get entrySaved => 'Sačuvano u vašem profilu.';

  @override
  String get entryChanged =>
      'Ovaj unos je u međuvremenu izmijenjen. Prikazana je najnovija verzija.';

  @override
  String get entryInvalid =>
      'Neki podaci nisu ispravni. Provjerite ih i pokušajte ponovo.';

  @override
  String get entryNotAllowed => 'Ne možete izmijeniti ovaj unos.';

  @override
  String get entryNotFound => 'Ovaj unos više nije dostupan.';

  @override
  String get doseHeading => 'Doza (nije obavezno)';

  @override
  String get doseAmount => 'Količina';

  @override
  String get doseUnit => 'Jedinica';

  @override
  String get unitMg => 'miligrami (mg)';

  @override
  String get unitG => 'grami (g)';

  @override
  String get unitUg => 'mikrogrami (µg)';

  @override
  String get unitMl => 'mililitri (ml)';

  @override
  String get unitIu => 'internacionalne jedinice (IJ)';

  @override
  String get unitTablet => 'tablete';

  @override
  String get unitCapsule => 'kapsule';

  @override
  String get unitDrop => 'kapi';

  @override
  String get unitPuff => 'udisaji (inhalator)';

  @override
  String get unitPatch => 'flasteri';

  @override
  String get doseBoth => 'Unesite i količinu i jedinicu ili ništa od toga.';

  @override
  String get numberAboveZero => 'Unesite broj veći od 0.';

  @override
  String get howOftenHeading => 'Koliko često (nije obavezno)';

  @override
  String get timesLabel => 'Puta';

  @override
  String get everyLabel => 'Svakih';

  @override
  String get periodUnitLabel => 'Jedinica vremena';

  @override
  String get periodHours => 'sati';

  @override
  String get periodDays => 'dana';

  @override
  String get periodWeeks => 'nedjelja';

  @override
  String get periodMonths => 'mjeseci';

  @override
  String get howOftenAll => 'Popunite sva tri polja ili ih ostavite prazna.';

  @override
  String get asNeeded => 'Po potrebi';

  @override
  String get medicationActive => 'Trenutno uzimam';

  @override
  String get medicationStopped => 'Više ne uzimam';

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
    return '$value IJ';
  }

  @override
  String doseTablet(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value tableta',
      few: '$value tablete',
      one: '$value tableta',
    );
    return '$_temp0';
  }

  @override
  String doseCapsule(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value kapsula',
      few: '$value kapsule',
      one: '$value kapsula',
    );
    return '$_temp0';
  }

  @override
  String doseDrop(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value kapi',
      few: '$value kapi',
      one: '$value kap',
    );
    return '$_temp0';
  }

  @override
  String dosePuff(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value udisaja',
      few: '$value udisaja',
      one: '$value udisaj',
    );
    return '$_temp0';
  }

  @override
  String dosePatch(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value flastera',
      few: '$value flastera',
      one: '$value flaster',
    );
    return '$_temp0';
  }

  @override
  String doseLine(String dose) {
    return 'Doza: $dose';
  }

  @override
  String howOftenPerHour(int frequency) {
    return '$frequency× na sat';
  }

  @override
  String howOftenPerDay(int frequency) {
    return '$frequency× dnevno';
  }

  @override
  String howOftenPerWeek(int frequency) {
    return '$frequency× nedjeljno';
  }

  @override
  String howOftenPerMonth(int frequency) {
    return '$frequency× mjesečno';
  }

  @override
  String howOftenEveryHours(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× svakih $every sati',
      few: '$frequency× svaka $every sata',
      one: '$frequency× svaki $every sat',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryDays(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× svakih $every dana',
      few: '$frequency× svaka $every dana',
      one: '$frequency× svaki $every dan',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryWeeks(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× svakih $every nedjelja',
      few: '$frequency× svake $every nedjelje',
      one: '$frequency× svake $every nedjelje',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryMonths(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× svakih $every mjeseci',
      few: '$frequency× svaka $every mjeseca',
      one: '$frequency× svaki $every mjesec',
    );
    return '$_temp0';
  }

  @override
  String get myHistory => 'Moja istorija';

  @override
  String get historyEmpty =>
      'Vaša istorija je još prazna. Ovdje se pojavljuju vaše posjete, dokumenti i odgovori.';

  @override
  String get historyNoMatch =>
      'Ništa u vašoj istoriji ne odgovara ovim filterima.';

  @override
  String get historyClinic => 'Klinika';

  @override
  String get historyAllClinics => 'Sve klinike';

  @override
  String get historyVisits => 'Posjete';

  @override
  String get historyVisit => 'Posjeta';

  @override
  String get historyAnswers => 'Odgovori na upitnik';

  @override
  String get historyFiles => 'Datoteke';

  @override
  String get historyFile => 'Datoteka';

  @override
  String get historyAnyDate => 'Bilo koji datum';

  @override
  String historyDateRange(String from, String to) {
    return 'od $from do $to';
  }

  @override
  String get historyClearDates => 'Poništi datume';

  @override
  String get historyShowMore => 'Prikaži više';

  @override
  String get historyMoreFailed =>
      'Ostatak vaše istorije nije se mogao učitati.';

  @override
  String get historyOpenFailed =>
      'Ova stavka se nije mogla otvoriti. Pokušajte ponovo.';

  @override
  String get sharingToggle =>
      'Podijeli moju istoriju iz drugih klinika sa ovom klinikom';

  @override
  String sharingIntro(String clinic) {
    return 'Označite ranije posjete koje $clinic smije da vidi. Za svaku označenu posjetu ljekari klinike $clinic kod kojih imate termin vide ono što je ovdje prikazano, sa dokumentima posjete, odgovorima na upitnike i datotekama, alergijama i lijekovima evidentiranim u njoj, kao i svime što se posjeti kasnije doda. Posjete koje ne označite ostaju skrivene od $clinic.';
  }

  @override
  String get sharingChangeLater =>
      'Posjete možete da dodate ili uklonite, ili da prestanete da dijelite, u bilo kom trenutku u dijelu Podijeljena istorija. Svaka promjena važi odmah.';

  @override
  String get sharingNoVisits =>
      'Nemate ranijih posjeta u drugim klinikama koje biste mogli da podijelite.';

  @override
  String get sharingSelectAll => 'Označite sve';

  @override
  String get sharingSelectNone => 'Poništite sve';

  @override
  String get sharedHistory => 'Podijeljena istorija';

  @override
  String get sharedHistoryIntro =>
      'Klinike kojima ste dali saglasnost i koliko vaših posjeta iz drugih klinika svaka od njih vidi.';

  @override
  String get sharedHistoryNone => 'Još niste dali saglasnost nijednoj klinici.';

  @override
  String sharedVisits(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count podijeljenih posjeta',
      few: '$count podijeljene posjete',
      one: '$count podijeljena posjeta',
      zero: 'Ništa nije podijeljeno',
    );
    return '$_temp0';
  }

  @override
  String get sharingSaved => 'Vaš izbor je sačuvan i važi odmah.';

  @override
  String get sharingStop => 'Prestanite da dijelite';

  @override
  String get sharingStopTitle => 'Prestati sa dijeljenjem vaše istorije?';

  @override
  String sharingStopBody(String clinic) {
    return '$clinic više neće vidjeti nijednu posjetu iz drugih klinika koju ste sa njom podijelili. Ovo važi odmah.';
  }

  @override
  String get sharingKeep => 'Nastavite da dijelite';

  @override
  String get sharingStopped =>
      'Više ne dijelite svoju istoriju sa ovom klinikom.';

  @override
  String get sharingChanged =>
      'Vaše podijeljene posjete su se u međuvremenu promijenile. Provjerite ih i ponovo sačuvajte.';

  @override
  String get bookedNotShared =>
      'Vaš termin je zakazan, ali vaša istorija nije mogla da bude podijeljena. Možete da je podijelite u dijelu Podijeljena istorija.';

  @override
  String get consentForHistory => 'Posjete podijeljene iz drugih klinika';

  @override
  String get waitlistTitle => 'Moja lista čekanja';

  @override
  String get waitlistJoinButton =>
      'Nema odgovarajućeg termina? Prijavite se na listu čekanja';

  @override
  String get waitlistJoinTitle => 'Prijava na listu čekanja';

  @override
  String get waitlistExplained =>
      'Kada se oslobodi termin kod ovog ljekara, nudi se pacijentima na čekanju, jednom po jednom. Dobićete obavještenje i termin možete da zakažete u aplikaciji dok je rezervisan za vas.';

  @override
  String get waitlistHowLong => 'Koliko dugo?';

  @override
  String waitlistDays(int days) {
    String _temp0 = intl.Intl.pluralLogic(
      days,
      locale: localeName,
      other: '$days dana',
      few: '$days dana',
      one: '$days dan',
    );
    return '$_temp0';
  }

  @override
  String get waitlistWhen => 'U koje doba dana?';

  @override
  String get waitlistAnyTime => 'Bilo kada';

  @override
  String get waitlistMornings => 'Ujutru, od 08:00 do 12:00';

  @override
  String get waitlistAfternoons => 'Poslijepodne, od 12:00 do 17:00';

  @override
  String get waitlistJoin => 'Prijavi se na listu čekanja';

  @override
  String get waitlistJoined => 'Na listi čekanja ste.';

  @override
  String get waitlistEmpty => 'Niste ni na jednoj listi čekanja.';

  @override
  String get waitlistAPractitioner => 'Ljekar';

  @override
  String waitlistUntil(String date) {
    return 'Čekate do $date';
  }

  @override
  String get waitlistOfferTitle => 'Oslobodio se termin za vas';

  @override
  String waitlistHeldUntil(String time) {
    return 'Rezervisan za vas do $time';
  }

  @override
  String get waitlistAcceptEffect =>
      'Prihvatanjem zakazujete ovaj termin. Kada plaćanje bude dostupno u aplikaciji Salus, prihvatanjem ćete ga i platiti.';

  @override
  String get waitlistAccept => 'Prihvati i zakaži';

  @override
  String get waitlistDecline => 'Odbij';

  @override
  String get waitlistBooked =>
      'Vaš termin je zakazan. Naći ćete ga u odjeljku Moji termini.';

  @override
  String get waitlistDeclined =>
      'Odbili ste termin. Ostajete na listi čekanja.';

  @override
  String get waitlistWithdraw => 'Napusti listu čekanja';

  @override
  String get waitlistWithdrawn => 'Napustili ste listu čekanja.';

  @override
  String get waitlistOff => 'Ova ustanova nema listu čekanja.';

  @override
  String get waitlistTooMany =>
      'Već ste pet puta na listi čekanja ove ustanove. Prvo napustite jednu od prijava.';

  @override
  String get waitlistOfferExpired =>
      'Ovaj termin više nije rezervisan za vas. Ostajete na listi čekanja.';
}
