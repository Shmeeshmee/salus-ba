// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Arabic (`ar`).
class AppLocalizationsAr extends AppLocalizations {
  AppLocalizationsAr([String locale = 'ar']) : super(locale);

  @override
  String get tryAgain => 'إعادة المحاولة';

  @override
  String get language => 'اللغة';

  @override
  String get useDeviceLanguage => 'استخدام لغة الجهاز';

  @override
  String get languageSaved => 'تم حفظ اللغة.';

  @override
  String get somethingWrong => 'حدث خطأ ما. يُرجى المحاولة مرة أخرى.';

  @override
  String get unreachable =>
      'تعذّر الاتصال بـ Salus. يُرجى التحقق من اتصالك بالإنترنت والمحاولة مرة أخرى.';

  @override
  String get wrongCredentials => 'البريد الإلكتروني أو كلمة المرور غير صحيحة.';

  @override
  String get wrongCode =>
      'الرمز غير صحيح. يُرجى التحقق من الوقت على جهازك والمحاولة مرة أخرى.';

  @override
  String get challengeEnded =>
      'انتهت مهلة تسجيل الدخول. يُرجى إدخال بريدك الإلكتروني وكلمة المرور مرة أخرى.';

  @override
  String get emailTaken =>
      'يوجد حساب مسجّل بهذا البريد الإلكتروني. يُرجى تسجيل الدخول بدلًا من ذلك.';

  @override
  String get invalidEmail => 'يُرجى إدخال عنوان بريد إلكتروني صحيح.';

  @override
  String get invalidPassword =>
      'يجب أن تتكوّن كلمة المرور من 12 إلى 128 حرفًا.';

  @override
  String get missingPassword => 'يُرجى إدخال كلمة المرور.';

  @override
  String get invalidCode => 'يتكوّن الرمز من 6 أرقام.';

  @override
  String get staffOnly =>
      'هذا التطبيق مخصّص للمرضى. يستخدم موظفو العيادات تطبيق الويب الخاص بـ Salus.';

  @override
  String get accountReady => 'حسابك جاهز. يُرجى تسجيل الدخول للمتابعة.';

  @override
  String get signOutFailed =>
      'تعذّر تسجيل الخروج. يُرجى التحقق من اتصالك بالإنترنت والمحاولة مرة أخرى.';

  @override
  String get tooManyAttemptsLater =>
      'عدد كبير جدًا من المحاولات. يُرجى المحاولة مرة أخرى لاحقًا.';

  @override
  String tooManyAttempts(int minutes) {
    String _temp0 = intl.Intl.pluralLogic(
      minutes,
      locale: localeName,
      other:
          'عدد كبير جدًا من المحاولات. يُرجى المحاولة مرة أخرى بعد $minutes دقيقة.',
      many:
          'عدد كبير جدًا من المحاولات. يُرجى المحاولة مرة أخرى بعد $minutes دقيقة.',
      few:
          'عدد كبير جدًا من المحاولات. يُرجى المحاولة مرة أخرى بعد $minutes دقائق.',
      two: 'عدد كبير جدًا من المحاولات. يُرجى المحاولة مرة أخرى بعد دقيقتين.',
      one: 'عدد كبير جدًا من المحاولات. يُرجى المحاولة مرة أخرى بعد دقيقة واحدة.',
      zero: 'عدد كبير جدًا من المحاولات. يُرجى المحاولة مرة أخرى بعد قليل.',
    );
    return '$_temp0';
  }

  @override
  String get signInTitle => 'تسجيل الدخول إلى Salus';

  @override
  String get email => 'البريد الإلكتروني';

  @override
  String get password => 'كلمة المرور';

  @override
  String get signIn => 'تسجيل الدخول';

  @override
  String get createAnAccount => 'إنشاء حساب';

  @override
  String get codePrompt =>
      'يُرجى إدخال الرمز المكوّن من 6 أرقام من تطبيق المصادقة.';

  @override
  String get code => 'الرمز';

  @override
  String get useAnotherAccount => 'استخدام حساب آخر';

  @override
  String get passwordHint => 'من 12 إلى 128 حرفًا';

  @override
  String get createAccount => 'إنشاء الحساب';

  @override
  String get haveAccount => 'لديّ حساب بالفعل';

  @override
  String signedInAs(String email) {
    return 'مسجّل الدخول باسم $email';
  }

  @override
  String get completeProfile => 'أكمِل ملفك الشخصي';

  @override
  String get completeProfileFirst => 'يُرجى إكمال ملفك الشخصي قبل حجز موعد.';

  @override
  String get bookAppointment => 'حجز موعد';

  @override
  String get myAppointments => 'مواعيدي';

  @override
  String get myConsents => 'موافقاتي';

  @override
  String get myProfile => 'ملفي الشخصي';

  @override
  String get signOut => 'تسجيل الخروج';

  @override
  String get profileIntro =>
      'تطّلع العيادات التي تحجز لديها على هذه البيانات. رقم الهاتف اختياري، ولا يستخدمه الطبيب إلا للتواصل معك بشأن موعد.';

  @override
  String get givenName => 'الاسم الأول';

  @override
  String get familyName => 'اسم العائلة';

  @override
  String get dateOfBirth => 'تاريخ الميلاد';

  @override
  String get dateFormatHint => 'YYYY-MM-DD';

  @override
  String get phoneOptional => 'الهاتف (اختياري)';

  @override
  String phoneHint(String example) {
    return 'مع رمز الدولة، مثل $example';
  }

  @override
  String get save => 'حفظ';

  @override
  String get notNow => 'ليس الآن';

  @override
  String get invalidGivenName => 'يُرجى إدخال اسمك الأول، بحد أقصى 100 حرف.';

  @override
  String get invalidFamilyName => 'يُرجى إدخال اسم عائلتك، بحد أقصى 100 حرف.';

  @override
  String get invalidName =>
      'يُرجى إدخال اسمك الأول واسم عائلتك، بحد أقصى 100 حرف لكل منهما.';

  @override
  String get birthDateFormat => 'يُرجى إدخال تاريخ ميلادك بالصيغة YYYY-MM-DD.';

  @override
  String get birthDateRange =>
      'يُرجى إدخال تاريخ ميلاد في الماضي، من عام 1900 فما بعد.';

  @override
  String invalidPhone(String example) {
    return 'يُرجى إدخال رقم الهاتف مع رمز الدولة، مثل $example.';
  }

  @override
  String get profileChanged =>
      'تم تغيير ملفك الشخصي في هذه الأثناء. يُرجى مراجعته وحفظه مرة أخرى.';

  @override
  String get chooseClinic => 'اختر عيادة';

  @override
  String get noClinics => 'لا توجد عيادات مدرجة حتى الآن.';

  @override
  String get noPractitioners => 'لا يوجد أطباء مدرجون في هذه العيادة حتى الآن.';

  @override
  String get choosePractitioner => 'اختر طبيبًا.';

  @override
  String get noOpenTimes => 'لا توجد لدى هذا الطبيب أوقات متاحة حتى الآن.';

  @override
  String timesIn(String zone) {
    return 'الأوقات بتوقيت $zone.';
  }

  @override
  String get previousDay => 'اليوم السابق';

  @override
  String get nextDay => 'اليوم التالي';

  @override
  String get noFreeTimes => 'لا توجد أوقات متاحة في هذا اليوم.';

  @override
  String get bookThisTime => 'حجز هذا الوقت';

  @override
  String get chooseAnotherTime => 'اختر وقتًا آخر';

  @override
  String get book => 'حجز';

  @override
  String timeRange(String start, String end, String zone) {
    return 'من $start إلى $end ($zone)';
  }

  @override
  String get slotTaken => 'لم يعد هذا الوقت متاحًا. يُرجى اختيار وقت آخر.';

  @override
  String get consentNeeded => 'يتطلّب الحجز لدى هذه العيادة موافقتك.';

  @override
  String get appointmentBooked => 'تم حجز موعدك.';

  @override
  String get unlistedClinic => 'عيادة لم تعد مدرجة';

  @override
  String get cancelTitle => 'هل تريد إلغاء هذا الموعد؟';

  @override
  String cancelBody(String date, String time) {
    return 'سيُلغى موعدك في $date الساعة $time. وستصلك رسالة بريد إلكتروني لتأكيد الإلغاء.';
  }

  @override
  String get keepIt => 'الإبقاء عليه';

  @override
  String get cancelAppointment => 'إلغاء الموعد';

  @override
  String get noAppointments => 'ليست لديك مواعيد حتى الآن.';

  @override
  String get upcoming => 'القادمة';

  @override
  String get past => 'السابقة';

  @override
  String get statusBooked => 'محجوز';

  @override
  String get statusCancelled => 'ملغى';

  @override
  String get statusAttended => 'تم الحضور';

  @override
  String get statusNotAttended => 'لم يتم الحضور';

  @override
  String get appointmentCancelled => 'تم إلغاء موعدك.';

  @override
  String get cancelWindowHint =>
      'يمكن إلغاؤه عبر التطبيق حتى 24 ساعة قبل موعد البدء.';

  @override
  String get cancelWindowClosed =>
      'يبدأ هذا الموعد خلال أقل من 24 ساعة، ولم يعد من الممكن إلغاؤه عبر التطبيق.';

  @override
  String get appointmentClosed => 'لم يعد هذا الموعد محجوزًا.';

  @override
  String consentTitle(String clinic) {
    return 'مشاركة بياناتك مع $clinic';
  }

  @override
  String consentIntro(String clinic) {
    return 'قبل حجزك الأول لدى $clinic، نحتاج إلى موافقتك على مشاركة بياناتك معها.';
  }

  @override
  String get whatIsShared => 'ما الذي تتم مشاركته';

  @override
  String get whatIsSharedBody =>
      'ملفك الشخصي (اسمك وتاريخ ميلادك، ورقم هاتفك إن كنت قد أدخلته) ومواعيدك لدى هذه العيادة.';

  @override
  String get whoSeesIt => 'من يطّلع عليها';

  @override
  String whoSeesItBody(String clinic) {
    return '$clinic. يطّلع الطبيب الذي تحجز لديه على ملفك الشخصي، ويطّلع مسؤولو العيادة على اسمك وأوقات مواعيدك.';
  }

  @override
  String get whatFor => 'لأي غرض';

  @override
  String get whatForBody =>
      'لرعايتك في هذه العيادة فقط: للتحضير لمواعيدك وإجرائها.';

  @override
  String get whoIsResponsible => 'من المسؤول';

  @override
  String whoIsResponsibleBody(String clinic) {
    return '$clinic هي المسؤولة عن بياناتك. وتعالجها Salus نيابةً عن العيادة.';
  }

  @override
  String get readPrivacyNotice => 'قراءة إشعار الخصوصية';

  @override
  String get privacyNoticeFailed => 'تعذّر فتح إشعار الخصوصية.';

  @override
  String get howToRevoke => 'كيفية سحب الموافقة';

  @override
  String get howToRevokeBody =>
      'يمكنك سحب هذه الموافقة في أي وقت من قسم موافقاتي. وعندها لن تعود العيادة قادرة على الاطلاع على ملفك الشخصي. وتحتفظ العيادة باسمك وبتفاصيل المواعيد التي تمت بالفعل، بوصفها جزءًا من سجلات الرعاية لديها. وتُلغى مواعيدك القادمة لديها.';

  @override
  String get agreeAndBook => 'الموافقة والحجز';

  @override
  String get revokeTitle => 'هل تريد سحب موافقتك؟';

  @override
  String revokeBody(String clinic) {
    return 'لن تعود $clinic قادرة على الاطلاع على ملفك الشخصي. وتحتفظ باسمك وبتفاصيل المواعيد التي تمت بالفعل، بوصفها جزءًا من سجلات الرعاية لديها. وستُلغى مواعيدك القادمة لدى $clinic. كما تتوقف مشاركة سجلّك من العيادات الأخرى مع $clinic، ولا تُعيدها الموافقة مجددًا.';
  }

  @override
  String get keepConsent => 'الإبقاء على الموافقة';

  @override
  String get revokeConsent => 'سحب الموافقة';

  @override
  String get noConsents => 'لم تقدّم أي موافقات حتى الآن.';

  @override
  String get consentsIntro =>
      'تشارك ملفك الشخصي ومواعيدك مع هذه العيادات لأغراض رعايتك.';

  @override
  String givenOn(String date) {
    return 'قُدّمت في $date';
  }

  @override
  String get consentActive => 'سارية';

  @override
  String revokedOn(String date) {
    return 'سُحبت في $date';
  }

  @override
  String get consentRevoked =>
      'تم سحب موافقتك. ستُلغى مواعيدك القادمة لدى هذه العيادة، وستصلك رسالة بريد إلكتروني عن كل موعد منها.';

  @override
  String get consentAlreadyRevoked => 'تم سحب هذه الموافقة من قبل.';

  @override
  String get joinConsultation => 'الانضمام إلى الاستشارة';

  @override
  String get joinFrom => 'يمكنك الانضمام قبل موعد البدء بـ 10 دقائق.';

  @override
  String get checkTitle => 'تحقّق من الكاميرا والميكروفون';

  @override
  String get checkIntro => 'سيراك الطبيب ويسمعك بمجرد انضمامك.';

  @override
  String get noSound =>
      'لم يُكتشف أي صوت. يُرجى التحقق من أن الميكروفون يعمل وغير مكتوم.';

  @override
  String get joinCameraOff => 'الانضمام والكاميرا مطفأة';

  @override
  String get joinMicrophoneOff => 'الانضمام والميكروفون مطفأ';

  @override
  String get joinNow => 'انضمام';

  @override
  String get joining => 'جارٍ الانضمام…';

  @override
  String get cameraDenied =>
      'لا يملك Salus إذنًا بالوصول إلى الكاميرا. يمكنك السماح بذلك من إعدادات جهازك، أو الانضمام دون فيديو.';

  @override
  String get microphoneDenied =>
      'لا يملك Salus إذنًا بالوصول إلى الميكروفون. يمكنك السماح بذلك من إعدادات جهازك. ومن دونه لن يتمكّن الطبيب من سماعك.';

  @override
  String get openSettings => 'فتح الإعدادات';

  @override
  String get noCamera => 'لم يُعثر على كاميرا.';

  @override
  String get noMicrophone => 'لم يُعثر على ميكروفون.';

  @override
  String get waiting => 'في انتظار الطبيب';

  @override
  String joinsYouHere(String name) {
    return 'سينضم إليك $name هنا. يُرجى إبقاء هذه الشاشة مفتوحة.';
  }

  @override
  String get mute => 'كتم الميكروفون';

  @override
  String get unmute => 'إلغاء كتم الميكروفون';

  @override
  String get cameraOff => 'إطفاء الكاميرا';

  @override
  String get cameraOn => 'تشغيل الكاميرا';

  @override
  String get leave => 'مغادرة';

  @override
  String peerCameraOff(String name) {
    return 'كاميرا $name مطفأة';
  }

  @override
  String peerMuted(String name) {
    return 'ميكروفون $name مكتوم';
  }

  @override
  String peerLeft(String name) {
    return 'غادر $name الاستشارة. ويمكنه الانضمام مجددًا حتى انتهاء الاستشارة.';
  }

  @override
  String get reconnecting => 'انقطع الاتصال. جارٍ إعادة الاتصال…';

  @override
  String get notRestored => 'تعذّرت استعادة الاتصال.';

  @override
  String get reconnected => 'تمت إعادة الاتصال';

  @override
  String get consultationOver => 'انتهت الاستشارة';

  @override
  String get closeScreen => 'يمكنك إغلاق هذه الشاشة.';

  @override
  String get timeLimit => 'بلغت الاستشارة الحد الزمني المسموح به وانتهت.';

  @override
  String get practitionerAbsent => 'لم ينضم الطبيب';

  @override
  String get timePassed =>
      'انقضى وقت الانضمام إلى هذه الاستشارة. يمكن للعيادة أن ترى أنك انضممت. وقد تتواصل العيادة معك، أو يمكنك حجز موعد جديد.';

  @override
  String get withdrewConsent =>
      'لقد سحبت موافقتك لهذه العيادة، ولذلك غادرت الاستشارة.';

  @override
  String get appointmentWasCancelled =>
      'أُلغي هذا الموعد، ولذلك انتهت الاستشارة.';

  @override
  String get windowNotOpen => 'تُفتح هذه الاستشارة قبل موعد البدء بـ 10 دقائق.';

  @override
  String get consultationEnded => 'انتهت هذه الاستشارة.';

  @override
  String get joinNeedsConsent => 'يتطلّب الانضمام موافقتك لهذه العيادة.';

  @override
  String get consultationNotAvailable => 'هذه الاستشارة غير متاحة.';

  @override
  String get cannotConnect =>
      'تعذّر بدء اتصال الفيديو. يُرجى التحقق من اتصالك بالإنترنت والمحاولة مرة أخرى.';

  @override
  String get whyCameraAndMicrophone =>
      'للانضمام إلى استشارة الفيديو، يحتاج Salus إلى الكاميرا والميكروفون. لا تُسجَّل الاستشارة صوتًا ولا فيديو. وإذا وافقت، يُنسخ ما يُقال نصيًا ويُترجم.';

  @override
  String get transcriptionOn => 'النسخ النصي مفعّل';

  @override
  String get transcriptionTitle => 'النسخ النصي والترجمة';

  @override
  String transcriptionIntro(String clinic) {
    return 'في استشارات الفيديو مع $clinic، يمكن نسخ ما تقوله أنت والطبيب نصيًا أثناء الحديث وترجمته، ليتمكن الطبيب من متابعتك بلغتك.';
  }

  @override
  String get transcriptionAssistance =>
      'ويحصل الطبيب أيضًا على مساعدة من الذكاء الاصطناعي انطلاقًا من النص المنسوخ: اقتراحات أثناء الاستشارة ومسودات للمستندات. وهي مجرد اقتراحات، يراجعها الطبيب ويتخذ كل قرار بنفسه.';

  @override
  String transcriptionKept(String clinic) {
    return 'يُحفظ النص المنسوخ مع سجلك الطبي لدى $clinic. لا تُسجَّل الاستشارة صوتًا ولا فيديو.';
  }

  @override
  String get transcriptionDecline =>
      'إذا لم توافق، تجري الاستشارة كالمعتاد، دون نسخ نصي أو ترجمة أو مساعدة من الذكاء الاصطناعي.';

  @override
  String get transcriptionWithdraw =>
      'يمكنك سحب موافقتك في أي وقت، حتى أثناء الاستشارة: اضغط على «النسخ النصي مفعّل» أو افتح موافقاتي. يتوقف النسخ النصي فورًا، ويبقى ما نُسخ حتى ذلك الحين في سجلك.';

  @override
  String get transcriptionAgree => 'الموافقة والمتابعة';

  @override
  String get transcriptionWithout => 'المتابعة دون نسخ نصي';

  @override
  String get stopTranscriptionTitle => 'هل تريد إيقاف النسخ النصي؟';

  @override
  String stopTranscriptionBody(String clinic) {
    return 'يتوقف النسخ النصي والترجمة ومساعدة الذكاء الاصطناعي فورًا في استشاراتك مع $clinic. تستمر الاستشارة الجارية بدونها، وتبقى مواعيدك كما هي. ويبقى ما نُسخ حتى الآن في سجلك.';
  }

  @override
  String get keepTranscription => 'الإبقاء على النسخ النصي';

  @override
  String get stopTranscription => 'إيقاف النسخ النصي';

  @override
  String get transcriptionStopped =>
      'النسخ النصي متوقف. تجري استشاراتك مع هذه العيادة بدونه.';

  @override
  String get consentForCare => 'المشاركة لأغراض الرعاية';

  @override
  String get consentForTranscription => 'النسخ النصي ومساعدة الذكاء الاصطناعي';

  @override
  String get toFillIn => 'بانتظار التعبئة';

  @override
  String get inProgress => 'قيد التعبئة';

  @override
  String get submittedStatus => 'مُرسَل';

  @override
  String get closedStatus => 'مُغلَق';

  @override
  String get fillIn => 'تعبئة';

  @override
  String get continueFilling => 'متابعة';

  @override
  String get viewAnswers => 'عرض الإجابات';

  @override
  String get submitBeforeStart => 'يُرجى إرساله قبل بدء موعدك.';

  @override
  String get questionnaireClosed =>
      'بدأ الموعد، ولذلك لم يعد من الممكن تعبئة هذا الاستبيان.';

  @override
  String savedAsYouGo(String clinic) {
    return 'تُحفظ إجاباتك أولًا بأول. ولا تطّلع عليها $clinic إلا بعد إرسالها.';
  }

  @override
  String get saved => 'تم الحفظ';

  @override
  String get saving => 'جارٍ الحفظ…';

  @override
  String get notSaved =>
      'لم يتم الحفظ. يُرجى التحقق من اتصالك بالإنترنت. ستُعاد محاولة الحفظ تلقائيًا.';

  @override
  String get requiredQuestion => 'إلزامي';

  @override
  String get moreInformation => 'مزيد من المعلومات';

  @override
  String get yourAnswer => 'إجابتك';

  @override
  String get chooseDate => 'اختر تاريخًا';

  @override
  String get selectOne => 'اختر إجابة واحدة.';

  @override
  String get selectAll => 'اختر كل ما ينطبق.';

  @override
  String get yes => 'نعم';

  @override
  String get no => 'لا';

  @override
  String get questionRequired => 'هذا السؤال إلزامي.';

  @override
  String get enterWholeNumber => 'يُرجى إدخال عدد صحيح.';

  @override
  String get enterNumber => 'يُرجى إدخال رقم.';

  @override
  String get numberTooLarge => 'هذا الرقم كبير جدًا.';

  @override
  String upToCharacters(int max) {
    return 'يُرجى إدخال $max حرف كحد أقصى.';
  }

  @override
  String get enterValidDate => 'يُرجى إدخال تاريخ صحيح.';

  @override
  String get submit => 'إرسال';

  @override
  String get submitTitle => 'هل تريد إرسال إجاباتك؟';

  @override
  String submitBody(String clinic) {
    return 'لا يمكنك تغييرها بعد الإرسال. وسيطّلع عليها طبيبك في $clinic.';
  }

  @override
  String get cancel => 'إلغاء';

  @override
  String get requiredMissing => 'لم تتم الإجابة عن بعض الأسئلة الإلزامية.';

  @override
  String get answersSubmitted => 'تم إرسال الإجابات';

  @override
  String get answersSubmittedBody =>
      'يمكن لطبيبك الاطلاع عليها قبل موعدك. ويمكنك عرضها من قسم مواعيدي.';

  @override
  String get yourAnswers => 'إجاباتك';

  @override
  String submittedAt(String when) {
    return 'أُرسلت في $when';
  }

  @override
  String dateAndTime(String date, String time) {
    return '$date، $time';
  }

  @override
  String get notAnswered => 'لم تتم الإجابة';

  @override
  String get leaveUnsaved =>
      'لم تُحفظ إجابتك الأخيرة بعد. هل تريد المغادرة على أي حال؟';

  @override
  String get stay => 'البقاء';

  @override
  String get fillingNeedsConsent =>
      'تتطلّب تعبئة هذا الاستبيان موافقتك لهذه العيادة.';

  @override
  String get pastDue =>
      'بدأ الموعد، ولذلك لم يعد من الممكن تغيير هذا الاستبيان.';

  @override
  String get withdrawnByClinic =>
      'سحبت العيادة هذا الاستبيان. لست بحاجة إلى تعبئته.';

  @override
  String get alreadySubmitted => 'لقد أرسلت هذا الاستبيان من قبل.';

  @override
  String get changedElsewhere =>
      'تم تغيير هذه الإجابات على جهاز آخر. تُعرض أحدث الإجابات.';

  @override
  String get questionnaireNotAvailable => 'هذا الاستبيان غير متاح.';

  @override
  String get done => 'تم';

  @override
  String get addFile => 'إضافة ملف';

  @override
  String get photosUpTo => 'صور بحجم يصل إلى 20 MB.';

  @override
  String get documentsUpTo => 'مستندات (PDF) بحجم يصل إلى 20 MB.';

  @override
  String get videosUpTo => 'مقاطع فيديو بحجم يصل إلى 200 MB.';

  @override
  String get oneFileHint => 'ملف واحد.';

  @override
  String upToFilesHint(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'حتى $n ملف.',
      many: 'حتى $n ملفًا.',
      few: 'حتى $n ملفات.',
      two: 'حتى ملفين.',
      one: 'حتى ملف واحد.',
      zero: 'حتى $n ملف.',
    );
    return '$_temp0';
  }

  @override
  String get takePhoto => 'التقاط صورة';

  @override
  String get recordVideo => 'تسجيل فيديو';

  @override
  String get chooseFromPhotos => 'الاختيار من الصور';

  @override
  String get chooseDocument => 'اختيار مستند';

  @override
  String get preparing => 'جارٍ التحضير…';

  @override
  String uploadingPercent(int n) {
    return 'جارٍ الرفع… $n%';
  }

  @override
  String get checkingFile => 'جارٍ فحص الملف…';

  @override
  String get added => 'تمت الإضافة';

  @override
  String get notUploaded => 'لم يتم الرفع. يُرجى التحقق من اتصالك بالإنترنت.';

  @override
  String get remove => 'إزالة';

  @override
  String get removeFile => 'إزالة الملف';

  @override
  String get removeFileTitle => 'هل تريد إزالة هذا الملف؟';

  @override
  String get removeFileBody => 'سيُحذف من إجاباتك.';

  @override
  String get fileRejected =>
      'لم يجتز هذا الملف فحص الأمان وتم حذفه. يُرجى اختيار ملف آخر.';

  @override
  String fileTooLarge(int max) {
    return 'حجم هذا الملف أكبر من $max MB.';
  }

  @override
  String get fileTypeRefused =>
      'لا يمكن إضافة هذا النوع من الملفات إلى هذا السؤال.';

  @override
  String tooManyFiles(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'يمكنك إضافة $n ملف كحد أقصى إلى هذا السؤال.',
      many: 'يمكنك إضافة $n ملفًا كحد أقصى إلى هذا السؤال.',
      few: 'يمكنك إضافة $n ملفات كحد أقصى إلى هذا السؤال.',
      two: 'يمكنك إضافة ملفين كحد أقصى إلى هذا السؤال.',
      one: 'يمكنك إضافة ملف واحد كحد أقصى إلى هذا السؤال.',
      zero: 'لا يمكنك إضافة ملفات إلى هذا السؤال.',
    );
    return '$_temp0';
  }

  @override
  String get filesFull =>
      'بلغ حجم الملفات في هذا الاستبيان 500 MB. يُرجى إزالة ملف لإضافة ملف آخر.';

  @override
  String get fileNotUploaded => 'تعذّر رفع الملف. يُرجى المحاولة مرة أخرى.';

  @override
  String get cannotPrepareFile =>
      'تعذّر تحضير هذا الملف للرفع. يُرجى اختيار ملف آخر.';

  @override
  String get filesBeingChecked =>
      'لا يزال فحص بعض الملفات جاريًا. يمكنك الإرسال بمجرد أن تصبح جاهزة.';

  @override
  String get leaveUploading =>
      'لا يزال رفع أحد الملفات جاريًا. هل تريد المغادرة على أي حال؟ سيتوقف الرفع.';

  @override
  String fileCount(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n ملف',
      many: '$n ملفًا',
      few: '$n ملفات',
      two: 'ملفان',
      one: 'ملف واحد',
      zero: '$n ملف',
    );
    return '$_temp0';
  }

  @override
  String get photoKind => 'صورة';

  @override
  String get documentKind => 'مستند';

  @override
  String get videoKind => 'فيديو';

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
    return '$kind، $size';
  }

  @override
  String uploadingLabel(String title, int n) {
    return 'جارٍ رفع $title، $n بالمئة';
  }

  @override
  String fileLabel(String kind, String title, String status) {
    return '$kind، $title، $status';
  }

  @override
  String get allowCamera => 'السماح بالوصول إلى الكاميرا';

  @override
  String get whyCameraForFiles =>
      'يحتاج Salus إلى الكاميرا لالتقاط الصورة أو تسجيل الفيديو الذي تضيفه إلى هذا الاستبيان.';

  @override
  String get continueLabel => 'متابعة';

  @override
  String get cameraDeniedForFiles =>
      'لا يملك Salus إذنًا بالوصول إلى الكاميرا. يمكنك السماح بذلك من إعدادات جهازك، أو اختيار ملف بدلًا من ذلك.';

  @override
  String get photosDenied =>
      'لا يملك Salus إذنًا بالوصول إلى صورك. يمكنك السماح بذلك من إعدادات جهازك.';

  @override
  String cameraPhotoTitle(String date, String time) {
    return 'صورة، $date، $time';
  }

  @override
  String cameraVideoTitle(String date, String time) {
    return 'فيديو، $date، $time';
  }

  @override
  String get specialty394814009 => 'طب الأسرة';

  @override
  String get specialty419192003 => 'الطب الباطني';

  @override
  String get specialty394579002 => 'أمراض القلب';

  @override
  String get specialty394582007 => 'الأمراض الجلدية';

  @override
  String get specialty394586005 => 'أمراض النساء';

  @override
  String get specialty394537008 => 'طب الأطفال';

  @override
  String get specialty394587001 => 'الطب النفسي';

  @override
  String get specialty394591006 => 'طب الأعصاب';

  @override
  String get specialty418960008 => 'الأنف والأذن والحنجرة';

  @override
  String get specialty394594003 => 'طب العيون';

  @override
  String get specialty394801008 => 'جراحة العظام';

  @override
  String get specialty394583002 => 'الغدد الصماء';

  @override
  String questionnaireNotInLanguage(String language) {
    return 'هذا الاستبيان غير متاح باللغة: $language.';
  }

  @override
  String questionnaireLanguages(String languages) {
    return 'هذا الاستبيان متاح باللغات التالية: $languages. اختر اللغة التي تفهمها أكثر من غيرها.';
  }

  @override
  String continueInLanguage(String language) {
    return 'المتابعة باللغة: $language';
  }

  @override
  String get languageUnavailable =>
      'لم تعد هذه اللغة متاحة لهذا الاستبيان. اختر لغة أخرى.';

  @override
  String specialtyAtClinic(String specialty, String clinic) {
    return '$specialty، $clinic';
  }

  @override
  String get documents => 'المستندات';

  @override
  String get noDocuments =>
      'لا توجد بعد مستندات موقعة لهذا الموعد. يظهر المستند هنا بعد أن يوقعه طبيبك.';

  @override
  String documentSignedOn(String date) {
    return 'تم التوقيع في $date';
  }

  @override
  String documentLanguage(String language) {
    return 'اللغة: $language';
  }

  @override
  String get sharePdf => 'حفظ ملف PDF أو مشاركته';

  @override
  String get transcript => 'النص المكتوب';

  @override
  String get transcriptNote =>
      'الاستشارة كما تم تحويلها إلى نص وترجمتها تلقائيًا. قد تحتوي على أخطاء.';

  @override
  String get transcriptEmpty => 'لم يُحوَّل أي شيء إلى نص في هذه الاستشارة.';

  @override
  String get speakerYou => 'أنت';

  @override
  String get speakerPractitioner => 'الطبيب';

  @override
  String get shareTranscript => 'مشاركة النص المكتوب';

  @override
  String transcriptTranslation(String text) {
    return 'الترجمة: $text';
  }

  @override
  String get termExplanationNote => 'للعلم فقط، وليس تشخيصًا';

  @override
  String get termNotReviewed => 'لم تتم مراجعته بعد';

  @override
  String get termClose => 'إغلاق';

  @override
  String get chatsTitle => 'الرسائل';

  @override
  String get chatsEmpty => 'ليست لديك محادثات بعد.';

  @override
  String chatUnread(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count رسالة غير مقروءة',
      many: '$count رسالة غير مقروءة',
      few: '$count رسائل غير مقروءة',
      two: 'رسالتان غير مقروءتين',
      one: 'رسالة واحدة غير مقروءة',
      zero: 'لا توجد رسائل غير مقروءة',
    );
    return '$_temp0';
  }

  @override
  String get chatDoctorUnknown => 'طبيبك';

  @override
  String get chatMessageHint => 'اكتب رسالة';

  @override
  String get chatSend => 'إرسال';

  @override
  String get chatAttach => 'إضافة ملف';

  @override
  String get chatSendFailed =>
      'لم تُرسَل الرسالة. تحقّق من اتصالك وحاول مرة أخرى.';

  @override
  String get chatFileRejected => 'لم يُقبَل هذا الملف.';

  @override
  String get chatFileRemove => 'إزالة الملف';

  @override
  String chatTooManyFiles(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'لديك بالفعل $count ملف بانتظار الإرسال. تُزال الملفات غير المرسلة بعد 24 ساعة؛ حاول مرة أخرى لاحقًا.',
      many:
          'لديك بالفعل $count ملفًا بانتظار الإرسال. تُزال الملفات غير المرسلة بعد 24 ساعة؛ حاول مرة أخرى لاحقًا.',
      few:
          'لديك بالفعل $count ملفات بانتظار الإرسال. تُزال الملفات غير المرسلة بعد 24 ساعة؛ حاول مرة أخرى لاحقًا.',
      two: 'لديك بالفعل ملفان بانتظار الإرسال. تُزال الملفات غير المرسلة بعد 24 ساعة؛ حاول مرة أخرى لاحقًا.',
      one: 'لديك بالفعل ملف واحد بانتظار الإرسال. تُزال الملفات غير المرسلة بعد 24 ساعة؛ حاول مرة أخرى لاحقًا.',
      zero: 'لا توجد لديك ملفات بانتظار الإرسال. تُزال الملفات غير المرسلة بعد 24 ساعة؛ حاول مرة أخرى لاحقًا.',
    );
    return '$_temp0';
  }

  @override
  String get chatOpenFailed => 'تعذّر فتح الملف. حاول مرة أخرى.';

  @override
  String get chatEndedSession =>
      'انتهت جلستك. سجّل الدخول مرة أخرى لرؤية الرسائل الجديدة.';

  @override
  String get chatEnded => 'لم يعد بالإمكان عرض رسائل جديدة في هذه المحادثة.';

  @override
  String get chatHours => 'مواعيد الرسائل';

  @override
  String chatHoursDay(String day, String start, String end) {
    return '$day: $start–$end';
  }

  @override
  String get chatHoursNone => 'لم تحدّد العيادة مواعيد للرسائل.';

  @override
  String chatHoursZone(String zone) {
    return 'الأوقات بتوقيت العيادة، $zone.';
  }

  @override
  String chatResponseTime(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other: 'تردّ العيادة عادةً خلال $hours ساعة.',
      many: 'تردّ العيادة عادةً خلال $hours ساعة.',
      few: 'تردّ العيادة عادةً خلال $hours ساعات.',
      two: 'تردّ العيادة عادةً خلال ساعتين.',
      one: 'تردّ العيادة عادةً خلال ساعة واحدة.',
      zero: 'تردّ العيادة عادةً خلال $hours ساعة.',
    );
    return '$_temp0';
  }

  @override
  String chatOutOfHours(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other:
          'العيادة خارج مواعيد الرسائل الآن. يمكنك الكتابة مع ذلك، وتردّ عادةً خلال $hours ساعة.',
      many:
          'العيادة خارج مواعيد الرسائل الآن. يمكنك الكتابة مع ذلك، وتردّ عادةً خلال $hours ساعة.',
      few:
          'العيادة خارج مواعيد الرسائل الآن. يمكنك الكتابة مع ذلك، وتردّ عادةً خلال $hours ساعات.',
      two: 'العيادة خارج مواعيد الرسائل الآن. يمكنك الكتابة مع ذلك، وتردّ عادةً خلال ساعتين.',
      one: 'العيادة خارج مواعيد الرسائل الآن. يمكنك الكتابة مع ذلك، وتردّ عادةً خلال ساعة واحدة.',
      zero:
          'العيادة خارج مواعيد الرسائل الآن. يمكنك الكتابة مع ذلك، وتردّ عادةً خلال $hours ساعة.',
    );
    return '$_temp0';
  }

  @override
  String chatSystemInvited(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion':
          'دُعي طبيب من عيادة أخرى إلى هذه المحادثة لإبداء رأي ثانٍ.',
      'specialist_review':
          'دُعي طبيب من عيادة أخرى إلى هذه المحادثة لتقييم تخصصي.',
      'handover': 'دُعي طبيب من عيادة أخرى إلى هذه المحادثة لتولّي الرعاية.',
      'other': 'دُعي طبيب من عيادة أخرى إلى هذه المحادثة.',
    });
    return '$_temp0';
  }

  @override
  String chatSystemAdded(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'انضم طبيب إلى المحادثة لإبداء رأي ثانٍ.',
      'specialist_review': 'انضم طبيب إلى المحادثة لتقييم تخصصي.',
      'handover': 'انضم طبيب إلى المحادثة لتولّي الرعاية.',
      'other': 'انضم طبيب إلى المحادثة.',
    });
    return '$_temp0';
  }

  @override
  String get chatSystemRemoved => 'لم يعد أحد الأطباء مشاركًا في المحادثة.';

  @override
  String get chatSystemGranted =>
      'سمحت لطبيب من عيادة أخرى بالانضمام إلى المحادثة.';

  @override
  String get chatSystemDeclined => 'رفضت طبيبًا من عيادة أخرى.';

  @override
  String get chatSystemOther => 'جرى تحديث المحادثة.';

  @override
  String get chatInviteTitle => 'طبيب من عيادة أخرى يطلب الانضمام';

  @override
  String chatInviteBody(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'تطلب عيادتك انضمام طبيب من عيادة أخرى إلى هذه المحادثة لإبداء رأي ثانٍ.',
      'specialist_review': 'تطلب عيادتك انضمام طبيب من عيادة أخرى إلى هذه المحادثة لتقييم تخصصي.',
      'handover': 'تطلب عيادتك انضمام طبيب من عيادة أخرى إلى هذه المحادثة لتولّي رعايتك.',
      'other': 'تطلب عيادتك انضمام طبيب من عيادة أخرى إلى هذه المحادثة.',
    });
    return '$_temp0';
  }

  @override
  String chatInviteWho(String doctor, String clinic) {
    return 'الطبيب: $doctor، $clinic';
  }

  @override
  String get chatInviteUnresolved =>
      'تعذّر تحميل اسم الطبيب المدعو وعيادته. يمكنك منح الوصول فقط بعد عرضهما.';

  @override
  String chatInviteSees(String history) {
    String _temp0 = intl.Intl.selectLogic(history, {
      'full': 'إذا وافقت، يرى الطبيب المحادثة كاملة، بما فيها الرسائل السابقة، ويمكنه الكتابة فيها.',
      'from': 'إذا وافقت، يرى الطبيب المحادثة ابتداءً من رسالة اختارتها عيادتك، ويمكنه الكتابة فيها.',
      'other': 'إذا وافقت، يرى الطبيب المحادثة ويمكنه الكتابة فيها.',
    });
    return '$_temp0';
  }

  @override
  String get chatInviteChoice =>
      'يمكنك الرفض. وإذا وافقت، يمكنك لاحقًا إلغاء وصول الطبيب من هذه المحادثة، وينتهي فورًا.';

  @override
  String get chatGrant => 'منح الوصول';

  @override
  String get chatDecline => 'رفض';

  @override
  String get chatGranted =>
      'تم منح الوصول. يمكنك إلغاؤه من هذه المحادثة في أي وقت.';

  @override
  String get chatDeclined => 'تم الرفض. لن ينضم الطبيب إلى هذه المحادثة.';

  @override
  String get chatRevoke => 'إلغاء وصول الطبيب من العيادة الأخرى';

  @override
  String chatGrantedTo(String doctor, String clinic) {
    return 'تم منح الوصول: $doctor، $clinic.';
  }

  @override
  String get chatGrantedToUnknown => 'تم منح الوصول لطبيب من عيادة أخرى.';

  @override
  String get chatRevokeBody =>
      'يفقد الطبيب من العيادة الأخرى الوصول إلى هذه المحادثة فورًا. وتبقى الرسائل المكتوبة سابقًا فيها.';

  @override
  String get chatRevokeConfirm => 'إلغاء الوصول';

  @override
  String get chatRevoked => 'تم إلغاء الوصول.';

  @override
  String get allergies => 'الحساسية';

  @override
  String get currentMedications => 'الأدوية الحالية';

  @override
  String get allergiesIntro =>
      'ترى العيادات التي منحتها موافقتك أنواع الحساسية المسجلة في ملفك الشخصي عندما يكون لديك موعد لديها.';

  @override
  String get medicationsIntro =>
      'ترى العيادات التي منحتها موافقتك الأدوية المسجلة في ملفك الشخصي عندما يكون لديك موعد لديها.';

  @override
  String get noAllergies => 'لم تضف أي حساسية حتى الآن.';

  @override
  String get noMedications => 'لم تضف أي دواء حتى الآن.';

  @override
  String get addAllergy => 'إضافة حساسية';

  @override
  String get addMedication => 'إضافة دواء';

  @override
  String get changeAllergy => 'تعديل الحساسية';

  @override
  String get changeMedication => 'تعديل الدواء';

  @override
  String get change => 'تعديل';

  @override
  String get allergySubstance => 'المادة أو الطعام';

  @override
  String get medicationName => 'الدواء';

  @override
  String get termSearchHint => 'اكتب للبحث، أو اكتبه بكلماتك الخاصة.';

  @override
  String get allergyCategory => 'النوع (اختياري)';

  @override
  String get notSpecified => 'غير محدد';

  @override
  String get categoryFood => 'طعام';

  @override
  String get categoryMedication => 'دواء';

  @override
  String get categoryEnvironment => 'البيئة';

  @override
  String get categoryBiologic => 'منتج بيولوجي';

  @override
  String get reactionOptional => 'رد الفعل (اختياري)';

  @override
  String get severity => 'الشدة';

  @override
  String get severityMild => 'خفيفة';

  @override
  String get severityModerate => 'متوسطة';

  @override
  String get severitySevere => 'شديدة';

  @override
  String get entryStatus => 'الحالة';

  @override
  String get allergyActive => 'نشطة';

  @override
  String get allergyInactive => 'غير نشطة';

  @override
  String get allergyResolved => 'زالت';

  @override
  String reactionLine(String reaction) {
    return 'رد الفعل: $reaction';
  }

  @override
  String reactionWithSeverity(String reaction, String severity) {
    return 'رد الفعل: $reaction ($severity)';
  }

  @override
  String get confirmedByClinician => 'أكّده طبيب';

  @override
  String get recordedByClinic => 'سجّلته عيادة خلال إحدى زياراتك';

  @override
  String get confirmedChangeNote =>
      'إذا عدّلت هذا الإدخال، فلن يعود مؤكدًا من طبيب.';

  @override
  String get nameRequired => 'يُرجى إدخال اسم، بحد أقصى 200 حرف.';

  @override
  String get addToProfile => 'إضافة إلى ملفي الشخصي';

  @override
  String get addToProfileTitle => 'هل تريد إضافته إلى ملفك الشخصي؟';

  @override
  String get addToProfileBody =>
      'ستراه عندئذٍ كل عيادة منحتها موافقتك، دون اسم العيادة التي سجّلته.';

  @override
  String get addedToProfile => 'أصبح الآن في ملفك الشخصي.';

  @override
  String get removeEntryTitle => 'هل تريد إزالة هذا الإدخال؟';

  @override
  String get removeEntryBody =>
      'لن يُعرض بعد الآن لك ولا لأي عيادة. إذا كانت عيادة قد سجّلته أو أكّدته، فإنه يبقى في سجلك الطبي مع وسمه بأنه أُزيل.';

  @override
  String get entryRemoved => 'تمت إزالة الإدخال.';

  @override
  String get entrySaved => 'تم الحفظ في ملفك الشخصي.';

  @override
  String get entryChanged =>
      'تم تغيير هذا الإدخال في هذه الأثناء. تُعرض أحدث نسخة.';

  @override
  String get entryInvalid =>
      'بعض البيانات غير صالحة. يُرجى مراجعتها والمحاولة مرة أخرى.';

  @override
  String get entryNotAllowed => 'لا يمكنك تعديل هذا الإدخال.';

  @override
  String get entryNotFound => 'هذا الإدخال لم يعد متاحًا.';

  @override
  String get doseHeading => 'الجرعة (اختياري)';

  @override
  String get doseAmount => 'الكمية';

  @override
  String get doseUnit => 'الوحدة';

  @override
  String get unitMg => 'مليغرام (ملغ)';

  @override
  String get unitG => 'غرام (غ)';

  @override
  String get unitUg => 'ميكروغرام (مكغ)';

  @override
  String get unitMl => 'مليلتر (مل)';

  @override
  String get unitIu => 'وحدات دولية';

  @override
  String get unitTablet => 'أقراص';

  @override
  String get unitCapsule => 'كبسولات';

  @override
  String get unitDrop => 'قطرات';

  @override
  String get unitPuff => 'بخات (بخاخ)';

  @override
  String get unitPatch => 'لصقات';

  @override
  String get doseBoth => 'يُرجى إدخال الكمية والوحدة معًا، أو ترك كليهما.';

  @override
  String get numberAboveZero => 'يُرجى إدخال رقم أكبر من 0.';

  @override
  String get howOftenHeading => 'كم مرة (اختياري)';

  @override
  String get timesLabel => 'عدد المرات';

  @override
  String get everyLabel => 'كل';

  @override
  String get periodUnitLabel => 'وحدة الوقت';

  @override
  String get periodHours => 'ساعات';

  @override
  String get periodDays => 'أيام';

  @override
  String get periodWeeks => 'أسابيع';

  @override
  String get periodMonths => 'أشهر';

  @override
  String get howOftenAll => 'يُرجى ملء الحقول الثلاثة كلها، أو تركها فارغة.';

  @override
  String get asNeeded => 'عند الحاجة';

  @override
  String get medicationActive => 'أتناوله حاليًا';

  @override
  String get medicationStopped => 'توقفت عن تناوله';

  @override
  String doseMg(String value) {
    return '$value ملغ';
  }

  @override
  String doseG(String value) {
    return '$value غ';
  }

  @override
  String doseUg(String value) {
    return '$value مكغ';
  }

  @override
  String doseMl(String value) {
    return '$value مل';
  }

  @override
  String doseIu(String value) {
    return '$value وحدة دولية';
  }

  @override
  String doseTablet(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value قرص',
      many: '$value قرصًا',
      few: '$value أقراص',
      two: '$value قرص',
      one: '$value قرص',
      zero: '$value قرص',
    );
    return '$_temp0';
  }

  @override
  String doseCapsule(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value كبسولة',
      many: '$value كبسولة',
      few: '$value كبسولات',
      two: '$value كبسولة',
      one: '$value كبسولة',
      zero: '$value كبسولة',
    );
    return '$_temp0';
  }

  @override
  String doseDrop(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value قطرة',
      many: '$value قطرة',
      few: '$value قطرات',
      two: '$value قطرة',
      one: '$value قطرة',
      zero: '$value قطرة',
    );
    return '$_temp0';
  }

  @override
  String dosePuff(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value بخة',
      many: '$value بخة',
      few: '$value بخات',
      two: '$value بخة',
      one: '$value بخة',
      zero: '$value بخة',
    );
    return '$_temp0';
  }

  @override
  String dosePatch(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value لصقة',
      many: '$value لصقة',
      few: '$value لصقات',
      two: '$value لصقة',
      one: '$value لصقة',
      zero: '$value لصقة',
    );
    return '$_temp0';
  }

  @override
  String doseLine(String dose) {
    return 'الجرعة: $dose';
  }

  @override
  String howOftenPerHour(int frequency) {
    return '$frequency× في الساعة';
  }

  @override
  String howOftenPerDay(int frequency) {
    return '$frequency× في اليوم';
  }

  @override
  String howOftenPerWeek(int frequency) {
    return '$frequency× في الأسبوع';
  }

  @override
  String howOftenPerMonth(int frequency) {
    return '$frequency× في الشهر';
  }

  @override
  String howOftenEveryHours(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× كل $every ساعة',
      many: '$frequency× كل $every ساعة',
      few: '$frequency× كل $every ساعات',
      two: '$frequency× كل $every ساعة',
      one: '$frequency× كل $every ساعة',
      zero: '$frequency× كل $every ساعة',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryDays(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× كل $every يوم',
      many: '$frequency× كل $every يومًا',
      few: '$frequency× كل $every أيام',
      two: '$frequency× كل $every يوم',
      one: '$frequency× كل $every يوم',
      zero: '$frequency× كل $every يوم',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryWeeks(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× كل $every أسبوع',
      many: '$frequency× كل $every أسبوعًا',
      few: '$frequency× كل $every أسابيع',
      two: '$frequency× كل $every أسبوع',
      one: '$frequency× كل $every أسبوع',
      zero: '$frequency× كل $every أسبوع',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryMonths(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× كل $every شهر',
      many: '$frequency× كل $every شهرًا',
      few: '$frequency× كل $every أشهر',
      two: '$frequency× كل $every شهر',
      one: '$frequency× كل $every شهر',
      zero: '$frequency× كل $every شهر',
    );
    return '$_temp0';
  }

  @override
  String get myHistory => 'سجلي';

  @override
  String get historyEmpty =>
      'سجلك فارغ حتى الآن. تظهر هنا زياراتك ومستنداتك وإجاباتك.';

  @override
  String get historyNoMatch => 'لا يوجد في سجلك ما يطابق عوامل التصفية هذه.';

  @override
  String get historyClinic => 'العيادة';

  @override
  String get historyAllClinics => 'كل العيادات';

  @override
  String get historyVisits => 'الزيارات';

  @override
  String get historyVisit => 'زيارة';

  @override
  String get historyAnswers => 'إجابات الاستبيان';

  @override
  String get historyFiles => 'الملفات';

  @override
  String get historyFile => 'ملف';

  @override
  String get historyAnyDate => 'أي تاريخ';

  @override
  String historyDateRange(String from, String to) {
    return 'من $from إلى $to';
  }

  @override
  String get historyClearDates => 'مسح التواريخ';

  @override
  String get historyShowMore => 'عرض المزيد';

  @override
  String get historyMoreFailed => 'تعذر تحميل المزيد من سجلك.';

  @override
  String get historyOpenFailed => 'تعذر فتح هذا العنصر. حاول مرة أخرى.';

  @override
  String get sharingToggle => 'مشاركة سجلّي من العيادات الأخرى مع هذه العيادة';

  @override
  String sharingIntro(String clinic) {
    return 'حدّد الزيارات السابقة التي يُسمح لـ $clinic بالاطلاع عليها. لكل زيارة تحدّدها، يرى أطباء $clinic الذين لديك موعد معهم ما هو معروض هنا، مع مستندات الزيارة وإجابات الاستبيانات والملفات، والحساسيات والأدوية المسجّلة فيها، وكل ما يُضاف إلى الزيارة لاحقًا. أما الزيارات التي لا تحدّدها فتبقى مخفية عن $clinic.';
  }

  @override
  String get sharingChangeLater =>
      'يمكنك إضافة زيارات أو إزالتها، أو إيقاف المشاركة، في أي وقت من قسم السجل المشترك. يسري كل تغيير فورًا.';

  @override
  String get sharingNoVisits =>
      'ليست لديك زيارات سابقة في عيادات أخرى يمكن مشاركتها.';

  @override
  String get sharingSelectAll => 'تحديد الكل';

  @override
  String get sharingSelectNone => 'إلغاء تحديد الكل';

  @override
  String get sharedHistory => 'السجل المشترك';

  @override
  String get sharedHistoryIntro =>
      'العيادات التي منحتها موافقتك، وعدد زياراتك من العيادات الأخرى التي تطّلع عليها كل منها.';

  @override
  String get sharedHistoryNone => 'لم تمنح موافقتك لأي عيادة بعد.';

  @override
  String sharedVisits(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count زيارة مُشارَكة',
      many: '$count زيارة مُشارَكة',
      few: '$count زيارات مُشارَكة',
      two: 'زيارتان مُشارَكتان',
      one: 'زيارة واحدة مُشارَكة',
      zero: 'لا شيء مُشارَك',
    );
    return '$_temp0';
  }

  @override
  String get sharingSaved => 'تم حفظ اختيارك ويسري فورًا.';

  @override
  String get sharingStop => 'إيقاف المشاركة';

  @override
  String get sharingStopTitle => 'هل تريد إيقاف مشاركة سجلّك؟';

  @override
  String sharingStopBody(String clinic) {
    return 'لن تعود $clinic قادرة على الاطلاع على أي من الزيارات من العيادات الأخرى التي شاركتها معها. يسري ذلك فورًا.';
  }

  @override
  String get sharingKeep => 'متابعة المشاركة';

  @override
  String get sharingStopped => 'لم تعد تشارك سجلّك مع هذه العيادة.';

  @override
  String get sharingChanged =>
      'تغيّرت زياراتك المُشارَكة في هذه الأثناء. راجعها واحفظها مرة أخرى.';

  @override
  String get bookedNotShared =>
      'تم حجز موعدك، لكن تعذّرت مشاركة سجلّك. يمكنك مشاركته من قسم السجل المشترك.';

  @override
  String get consentForHistory => 'زيارات مُشارَكة من عيادات أخرى';

  @override
  String get reminders => 'التذكيرات';

  @override
  String get remindersIntro => 'اختر كيف تذكّرك العيادات بمواعيدك.';

  @override
  String get remindersPush => 'الإشعارات الفورية';

  @override
  String get remindersEmail => 'البريد الإلكتروني';

  @override
  String get remindersSms => 'الرسائل القصيرة';

  @override
  String get remindersSmsHint =>
      'التذكيرات وعروض قائمة الانتظار برسالة قصيرة إلى رقم الهاتف في ملفك الشخصي، فقط لدى العيادات التي تقدم الرسائل القصيرة. لا تُستخدم أبدًا للإعلانات.';

  @override
  String get quietHours => 'ساعات الهدوء';

  @override
  String get quietHoursHint =>
      'تنتظر الإشعارات الفورية والرسائل القصيرة حتى تنتهي ساعات الهدوء. وإذا كانت ستصل حينها قبل الموعد بأقل من 30 دقيقة، فلا تُرسل. لا يتم تأخير رسائل البريد الإلكتروني.';

  @override
  String get quietHoursFrom => 'من';

  @override
  String get quietHoursUntil => 'إلى';

  @override
  String get remindersSaved => 'تم حفظ اختياراتك.';

  @override
  String get remindersPhoneRequired =>
      'تتطلب الرسائل القصيرة رقم هاتف في ملفك الشخصي. أضفه في ملفي الشخصي، ثم حاول مرة أخرى.';

  @override
  String get remindersInvalid =>
      'تعذّر حفظ هذه الإعدادات. تحقق منها وحاول مرة أخرى.';

  @override
  String get confirmAttendance => 'تأكيد الحضور';

  @override
  String get appointmentConfirmedStatus => 'مؤكد';

  @override
  String get appointmentConfirmedNotice => 'تم تأكيد موعدك.';

  @override
  String get changeTime => 'تغيير الوقت';

  @override
  String rescheduleUntil(String date, String time) {
    return 'يمكن تغيير الوقت حتى $date الساعة $time.';
  }

  @override
  String reschedulesLeft(int count) {
    return 'عدد مرات تغيير الوقت المتبقية: $count';
  }

  @override
  String get rescheduleTitle => 'اختر وقتًا جديدًا';

  @override
  String rescheduleCurrent(String date, String time) {
    return 'المحجوز حاليًا: $date الساعة $time';
  }

  @override
  String get rescheduleConfirmTitle => 'نقل الموعد؟';

  @override
  String rescheduleConfirmBody(String date, String time) {
    return 'سيُنقل موعدك إلى $date الساعة $time. وسيُتاح وقتك الحالي لغيرك.';
  }

  @override
  String get moveAppointment => 'نقل الموعد';

  @override
  String get appointmentMoved => 'تم نقل موعدك إلى الوقت الجديد.';

  @override
  String get rescheduleWindowClosed =>
      'لم يعد من الممكن تغيير وقت هذا الموعد عبر التطبيق. يرجى التواصل مع العيادة.';

  @override
  String get rescheduleLimit =>
      'تم تغيير وقت هذا الموعد بالعدد الذي تسمح به العيادة.';

  @override
  String get notReschedulable => 'لا يمكن تغيير وقت هذا الموعد عبر التطبيق.';

  @override
  String get waitlistTitle => 'قائمة الانتظار الخاصة بي';

  @override
  String get waitlistJoinButton =>
      'لا يوجد موعد مناسب؟ انضم إلى قائمة الانتظار';

  @override
  String get waitlistJoinTitle => 'الانضمام إلى قائمة الانتظار';

  @override
  String get waitlistExplained =>
      'عندما يصبح موعد لدى هذا الطبيب متاحًا، يُعرض على المرضى المنتظرين واحدًا تلو الآخر. سيصلك إشعار ويمكنك حجزه في التطبيق ما دام محجوزًا لك.';

  @override
  String get waitlistHowLong => 'لأي مدة؟';

  @override
  String waitlistDays(int days) {
    String _temp0 = intl.Intl.pluralLogic(
      days,
      locale: localeName,
      other: '$days يوم',
      many: '$days يومًا',
      few: '$days أيام',
      two: 'يومان',
      one: 'يوم واحد',
      zero: '$days يوم',
    );
    return '$_temp0';
  }

  @override
  String get waitlistWhen => 'في أي وقت من اليوم؟';

  @override
  String get waitlistAnyTime => 'في أي وقت';

  @override
  String get waitlistMornings => 'صباحًا، من 08:00 إلى 12:00';

  @override
  String get waitlistAfternoons => 'بعد الظهر، من 12:00 إلى 17:00';

  @override
  String get waitlistJoin => 'انضم إلى قائمة الانتظار';

  @override
  String get waitlistJoined => 'أنت في قائمة الانتظار.';

  @override
  String get waitlistEmpty => 'لست في أي قائمة انتظار.';

  @override
  String get waitlistAPractitioner => 'طبيب';

  @override
  String waitlistUntil(String date) {
    return 'تنتظر حتى $date';
  }

  @override
  String get waitlistOfferTitle => 'أصبح موعد متاحًا لك';

  @override
  String waitlistHeldUntil(String time) {
    return 'محجوز لك حتى $time';
  }

  @override
  String get waitlistAcceptEffect =>
      'بالقبول تحجز هذا الموعد. وعندما يصبح الدفع متاحًا في Salus، ستدفع ثمنه أيضًا عند القبول.';

  @override
  String get waitlistAccept => 'اقبل واحجز';

  @override
  String get waitlistDecline => 'ارفض';

  @override
  String get waitlistBooked => 'تم حجز موعدك. ستجده في مواعيدي.';

  @override
  String get waitlistDeclined => 'رفضت الموعد. تبقى في قائمة الانتظار.';

  @override
  String get waitlistWithdraw => 'مغادرة قائمة الانتظار';

  @override
  String get waitlistWithdrawn => 'غادرت قائمة الانتظار.';

  @override
  String get waitlistOff => 'لا توجد قائمة انتظار لدى هذه العيادة.';

  @override
  String get waitlistTooMany =>
      'أنت مسجل بالفعل خمس مرات في قائمة انتظار هذه العيادة. غادر إحداها أولًا.';

  @override
  String get waitlistOfferExpired =>
      'لم يعد هذا الموعد محجوزًا لك. تبقى في قائمة الانتظار.';
}
