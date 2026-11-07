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
    return 'لن تعود $clinic قادرة على الاطلاع على ملفك الشخصي. وتحتفظ باسمك وبتفاصيل المواعيد التي تمت بالفعل، بوصفها جزءًا من سجلات الرعاية لديها. وستُلغى مواعيدك القادمة لدى $clinic.';
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
      'للانضمام إلى استشارة الفيديو، يحتاج Salus إلى الكاميرا والميكروفون. لا يتم تسجيل الاستشارة.';

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
}
