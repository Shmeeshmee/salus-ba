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
    return '$clinic artık profilinizi göremeyecek. Klinik, sağlık kayıtlarının bir parçası olarak adınızı ve gerçekleşmiş randevularınızın bilgilerini saklar. $clinic ile olan gelecek randevularınız iptal edilecek.';
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
      'Görüntülü görüşmeye katılmak için Salus\'un kameranıza ve mikrofonunuza erişmesi gerekir. Görüşme kaydedilmez.';

  @override
  String get transcriptionOn => 'Transkripsiyon açık';

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
}
