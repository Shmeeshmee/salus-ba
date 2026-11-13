// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Turkish (`tr`).
class AppLocalizationsTr extends AppLocalizations {
  AppLocalizationsTr([String locale = 'tr']) : super(locale);

  @override
  String get tryAgain => 'Tekrar dene';

  @override
  String get language => 'Dil';

  @override
  String get useDeviceLanguage => 'Cihazın dilini kullan';

  @override
  String get languageSaved => 'Dil kaydedildi.';

  @override
  String get somethingWrong => 'Bir sorun oluştu. Tekrar deneyin.';

  @override
  String get unreachable =>
      'Salus\'a ulaşılamadı. İnternet bağlantınızı kontrol edip tekrar deneyin.';

  @override
  String get wrongCredentials => 'E-posta adresi veya şifre yanlış.';

  @override
  String get wrongCode =>
      'Kod yanlış. Cihazınızdaki saati kontrol edip tekrar deneyin.';

  @override
  String get challengeEnded =>
      'Oturum açma işleminin süresi doldu. E-posta adresinizi ve şifrenizi yeniden girin.';

  @override
  String get emailTaken =>
      'Bu e-posta adresiyle zaten bir hesap var. Bunun yerine oturum açın.';

  @override
  String get invalidEmail => 'Geçerli bir e-posta adresi girin.';

  @override
  String get invalidPassword => 'Şifre 12 ile 128 karakter arasında olmalıdır.';

  @override
  String get missingPassword => 'Şifrenizi girin.';

  @override
  String get invalidCode => 'Kod 6 hanelidir.';

  @override
  String get staffOnly =>
      'Bu uygulama hastalar içindir. Klinik personeli Salus web uygulamasını kullanır.';

  @override
  String get accountReady => 'Hesabınız hazır. Devam etmek için oturum açın.';

  @override
  String get signOutFailed =>
      'Oturum kapatılamadı. İnternet bağlantınızı kontrol edip tekrar deneyin.';

  @override
  String get tooManyAttemptsLater =>
      'Çok fazla deneme yapıldı. Daha sonra tekrar deneyin.';

  @override
  String tooManyAttempts(int minutes) {
    String _temp0 = intl.Intl.pluralLogic(
      minutes,
      locale: localeName,
      other: 'Çok fazla deneme yapıldı. $minutes dakika sonra tekrar deneyin.',
      one: 'Çok fazla deneme yapıldı. $minutes dakika sonra tekrar deneyin.',
    );
    return '$_temp0';
  }

  @override
  String get signInTitle => 'Salus\'ta oturum açın';

  @override
  String get email => 'E-posta';

  @override
  String get password => 'Şifre';

  @override
  String get signIn => 'Oturum aç';

  @override
  String get createAnAccount => 'Hesap oluştur';

  @override
  String get codePrompt =>
      'Kimlik doğrulama uygulamanızdaki 6 haneli kodu girin.';

  @override
  String get code => 'Kod';

  @override
  String get useAnotherAccount => 'Başka bir hesap kullan';

  @override
  String get passwordHint => '12 ile 128 karakter';

  @override
  String get createAccount => 'Hesabı oluştur';

  @override
  String get haveAccount => 'Zaten bir hesabım var';

  @override
  String signedInAs(String email) {
    return 'Oturum açan hesap: $email';
  }

  @override
  String get completeProfile => 'Profilinizi tamamlayın';

  @override
  String get completeProfileFirst =>
      'Randevu almadan önce profilinizi tamamlayın.';

  @override
  String get bookAppointment => 'Randevu al';

  @override
  String get myAppointments => 'Randevularım';

  @override
  String get myConsents => 'Onaylarım';

  @override
  String get myProfile => 'Profilim';

  @override
  String get signOut => 'Oturumu kapat';

  @override
  String get profileIntro =>
      'Randevu aldığınız klinikler bu bilgileri görür. Telefon numarası isteğe bağlıdır; sağlık uzmanı bunu yalnızca bir randevuyla ilgili olarak size ulaşmak için kullanır.';

  @override
  String get givenName => 'Ad';

  @override
  String get familyName => 'Soyadı';

  @override
  String get dateOfBirth => 'Doğum tarihi';

  @override
  String get dateFormatHint => 'YYYY-AA-GG';

  @override
  String get phoneOptional => 'Telefon (isteğe bağlı)';

  @override
  String phoneHint(String example) {
    return 'Ülke koduyla birlikte, örneğin $example';
  }

  @override
  String get save => 'Kaydet';

  @override
  String get notNow => 'Şimdi değil';

  @override
  String get invalidGivenName => 'Adınızı girin (en fazla 100 karakter).';

  @override
  String get invalidFamilyName => 'Soyadınızı girin (en fazla 100 karakter).';

  @override
  String get invalidName =>
      'Adınızı ve soyadınızı girin (her biri en fazla 100 karakter).';

  @override
  String get birthDateFormat => 'Doğum tarihinizi YYYY-AA-GG biçiminde girin.';

  @override
  String get birthDateRange =>
      '1900 ile bugün arasında bir doğum tarihi girin.';

  @override
  String invalidPhone(String example) {
    return 'Telefon numarasını ülke koduyla birlikte girin, örneğin $example.';
  }

  @override
  String get profileChanged =>
      'Profiliniz bu arada değiştirildi. Kontrol edip yeniden kaydedin.';

  @override
  String get chooseClinic => 'Klinik seçin';

  @override
  String get noClinics => 'Henüz listelenmiş bir klinik yok.';

  @override
  String get noPractitioners =>
      'Bu klinikte henüz listelenmiş bir sağlık uzmanı yok.';

  @override
  String get choosePractitioner => 'Bir sağlık uzmanı seçin.';

  @override
  String get noOpenTimes => 'Bu sağlık uzmanının henüz uygun saati yok.';

  @override
  String timesIn(String zone) {
    return 'Saatler $zone saat dilimine göredir.';
  }

  @override
  String get previousDay => 'Önceki gün';

  @override
  String get nextDay => 'Sonraki gün';

  @override
  String get noFreeTimes => 'Bu gün için uygun saat yok.';

  @override
  String get bookThisTime => 'Bu saate randevu al';

  @override
  String get chooseAnotherTime => 'Başka bir saat seç';

  @override
  String get book => 'Randevu al';

  @override
  String timeRange(String start, String end, String zone) {
    return '$start–$end ($zone)';
  }

  @override
  String get slotTaken => 'Bu saat artık uygun değil. Başka bir saat seçin.';

  @override
  String get consentNeeded =>
      'Bu klinikten randevu almak için onayınız gerekiyor.';

  @override
  String get appointmentBooked => 'Randevunuz alındı.';

  @override
  String get unlistedClinic => 'Artık listede olmayan bir klinik';

  @override
  String get cancelTitle => 'Bu randevu iptal edilsin mi?';

  @override
  String cancelBody(String date, String time) {
    return '$date günü saat $time için olan randevunuz iptal edilecek. İptali onaylayan bir e-posta alacaksınız.';
  }

  @override
  String get keepIt => 'Randevuyu koru';

  @override
  String get cancelAppointment => 'Randevuyu iptal et';

  @override
  String get noAppointments => 'Henüz randevunuz yok.';

  @override
  String get upcoming => 'Yaklaşan';

  @override
  String get past => 'Geçmiş';

  @override
  String get statusBooked => 'Planlandı';

  @override
  String get statusCancelled => 'İptal edildi';

  @override
  String get statusAttended => 'Katılım sağlandı';

  @override
  String get statusNotAttended => 'Katılım sağlanmadı';

  @override
  String get appointmentCancelled => 'Randevunuz iptal edildi.';

  @override
  String get cancelWindowHint =>
      'Başlangıçtan 24 saat öncesine kadar uygulamadan iptal edilebilir.';

  @override
  String get cancelWindowClosed =>
      'Bu randevunun başlamasına 24 saatten az kaldığı için artık uygulamadan iptal edilemez.';

  @override
  String get appointmentClosed => 'Bu randevu artık geçerli değil.';

  @override
  String consentTitle(String clinic) {
    return 'Bilgilerinizi $clinic ile paylaşın';
  }

  @override
  String consentIntro(String clinic) {
    return '$clinic ile ilk randevunuzu almadan önce, bilgilerinizi bu klinikle paylaşmak için onayınıza ihtiyacımız var.';
  }

  @override
  String get whatIsShared => 'Neler paylaşılır';

  @override
  String get whatIsSharedBody =>
      'Profiliniz (adınız, doğum tarihiniz ve verdiyseniz telefon numaranız) ve bu klinikteki randevularınız.';

  @override
  String get whoSeesIt => 'Kimler görür';

  @override
  String whoSeesItBody(String clinic) {
    return '$clinic. Randevu aldığınız sağlık uzmanı profilinizi görür; kliniğin yöneticileri adınızı ve randevu saatlerinizi görür.';
  }

  @override
  String get whatFor => 'Ne amaçla';

  @override
  String get whatForBody =>
      'Yalnızca bu klinikte size sağlık hizmeti sunmak için: randevularınızı hazırlamak ve gerçekleştirmek amacıyla.';

  @override
  String get whoIsResponsible => 'Sorumlu kim';

  @override
  String whoIsResponsibleBody(String clinic) {
    return 'Verilerinizden $clinic sorumludur. Salus verilerinizi klinik adına işler.';
  }

  @override
  String get readPrivacyNotice => 'Gizlilik bildirimini okuyun';

  @override
  String get privacyNoticeFailed => 'Gizlilik bildirimi açılamadı.';

  @override
  String get howToRevoke => 'Onay nasıl geri alınır';

  @override
  String get howToRevokeBody =>
      'Bu onayı istediğiniz zaman Onaylarım bölümünden geri alabilirsiniz. Bu durumda klinik profilinizi artık göremez. Klinik, sağlık kayıtlarının bir parçası olarak adınızı ve gerçekleşmiş randevularınızın bilgilerini saklar. Bu klinikteki gelecek randevularınız iptal edilir.';

  @override
  String get agreeAndBook => 'Onayla ve randevu al';

  @override
  String get revokeTitle => 'Onayınızı geri almak istiyor musunuz?';

  @override
  String revokeBody(String clinic) {
    return '$clinic artık profilinizi göremeyecek. Klinik, sağlık kayıtlarının bir parçası olarak adınızı ve gerçekleşmiş randevularınızın bilgilerini saklar. $clinic ile olan gelecek randevularınız iptal edilecek. Diğer kliniklerdeki geçmişinizin $clinic ile paylaşımı da sona erer; yeniden onay vermeniz bu paylaşımı geri getirmez.';
  }

  @override
  String get keepConsent => 'Onayı koru';

  @override
  String get revokeConsent => 'Onayı geri al';

  @override
  String get noConsents => 'Henüz hiçbir onay vermediniz.';

  @override
  String get consentsIntro =>
      'Profilinizi ve randevularınızı, sağlık hizmeti almak için bu kliniklerle paylaşıyorsunuz.';

  @override
  String givenOn(String date) {
    return 'Verildiği tarih: $date';
  }

  @override
  String get consentActive => 'Etkin';

  @override
  String revokedOn(String date) {
    return 'Geri alındığı tarih: $date';
  }

  @override
  String get consentRevoked =>
      'Onayınız geri alındı. Bu klinikteki gelecek randevularınız iptal edilecek ve her biri için bir e-posta alacaksınız.';

  @override
  String get consentAlreadyRevoked => 'Bu onay zaten geri alınmış.';

  @override
  String get joinConsultation => 'Görüşmeye katıl';

  @override
  String get joinFrom => 'Başlangıçtan 10 dakika önce katılabilirsiniz.';

  @override
  String get checkTitle => 'Kameranızı ve mikrofonunuzu kontrol edin';

  @override
  String get checkIntro =>
      'Katıldığınızda sağlık uzmanı sizi görüp duyabilecek.';

  @override
  String get noSound =>
      'Ses algılanmadı. Mikrofonunuzun açık olduğunu ve sesinin kapatılmadığını kontrol edin.';

  @override
  String get joinCameraOff => 'Kamera kapalı katıl';

  @override
  String get joinMicrophoneOff => 'Mikrofon kapalı katıl';

  @override
  String get joinNow => 'Katıl';

  @override
  String get joining => 'Bağlanılıyor…';

  @override
  String get cameraDenied =>
      'Salus\'un kameranıza erişimi yok. Cihaz ayarlarınızdan izin verebilir veya görüntü olmadan katılabilirsiniz.';

  @override
  String get microphoneDenied =>
      'Salus\'un mikrofonunuza erişimi yok. Cihaz ayarlarınızdan izin verebilirsiniz. Mikrofon olmadan sağlık uzmanı sizi duyamaz.';

  @override
  String get openSettings => 'Ayarları aç';

  @override
  String get noCamera => 'Kamera bulunamadı.';

  @override
  String get noMicrophone => 'Mikrofon bulunamadı.';

  @override
  String get waiting => 'Sağlık uzmanı bekleniyor';

  @override
  String joinsYouHere(String name) {
    return '$name birazdan burada size katılacak. Bu ekranı açık tutun.';
  }

  @override
  String get mute => 'Mikrofonu kapat';

  @override
  String get unmute => 'Mikrofonu aç';

  @override
  String get cameraOff => 'Kamerayı kapat';

  @override
  String get cameraOn => 'Kamerayı aç';

  @override
  String get leave => 'Ayrıl';

  @override
  String peerCameraOff(String name) {
    return '$name: kamera kapalı';
  }

  @override
  String peerMuted(String name) {
    return '$name: mikrofon kapalı';
  }

  @override
  String peerLeft(String name) {
    return '$name görüşmeden ayrıldı. Görüşme bitene kadar yeniden katılabilir.';
  }

  @override
  String get reconnecting => 'Bağlantı kesildi. Yeniden bağlanılıyor…';

  @override
  String get notRestored => 'Bağlantı yeniden kurulamadı.';

  @override
  String get reconnected => 'Yeniden bağlanıldı';

  @override
  String get consultationOver => 'Görüşme sona erdi';

  @override
  String get closeScreen => 'Bu ekranı kapatabilirsiniz.';

  @override
  String get timeLimit => 'Görüşme süre sınırına ulaştı ve sona erdi.';

  @override
  String get practitionerAbsent => 'Sağlık uzmanı katılmadı';

  @override
  String get timePassed =>
      'Bu görüşmeye katılma süresi geçti. Klinik, katıldığınızı görebilir. Klinik sizinle iletişime geçebilir ya da yeni bir randevu alabilirsiniz.';

  @override
  String get withdrewConsent =>
      'Bu klinik için onayınızı geri aldığınızdan görüşmeden ayrıldınız.';

  @override
  String get appointmentWasCancelled =>
      'Bu randevu iptal edildiği için görüşme sona erdi.';

  @override
  String get windowNotOpen => 'Bu görüşme, başlangıçtan 10 dakika önce açılır.';

  @override
  String get consultationEnded => 'Bu görüşme sona erdi.';

  @override
  String get joinNeedsConsent =>
      'Görüşmeye katılmak için bu kliniğe onay vermeniz gerekiyor.';

  @override
  String get consultationNotAvailable => 'Bu görüşmeye erişilemiyor.';

  @override
  String get cannotConnect =>
      'Görüntülü bağlantı başlatılamadı. İnternet bağlantınızı kontrol edip tekrar deneyin.';

  @override
  String get whyCameraAndMicrophone =>
      'Görüntülü görüşmeye katılmak için Salus\'un kameranıza ve mikrofonunuza erişmesi gerekir. Görüşme ses veya görüntü olarak kaydedilmez. Onay verirseniz söylenenler yazıya dökülür ve çevrilir.';

  @override
  String get transcriptionOn => 'Transkripsiyon açık';

  @override
  String get transcriptionTitle => 'Transkripsiyon ve çeviri';

  @override
  String transcriptionIntro(String clinic) {
    return '$clinic ile görüntülü görüşmelerinizde sizin ve sağlık uzmanının söyledikleri konuşurken yazıya dökülebilir ve çevrilebilir; böylece sağlık uzmanı sizi kendi dilinizde takip edebilir.';
  }

  @override
  String get transcriptionAssistance =>
      'Sağlık uzmanı ayrıca yazıya dökülen metinden yapay zekâ desteği alır: görüşme sırasında öneriler ve belge taslakları. Bunlar yalnızca önerilerdir; sağlık uzmanı bunları kontrol eder ve her kararı kendisi verir.';

  @override
  String transcriptionKept(String clinic) {
    return 'Yazıya dökülen metin, sağlık kaydınızla birlikte $clinic tarafından saklanır. Görüşme ses veya görüntü olarak kaydedilmez.';
  }

  @override
  String get transcriptionDecline =>
      'Onay vermezseniz görüşme her zamanki gibi; transkripsiyon, çeviri ve yapay zekâ desteği olmadan yapılır.';

  @override
  String get transcriptionWithdraw =>
      'Onayınızı istediğiniz zaman, görüşme sırasında da geri alabilirsiniz: “Transkripsiyon açık” yazısına dokunun veya Onaylarım bölümünü açın. Transkripsiyon hemen durur; o ana kadar yazıya dökülenler kaydınızda kalır.';

  @override
  String get transcriptionAgree => 'Onayla ve devam et';

  @override
  String get transcriptionWithout => 'Transkripsiyon olmadan devam et';

  @override
  String get stopTranscriptionTitle => 'Transkripsiyon durdurulsun mu?';

  @override
  String stopTranscriptionBody(String clinic) {
    return '$clinic ile görüşmelerinizde transkripsiyon, çeviri ve yapay zekâ desteği hemen durur. Devam eden bir görüşme bunlar olmadan sürer ve randevularınız olduğu gibi kalır. Şimdiye kadar yazıya dökülenler kaydınızda kalır.';
  }

  @override
  String get keepTranscription => 'Transkripsiyonu sürdür';

  @override
  String get stopTranscription => 'Transkripsiyonu durdur';

  @override
  String get transcriptionStopped =>
      'Transkripsiyon kapalı. Bu klinikle görüşmeleriniz transkripsiyon olmadan yapılır.';

  @override
  String get consentForCare => 'Bakım için paylaşım';

  @override
  String get consentForTranscription => 'Transkripsiyon ve yapay zekâ desteği';

  @override
  String get toFillIn => 'Doldurulacak';

  @override
  String get inProgress => 'Devam ediyor';

  @override
  String get submittedStatus => 'Gönderildi';

  @override
  String get closedStatus => 'Kapandı';

  @override
  String get fillIn => 'Doldur';

  @override
  String get continueFilling => 'Devam et';

  @override
  String get viewAnswers => 'Yanıtları görüntüle';

  @override
  String get submitBeforeStart => 'Lütfen randevunuz başlamadan önce gönderin.';

  @override
  String get questionnaireClosed =>
      'Randevu başladığı için bu soru formu artık doldurulamaz.';

  @override
  String savedAsYouGo(String clinic) {
    return 'Yanıtlarınız siz ilerledikçe kaydedilir. $clinic yanıtları ancak siz gönderdikten sonra görür.';
  }

  @override
  String get saved => 'Kaydedildi';

  @override
  String get saving => 'Kaydediliyor…';

  @override
  String get notSaved =>
      'Kaydedilmedi. İnternet bağlantınızı kontrol edin. Kaydetme yeniden denenecek.';

  @override
  String get requiredQuestion => 'Zorunlu';

  @override
  String get moreInformation => 'Daha fazla bilgi';

  @override
  String get yourAnswer => 'Yanıtınız';

  @override
  String get chooseDate => 'Tarih seçin';

  @override
  String get selectOne => 'Birini seçin.';

  @override
  String get selectAll => 'Uygun olanların tümünü seçin.';

  @override
  String get yes => 'Evet';

  @override
  String get no => 'Hayır';

  @override
  String get questionRequired => 'Bu soru zorunludur.';

  @override
  String get enterWholeNumber => 'Bir tam sayı girin.';

  @override
  String get enterNumber => 'Bir sayı girin.';

  @override
  String get numberTooLarge => 'Bu sayı çok büyük.';

  @override
  String upToCharacters(int max) {
    return 'En fazla $max karakter girin.';
  }

  @override
  String get enterValidDate => 'Geçerli bir tarih girin.';

  @override
  String get submit => 'Gönder';

  @override
  String get submitTitle => 'Yanıtlarınız gönderilsin mi?';

  @override
  String submitBody(String clinic) {
    return 'Gönderdikten sonra yanıtlarınızı değiştiremezsiniz. $clinic bünyesindeki sağlık uzmanınız bunları görecek.';
  }

  @override
  String get cancel => 'İptal';

  @override
  String get requiredMissing => 'Bazı zorunlu sorular yanıtlanmadı.';

  @override
  String get answersSubmitted => 'Yanıtlar gönderildi';

  @override
  String get answersSubmittedBody =>
      'Sağlık uzmanınız bunları randevunuzdan önce görebilir. Yanıtlarınızı Randevularım bölümünden görüntüleyebilirsiniz.';

  @override
  String get yourAnswers => 'Yanıtlarınız';

  @override
  String submittedAt(String when) {
    return 'Gönderilme: $when';
  }

  @override
  String dateAndTime(String date, String time) {
    return '$date, $time';
  }

  @override
  String get notAnswered => 'Yanıtlanmadı';

  @override
  String get leaveUnsaved =>
      'Son yanıtınız henüz kaydedilmedi. Yine de ayrılmak istiyor musunuz?';

  @override
  String get stay => 'Kal';

  @override
  String get fillingNeedsConsent =>
      'Bu soru formunu doldurmak için bu kliniğe onay vermeniz gerekiyor.';

  @override
  String get pastDue =>
      'Randevu başladığı için bu soru formu artık değiştirilemez.';

  @override
  String get withdrawnByClinic =>
      'Klinik bu soru formunu geri çekti. Doldurmanız gerekmiyor.';

  @override
  String get alreadySubmitted => 'Bu soru formunu zaten gönderdiniz.';

  @override
  String get changedElsewhere =>
      'Bu yanıtlar başka bir cihazda değiştirildi. En güncel yanıtlar gösteriliyor.';

  @override
  String get questionnaireNotAvailable => 'Bu soru formuna erişilemiyor.';

  @override
  String get done => 'Tamam';

  @override
  String get addFile => 'Dosya ekle';

  @override
  String get photosUpTo => 'Fotoğraflar en fazla 20 MB olabilir.';

  @override
  String get documentsUpTo => 'Belgeler (PDF) en fazla 20 MB olabilir.';

  @override
  String get videosUpTo => 'Videolar en fazla 200 MB olabilir.';

  @override
  String get oneFileHint => '1 dosya.';

  @override
  String upToFilesHint(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'En fazla $n dosya.',
      one: 'En fazla $n dosya.',
    );
    return '$_temp0';
  }

  @override
  String get takePhoto => 'Fotoğraf çek';

  @override
  String get recordVideo => 'Video kaydet';

  @override
  String get chooseFromPhotos => 'Fotoğraflardan seç';

  @override
  String get chooseDocument => 'Belge seç';

  @override
  String get preparing => 'Hazırlanıyor…';

  @override
  String uploadingPercent(int n) {
    return 'Yükleniyor… %$n';
  }

  @override
  String get checkingFile => 'Dosya kontrol ediliyor…';

  @override
  String get added => 'Eklendi';

  @override
  String get notUploaded => 'Yüklenmedi. İnternet bağlantınızı kontrol edin.';

  @override
  String get remove => 'Kaldır';

  @override
  String get removeFile => 'Dosyayı kaldır';

  @override
  String get removeFileTitle => 'Bu dosya kaldırılsın mı?';

  @override
  String get removeFileBody => 'Dosya yanıtlarınızdan silinir.';

  @override
  String get fileRejected =>
      'Bu dosya güvenlik kontrolünden geçemedi ve silindi. Başka bir dosya seçin.';

  @override
  String fileTooLarge(int max) {
    return 'Bu dosyanın boyutu $max MB sınırını aşıyor.';
  }

  @override
  String get fileTypeRefused => 'Bu türde bir dosya bu soruya eklenemez.';

  @override
  String tooManyFiles(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Bu soruya en fazla $n dosya ekleyebilirsiniz.',
      one: 'Bu soruya en fazla $n dosya ekleyebilirsiniz.',
    );
    return '$_temp0';
  }

  @override
  String get filesFull =>
      'Bu soru formundaki dosyalar 500 MB sınırına ulaştı. Başka bir dosya eklemek için bir dosyayı kaldırın.';

  @override
  String get fileNotUploaded => 'Dosya yüklenemedi. Tekrar deneyin.';

  @override
  String get cannotPrepareFile =>
      'Bu dosya yüklemeye hazırlanamadı. Başka bir dosya seçin.';

  @override
  String get filesBeingChecked =>
      'Bazı dosyalar hâlâ kontrol ediliyor. Hazır olduklarında gönderebilirsiniz.';

  @override
  String get leaveUploading =>
      'Bir dosya hâlâ yükleniyor. Yine de ayrılmak istiyor musunuz? Yükleme durur.';

  @override
  String fileCount(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n dosya',
      one: '$n dosya',
    );
    return '$_temp0';
  }

  @override
  String get photoKind => 'Fotoğraf';

  @override
  String get documentKind => 'Belge';

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
    return '$title yükleniyor, yüzde $n';
  }

  @override
  String fileLabel(String kind, String title, String status) {
    return '$kind, $title, $status';
  }

  @override
  String get allowCamera => 'Kamera erişimine izin verin';

  @override
  String get whyCameraForFiles =>
      'Bu soru formuna ekleyeceğiniz fotoğrafı veya videoyu çekmek için Salus\'un kameraya erişmesi gerekir.';

  @override
  String get continueLabel => 'Devam';

  @override
  String get cameraDeniedForFiles =>
      'Salus\'un kameranıza erişimi yok. Cihaz ayarlarınızdan izin verebilir veya bunun yerine bir dosya seçebilirsiniz.';

  @override
  String get photosDenied =>
      'Salus\'un fotoğraflarınıza erişimi yok. Cihaz ayarlarınızdan izin verebilirsiniz.';

  @override
  String cameraPhotoTitle(String date, String time) {
    return 'Fotoğraf, $date, $time';
  }

  @override
  String cameraVideoTitle(String date, String time) {
    return 'Video, $date, $time';
  }

  @override
  String get specialty394814009 => 'Aile hekimliği';

  @override
  String get specialty419192003 => 'İç hastalıkları';

  @override
  String get specialty394579002 => 'Kardiyoloji';

  @override
  String get specialty394582007 => 'Dermatoloji';

  @override
  String get specialty394586005 => 'Kadın hastalıkları';

  @override
  String get specialty394537008 => 'Çocuk sağlığı ve hastalıkları';

  @override
  String get specialty394587001 => 'Psikiyatri';

  @override
  String get specialty394591006 => 'Nöroloji';

  @override
  String get specialty418960008 => 'Kulak Burun Boğaz (KBB)';

  @override
  String get specialty394594003 => 'Göz hastalıkları';

  @override
  String get specialty394801008 => 'Ortopedi';

  @override
  String get specialty394583002 => 'Endokrinoloji';

  @override
  String questionnaireNotInLanguage(String language) {
    return 'Bu soru formu şu dilde mevcut değil: $language.';
  }

  @override
  String questionnaireLanguages(String languages) {
    return 'Şu dillerde mevcut: $languages. En iyi anladığınız dili seçin.';
  }

  @override
  String continueInLanguage(String language) {
    return 'Şu dilde devam et: $language';
  }

  @override
  String get languageUnavailable =>
      'Bu dil artık bu soru formu için mevcut değil. Başka bir dil seçin.';

  @override
  String specialtyAtClinic(String specialty, String clinic) {
    return '$specialty, $clinic';
  }

  @override
  String get documents => 'Belgeler';

  @override
  String get noDocuments =>
      'Bu randevu için henüz imzalanmış belge yok. Sağlık uzmanınız bir belgeyi imzaladığında belge burada görünür.';

  @override
  String documentSignedOn(String date) {
    return 'İmza tarihi: $date';
  }

  @override
  String documentLanguage(String language) {
    return 'Dil: $language';
  }

  @override
  String get sharePdf => 'PDF\'yi kaydet veya paylaş';

  @override
  String get transcript => 'Görüşme dökümü';

  @override
  String get transcriptNote =>
      'Görüşmenin otomatik olarak yazıya dökülmüş ve çevrilmiş hâli. Hatalar içerebilir.';

  @override
  String get transcriptEmpty => 'Bu görüşmede yazıya dökülen bir şey yok.';

  @override
  String get speakerYou => 'Siz';

  @override
  String get speakerPractitioner => 'Sağlık uzmanı';

  @override
  String get shareTranscript => 'Dökümü paylaş';

  @override
  String transcriptTranslation(String text) {
    return 'Çeviri: $text';
  }

  @override
  String get termExplanationNote => 'Bilgi amaçlıdır, tanı değildir';

  @override
  String get termNotReviewed => 'Henüz incelenmedi';

  @override
  String get termClose => 'Kapat';

  @override
  String get chatsTitle => 'Mesajlar';

  @override
  String get chatsEmpty => 'Henüz bir yazışmanız yok.';

  @override
  String chatUnread(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count okunmamış mesaj',
      one: '$count okunmamış mesaj',
    );
    return '$_temp0';
  }

  @override
  String get chatDoctorUnknown => 'Sağlık uzmanınız';

  @override
  String get chatMessageHint => 'Bir mesaj yazın';

  @override
  String get chatSend => 'Gönder';

  @override
  String get chatAttach => 'Dosya ekle';

  @override
  String get chatSendFailed =>
      'Mesaj gönderilmedi. Bağlantınızı kontrol edip tekrar deneyin.';

  @override
  String get chatFileRejected => 'Bu dosya kabul edilmedi.';

  @override
  String get chatFileRemove => 'Dosyayı kaldır';

  @override
  String chatTooManyFiles(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'Gönderilmeyi bekleyen $count dosyanız zaten var. Gönderilmeyen dosyalar 24 saat sonra kaldırılır; daha sonra tekrar deneyin.',
      one:
          'Gönderilmeyi bekleyen $count dosyanız zaten var. Gönderilmeyen dosyalar 24 saat sonra kaldırılır; daha sonra tekrar deneyin.',
    );
    return '$_temp0';
  }

  @override
  String get chatOpenFailed => 'Dosya açılamadı. Tekrar deneyin.';

  @override
  String get chatEndedSession =>
      'Oturumunuz sona erdi. Yeni mesajları görmek için tekrar giriş yapın.';

  @override
  String get chatEnded => 'Bu yazışmada yeni mesajlar artık gösterilemiyor.';

  @override
  String get chatHours => 'Mesaj saatleri';

  @override
  String chatHoursDay(String day, String start, String end) {
    return '$day: $start–$end';
  }

  @override
  String get chatHoursNone => 'Klinik mesaj saatlerini belirlemedi.';

  @override
  String chatHoursZone(String zone) {
    return 'Saatler kliniğin saat dilimindedir: $zone.';
  }

  @override
  String chatResponseTime(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other: 'Klinik genellikle $hours saat içinde yanıt verir.',
      one: 'Klinik genellikle $hours saat içinde yanıt verir.',
    );
    return '$_temp0';
  }

  @override
  String chatOutOfHours(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other:
          'Klinik şu anda mesaj saatlerinin dışında. Yine de yazabilirsiniz; genellikle $hours saat içinde yanıt verir.',
      one:
          'Klinik şu anda mesaj saatlerinin dışında. Yine de yazabilirsiniz; genellikle $hours saat içinde yanıt verir.',
    );
    return '$_temp0';
  }

  @override
  String chatSystemInvited(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Başka bir klinikten bir sağlık uzmanı ikinci görüş için bu yazışmaya davet edildi.',
      'specialist_review': 'Başka bir klinikten bir sağlık uzmanı uzman değerlendirmesi için bu yazışmaya davet edildi.',
      'handover': 'Başka bir klinikten bir sağlık uzmanı tedaviyi devralmak üzere bu yazışmaya davet edildi.',
      'other':
          'Başka bir klinikten bir sağlık uzmanı bu yazışmaya davet edildi.',
    });
    return '$_temp0';
  }

  @override
  String chatSystemAdded(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion':
          'Bir sağlık uzmanı ikinci görüş için yazışmaya katıldı.',
      'specialist_review':
          'Bir sağlık uzmanı uzman değerlendirmesi için yazışmaya katıldı.',
      'handover':
          'Bir sağlık uzmanı tedaviyi devralmak üzere yazışmaya katıldı.',
      'other': 'Bir sağlık uzmanı yazışmaya katıldı.',
    });
    return '$_temp0';
  }

  @override
  String get chatSystemRemoved =>
      'Bir sağlık uzmanı artık yazışmada yer almıyor.';

  @override
  String get chatSystemGranted =>
      'Başka bir klinikten bir sağlık uzmanının yazışmaya katılmasına izin verdiniz.';

  @override
  String get chatSystemDeclined =>
      'Başka bir klinikten bir sağlık uzmanını reddettiniz.';

  @override
  String get chatSystemOther => 'Yazışma güncellendi.';

  @override
  String get chatInviteTitle =>
      'Başka bir klinikten bir sağlık uzmanı katılmak istiyor';

  @override
  String chatInviteBody(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Kliniğiniz, başka bir klinikten bir sağlık uzmanının ikinci görüş için bu yazışmaya katılmasını istiyor.',
      'specialist_review': 'Kliniğiniz, başka bir klinikten bir sağlık uzmanının uzman değerlendirmesi için bu yazışmaya katılmasını istiyor.',
      'handover': 'Kliniğiniz, başka bir klinikten bir sağlık uzmanının tedavinizi devralmak üzere bu yazışmaya katılmasını istiyor.',
      'other': 'Kliniğiniz, başka bir klinikten bir sağlık uzmanının bu yazışmaya katılmasını istiyor.',
    });
    return '$_temp0';
  }

  @override
  String chatInviteWho(String doctor, String clinic) {
    return 'Sağlık uzmanı: $doctor, $clinic';
  }

  @override
  String get chatInviteUnresolved =>
      'Davet edilen sağlık uzmanının adı ve kliniği yüklenemedi. Erişime ancak bunlar gösterildiğinde izin verebilirsiniz.';

  @override
  String chatInviteSees(String history) {
    String _temp0 = intl.Intl.selectLogic(history, {
      'full': 'Kabul ederseniz sağlık uzmanı, önceki mesajlar dahil yazışmanın tamamını görür ve yazışmaya yazabilir.',
      'from': 'Kabul ederseniz sağlık uzmanı, yazışmayı kliniğinizin seçtiği bir mesajdan itibaren görür ve yazışmaya yazabilir.',
      'other': 'Kabul ederseniz sağlık uzmanı yazışmayı görür ve yazışmaya yazabilir.',
    });
    return '$_temp0';
  }

  @override
  String get chatInviteChoice =>
      'Reddedebilirsiniz. Kabul ederseniz sağlık uzmanının erişimini daha sonra bu yazışmadan geri alabilirsiniz; erişim hemen sona erer.';

  @override
  String get chatGrant => 'Erişim ver';

  @override
  String get chatDecline => 'Reddet';

  @override
  String get chatGranted =>
      'Erişim verildi. İstediğiniz zaman bu yazışmadan geri alabilirsiniz.';

  @override
  String get chatDeclined =>
      'Reddedildi. Sağlık uzmanı bu yazışmaya katılmıyor.';

  @override
  String get chatRevoke =>
      'Diğer klinikteki sağlık uzmanının erişimini geri al';

  @override
  String chatGrantedTo(String doctor, String clinic) {
    return 'Erişim verildi: $doctor, $clinic.';
  }

  @override
  String get chatGrantedToUnknown =>
      'Başka bir klinikteki bir sağlık uzmanına erişim verildi.';

  @override
  String get chatRevokeBody =>
      'Diğer klinikteki sağlık uzmanı bu yazışmaya erişimini hemen kaybeder. Daha önce yazılmış mesajlar yazışmada kalır.';

  @override
  String get chatRevokeConfirm => 'Erişimi geri al';

  @override
  String get chatRevoked => 'Erişim geri alındı.';

  @override
  String get allergies => 'Alerjiler';

  @override
  String get currentMedications => 'Kullandığım ilaçlar';

  @override
  String get allergiesIntro =>
      'Onay verdiğiniz klinikler, onlarla randevunuz olduğunda profilinizdeki alerjileri görür.';

  @override
  String get medicationsIntro =>
      'Onay verdiğiniz klinikler, onlarla randevunuz olduğunda profilinizdeki ilaçları görür.';

  @override
  String get noAllergies => 'Henüz bir alerji eklemediniz.';

  @override
  String get noMedications => 'Henüz bir ilaç eklemediniz.';

  @override
  String get addAllergy => 'Alerji ekle';

  @override
  String get addMedication => 'İlaç ekle';

  @override
  String get changeAllergy => 'Alerjiyi değiştir';

  @override
  String get changeMedication => 'İlacı değiştir';

  @override
  String get change => 'Değiştir';

  @override
  String get allergySubstance => 'Madde veya besin';

  @override
  String get medicationName => 'İlaç';

  @override
  String get termSearchHint =>
      'Aramak için yazın veya kendi kelimelerinizle girin.';

  @override
  String get allergyCategory => 'Tür (isteğe bağlı)';

  @override
  String get notSpecified => 'Belirtilmedi';

  @override
  String get categoryFood => 'Besin';

  @override
  String get categoryMedication => 'İlaç';

  @override
  String get categoryEnvironment => 'Çevre';

  @override
  String get categoryBiologic => 'Biyolojik ürün';

  @override
  String get reactionOptional => 'Reaksiyon (isteğe bağlı)';

  @override
  String get severity => 'Şiddet';

  @override
  String get severityMild => 'Hafif';

  @override
  String get severityModerate => 'Orta';

  @override
  String get severitySevere => 'Şiddetli';

  @override
  String get entryStatus => 'Durum';

  @override
  String get allergyActive => 'Etkin';

  @override
  String get allergyInactive => 'Etkin değil';

  @override
  String get allergyResolved => 'Geçti';

  @override
  String reactionLine(String reaction) {
    return 'Reaksiyon: $reaction';
  }

  @override
  String reactionWithSeverity(String reaction, String severity) {
    return 'Reaksiyon: $reaction ($severity)';
  }

  @override
  String get confirmedByClinician => 'Bir hekim tarafından doğrulandı';

  @override
  String get recordedByClinic =>
      'Ziyaretlerinizden birinde bir klinik tarafından kaydedildi';

  @override
  String get confirmedChangeNote =>
      'Bu kaydı değiştirirseniz artık hekim tarafından doğrulanmış olmaz.';

  @override
  String get nameRequired => 'En fazla 200 karakterlik bir ad girin.';

  @override
  String get addToProfile => 'Profilime ekle';

  @override
  String get addToProfileTitle => 'Profilinize eklensin mi?';

  @override
  String get addToProfileBody =>
      'Bu durumda onay verdiğiniz her klinik bunu, kaydeden kliniğin adı olmadan görür.';

  @override
  String get addedToProfile => 'Artık profilinizde.';

  @override
  String get removeEntryTitle => 'Bu kayıt kaldırılsın mı?';

  @override
  String get removeEntryBody =>
      'Artık size veya herhangi bir kliniğe gösterilmez. Bir klinik kaydettiyse veya doğruladıysa, kaldırıldı olarak işaretlenip tıbbi kaydınızda kalır.';

  @override
  String get entryRemoved => 'Kayıt kaldırıldı.';

  @override
  String get entrySaved => 'Profilinize kaydedildi.';

  @override
  String get entryChanged =>
      'Bu kayıt bu arada değiştirildi. En güncel hâli gösteriliyor.';

  @override
  String get entryInvalid =>
      'Bazı bilgiler geçerli değil. Kontrol edip yeniden deneyin.';

  @override
  String get entryNotAllowed => 'Bu kaydı değiştiremezsiniz.';

  @override
  String get entryNotFound => 'Bu kayda artık erişilemiyor.';

  @override
  String get doseHeading => 'Doz (isteğe bağlı)';

  @override
  String get doseAmount => 'Miktar';

  @override
  String get doseUnit => 'Birim';

  @override
  String get unitMg => 'miligram (mg)';

  @override
  String get unitG => 'gram (g)';

  @override
  String get unitUg => 'mikrogram (µg)';

  @override
  String get unitMl => 'mililitre (ml)';

  @override
  String get unitIu => 'uluslararası birim (IU)';

  @override
  String get unitTablet => 'tablet';

  @override
  String get unitCapsule => 'kapsül';

  @override
  String get unitDrop => 'damla';

  @override
  String get unitPuff => 'püskürtme (inhaler)';

  @override
  String get unitPatch => 'bant';

  @override
  String get doseBoth =>
      'Miktarı ve birimi birlikte girin ya da ikisini de boş bırakın.';

  @override
  String get numberAboveZero => '0\'dan büyük bir sayı girin.';

  @override
  String get howOftenHeading => 'Ne sıklıkla (isteğe bağlı)';

  @override
  String get timesLabel => 'Kez';

  @override
  String get everyLabel => 'Her';

  @override
  String get periodUnitLabel => 'Zaman birimi';

  @override
  String get periodHours => 'saat';

  @override
  String get periodDays => 'gün';

  @override
  String get periodWeeks => 'hafta';

  @override
  String get periodMonths => 'ay';

  @override
  String get howOftenAll => 'Üç alanın hepsini doldurun ya da boş bırakın.';

  @override
  String get asNeeded => 'Gerektiğinde';

  @override
  String get medicationActive => 'Şu anda kullanıyorum';

  @override
  String get medicationStopped => 'Bıraktım';

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
    return '$value IU';
  }

  @override
  String doseTablet(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value tablet',
      one: '$value tablet',
    );
    return '$_temp0';
  }

  @override
  String doseCapsule(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value kapsül',
      one: '$value kapsül',
    );
    return '$_temp0';
  }

  @override
  String doseDrop(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value damla',
      one: '$value damla',
    );
    return '$_temp0';
  }

  @override
  String dosePuff(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value püskürtme',
      one: '$value püskürtme',
    );
    return '$_temp0';
  }

  @override
  String dosePatch(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value bant',
      one: '$value bant',
    );
    return '$_temp0';
  }

  @override
  String doseLine(String dose) {
    return 'Doz: $dose';
  }

  @override
  String howOftenPerHour(int frequency) {
    return 'Saatte $frequency kez';
  }

  @override
  String howOftenPerDay(int frequency) {
    return 'Günde $frequency kez';
  }

  @override
  String howOftenPerWeek(int frequency) {
    return 'Haftada $frequency kez';
  }

  @override
  String howOftenPerMonth(int frequency) {
    return 'Ayda $frequency kez';
  }

  @override
  String howOftenEveryHours(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: 'Her $every saatte $frequency kez',
      one: 'Her $every saatte $frequency kez',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryDays(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: 'Her $every günde $frequency kez',
      one: 'Her $every günde $frequency kez',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryWeeks(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: 'Her $every haftada $frequency kez',
      one: 'Her $every haftada $frequency kez',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryMonths(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: 'Her $every ayda $frequency kez',
      one: 'Her $every ayda $frequency kez',
    );
    return '$_temp0';
  }

  @override
  String get myHistory => 'Geçmişim';

  @override
  String get historyEmpty =>
      'Geçmişiniz henüz boş. Ziyaretleriniz, belgeleriniz ve yanıtlarınız burada görünür.';

  @override
  String get historyNoMatch => 'Geçmişinizde bu filtrelere uyan bir şey yok.';

  @override
  String get historyClinic => 'Klinik';

  @override
  String get historyAllClinics => 'Tüm klinikler';

  @override
  String get historyVisits => 'Ziyaretler';

  @override
  String get historyVisit => 'Ziyaret';

  @override
  String get historyAnswers => 'Anket yanıtları';

  @override
  String get historyFiles => 'Dosyalar';

  @override
  String get historyFile => 'Dosya';

  @override
  String get historyAnyDate => 'Herhangi bir tarih';

  @override
  String historyDateRange(String from, String to) {
    return '$from – $to';
  }

  @override
  String get historyClearDates => 'Tarihleri temizle';

  @override
  String get historyShowMore => 'Daha fazla göster';

  @override
  String get historyMoreFailed => 'Geçmişinizin devamı yüklenemedi.';

  @override
  String get historyOpenFailed => 'Bu öğe açılamadı. Tekrar deneyin.';

  @override
  String get sharingToggle =>
      'Diğer kliniklerdeki geçmişimi bu klinikle paylaş';

  @override
  String sharingIntro(String clinic) {
    return '$clinic kliniğinin görebileceği önceki ziyaretleri işaretleyin. İşaretlediğiniz her ziyaret için $clinic kliniğinde randevunuz olan doktorlar, burada gösterileni ziyaretin belgeleri, anket yanıtları ve dosyaları, ziyarette kaydedilen alerjiler ve ilaçlar ve ziyarete sonradan eklenen her şeyle birlikte görür. İşaretlemediğiniz ziyaretler $clinic kliniğinden gizli kalır.';
  }

  @override
  String get sharingChangeLater =>
      'Ziyaret eklemeyi veya çıkarmayı ya da paylaşımı durdurmayı istediğiniz zaman Paylaşılan geçmiş bölümünden yapabilirsiniz. Her değişiklik hemen geçerli olur.';

  @override
  String get sharingNoVisits =>
      'Paylaşabileceğiniz, diğer kliniklerde önceki bir ziyaretiniz yok.';

  @override
  String get sharingSelectAll => 'Tümünü seç';

  @override
  String get sharingSelectNone => 'Hiçbirini seçme';

  @override
  String get sharedHistory => 'Paylaşılan geçmiş';

  @override
  String get sharedHistoryIntro =>
      'Onay verdiğiniz klinikler ve her birinin diğer kliniklerdeki ziyaretlerinizden kaçını gördüğü.';

  @override
  String get sharedHistoryNone => 'Henüz hiçbir kliniğe onay vermediniz.';

  @override
  String sharedVisits(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count ziyaret paylaşıldı',
      one: '$count ziyaret paylaşıldı',
      zero: 'Hiçbir şey paylaşılmadı',
    );
    return '$_temp0';
  }

  @override
  String get sharingSaved => 'Seçiminiz kaydedildi ve hemen geçerli.';

  @override
  String get sharingStop => 'Paylaşımı durdur';

  @override
  String get sharingStopTitle => 'Geçmişinizin paylaşımı durdurulsun mu?';

  @override
  String sharingStopBody(String clinic) {
    return '$clinic, diğer kliniklerden onunla paylaştığınız ziyaretlerin hiçbirini artık göremez. Bu hemen geçerli olur.';
  }

  @override
  String get sharingKeep => 'Paylaşmaya devam et';

  @override
  String get sharingStopped =>
      'Geçmişinizi artık bu klinikle paylaşmıyorsunuz.';

  @override
  String get sharingChanged =>
      'Paylaşılan ziyaretleriniz bu arada değişti. Kontrol edip yeniden kaydedin.';

  @override
  String get bookedNotShared =>
      'Randevunuz alındı, ancak geçmişiniz paylaşılamadı. Paylaşılan geçmiş bölümünden paylaşabilirsiniz.';

  @override
  String get consentForHistory => 'Diğer kliniklerden paylaşılan ziyaretler';

  @override
  String get reminders => 'Hatırlatmalar';

  @override
  String get remindersIntro =>
      'Kliniklerin size randevularınızı nasıl hatırlatacağını seçin.';

  @override
  String get remindersPush => 'Anlık bildirimler';

  @override
  String get remindersEmail => 'E-posta';

  @override
  String get remindersSms => 'SMS';

  @override
  String get remindersSmsHint =>
      'Hatırlatmalar ve bekleme listesi teklifleri, yalnızca SMS sunan kliniklerde, profilinizdeki telefon numarasına SMS ile gönderilir. Asla reklam için kullanılmaz.';

  @override
  String get quietHours => 'Sessiz saatler';

  @override
  String get quietHoursHint =>
      'Anlık bildirimler ve SMS\'ler sessiz saatler bitene kadar bekler. O zaman randevuya 30 dakikadan az kalacaksa gönderilmez. E-postalar bekletilmez.';

  @override
  String get quietHoursFrom => 'Başlangıç';

  @override
  String get quietHoursUntil => 'Bitiş';

  @override
  String get remindersSaved => 'Seçimleriniz kaydedildi.';

  @override
  String get remindersPhoneRequired =>
      'SMS için profilinizde bir telefon numarası gerekir. Profilim bölümünden ekleyip tekrar deneyin.';

  @override
  String get remindersInvalid =>
      'Bu ayarlar kaydedilemedi. Kontrol edip tekrar deneyin.';

  @override
  String get confirmAttendance => 'Geleceğimi onayla';

  @override
  String get appointmentConfirmedStatus => 'Onaylandı';

  @override
  String get appointmentConfirmedNotice => 'Randevunuz onaylandı.';

  @override
  String get changeTime => 'Saati değiştir';

  @override
  String rescheduleUntil(String date, String time) {
    return 'Saati değiştirmek için son tarih: $date, saat $time.';
  }

  @override
  String reschedulesLeft(int count) {
    return 'Kalan saat değişikliği hakkı: $count';
  }

  @override
  String get rescheduleTitle => 'Yeni bir saat seçin';

  @override
  String rescheduleCurrent(String date, String time) {
    return 'Şu anki randevu: $date, saat $time';
  }

  @override
  String get rescheduleConfirmTitle => 'Randevu taşınsın mı?';

  @override
  String rescheduleConfirmBody(String date, String time) {
    return 'Randevunuz $date günü saat $time için taşınacak. Mevcut saatiniz serbest bırakılacak.';
  }

  @override
  String get moveAppointment => 'Randevuyu taşı';

  @override
  String get appointmentMoved => 'Randevunuz yeni saate taşındı.';

  @override
  String get rescheduleWindowClosed =>
      'Bu randevunun saati artık uygulamadan değiştirilemez. Lütfen klinikle iletişime geçin.';

  @override
  String get rescheduleLimit =>
      'Bu randevunun saati, kliniğin izin verdiği sayıda değiştirildi.';

  @override
  String get notReschedulable =>
      'Bu randevunun saati uygulamadan değiştirilemez.';

  @override
  String get waitlistTitle => 'Bekleme listem';

  @override
  String get waitlistJoinButton =>
      'Uygun saat yok mu? Bekleme listesine katılın';

  @override
  String get waitlistJoinTitle => 'Bekleme listesine katıl';

  @override
  String get waitlistExplained =>
      'Bu doktorda bir saat boşaldığında, bekleyen hastalara sırayla teklif edilir. Bir bildirim alırsınız ve saat sizin için ayrılmışken uygulamada randevu alabilirsiniz.';

  @override
  String get waitlistHowLong => 'Ne kadar süreyle?';

  @override
  String waitlistDays(int days) {
    String _temp0 = intl.Intl.pluralLogic(
      days,
      locale: localeName,
      other: '$days gün',
      one: '$days gün',
    );
    return '$_temp0';
  }

  @override
  String get waitlistWhen => 'Günün hangi saatinde?';

  @override
  String get waitlistAnyTime => 'Herhangi bir saat';

  @override
  String get waitlistMornings => 'Sabah, 08:00 ile 12:00 arası';

  @override
  String get waitlistAfternoons => 'Öğleden sonra, 12:00 ile 17:00 arası';

  @override
  String get waitlistJoin => 'Bekleme listesine katıl';

  @override
  String get waitlistJoined => 'Bekleme listesindesiniz.';

  @override
  String get waitlistEmpty => 'Hiçbir bekleme listesinde değilsiniz.';

  @override
  String get waitlistAPractitioner => 'Bir doktor';

  @override
  String waitlistUntil(String date) {
    return '$date tarihine kadar bekliyorsunuz';
  }

  @override
  String get waitlistOfferTitle => 'Sizin için bir saat boşaldı';

  @override
  String waitlistHeldUntil(String time) {
    return '$time saatine kadar sizin için ayrıldı';
  }

  @override
  String get waitlistAcceptEffect =>
      'Kabul ettiğinizde bu saat için randevunuz alınır. Salus\'ta ödeme mümkün olduğunda, kabul ettiğinizde ücreti de ödenir.';

  @override
  String get waitlistAccept => 'Kabul et ve randevu al';

  @override
  String get waitlistDecline => 'Reddet';

  @override
  String get waitlistBooked =>
      'Randevunuz alındı. Randevularım bölümünde bulabilirsiniz.';

  @override
  String get waitlistDeclined =>
      'Saati reddettiniz. Bekleme listesinde kalıyorsunuz.';

  @override
  String get waitlistWithdraw => 'Bekleme listesinden çık';

  @override
  String get waitlistWithdrawn => 'Bekleme listesinden çıktınız.';

  @override
  String get waitlistOff => 'Bu kliniğin bekleme listesi yok.';

  @override
  String get waitlistTooMany =>
      'Bu kliniğin bekleme listesinde zaten beş kez varsınız. Önce birinden çıkın.';

  @override
  String get waitlistOfferExpired =>
      'Bu saat artık sizin için ayrılmış değil. Bekleme listesinde kalıyorsunuz.';
}
