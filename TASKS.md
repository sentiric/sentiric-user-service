# 👤 Sentiric User Service - Geliştirme Yol Haritası (v4.0)

Bu belge, `user-service`'in geliştirme görevlerini projenin genel fazlarına uygun olarak listeler.

---

### **FAZ 0: SÜRDÜRÜLEBİLİR TEMELİN ATILMASI (Mevcut Öncelik)**

**Amaç:** Kod tabanını, gelecekteki geliştirmeleri kolaylaştıracak, test edilebilirliği artıracak ve bakımı basitleştirecek modüler bir mimariye kavuşturmak.

-   [x] **Görev ID: USER-000A - Temel gRPC Sunucusu ve Veritabanı Entegrasyonu**
    -   **Durum:** ✅ **Tamamlandı**
    -   **Kabul Kriterleri:** Servis, mTLS ile güvenli bir gRPC sunucusu sunar ve PostgreSQL'e başarılı bir şekilde bağlanır.

-   [⏳] **Görev ID: USER-001A - Monolitik Yapının Modülerleştirilmesi**
    -   **Durum:** ⏳ **Devam Ediyor**
    -   **Açıklama:** `main.go` içerisindeki tüm mantığı; konfigürasyon (`config`), veritabanı bağlantısı (`database`) ve gRPC sunucu yönetimi (`server`) gibi sorumlulukları ayrılmış paketlere böl.
    -   **Kabul Kriterleri:**
        -   [ ] `main.go` dosyası sadece uygulamanın başlangıç noktası haline gelmeli.
        -   [ ] Veritabanı bağlantı mantığı `internal/database` paketine taşınmalı.
        -   [ ] Ortam değişkeni yönetimi `internal/config` paketine taşınmalı.
        -   [ ] gRPC sunucusu ve handler'ları `internal/server` paketine taşınmalı.
        -   [ ] Refaktör sonrası servis, mevcut tüm işlevselliğini korumalı.

---

### **FAZ 1: Temel Varlık Yönetimi**

**Amaç:** Diğer servislerin temel çağrı akışını tamamlayabilmesi için gereken minimum kullanıcı bulma ve oluşturma yeteneklerini sağlamak.

-   [x] **Görev ID: USER-000B - `FindUserByContact` RPC'si**
    -   **Durum:** ✅ **Tamamlandı**

-   [x] **Görev ID: USER-000C - `CreateUser` RPC'si**
    -   **Durum:** ✅ **Tamamlandı**

-   [x] **Görev ID: USER-000D - `GetUser` RPC'si**
    -   **Durum:** ✅ **Tamamlandı**

---

### **FAZ 2: Platformun Yönetilebilir Hale Getirilmesi (Sıradaki Öncelik)**

**Amaç:** `dashboard-ui` gibi yönetim araçlarının, platformdaki tüm kullanıcıları ve kiracıları tam olarak yönetebilmesini sağlamak.

-   [ ] **Görev ID: USER-002 - `UpdateUser` RPC'si**
    -   **Açıklama:** Bir kullanıcının adını, tipini veya tercih ettiği dili güncellemek için bir RPC ekle.

-   [ ] **Görev ID: USER-003 - `DeleteUser` RPC'si**
    -   **Açıklama:** Bir kullanıcıyı ve ona bağlı tüm varlıkları güvenli bir şekilde silen bir RPC ekle.

-   [ ] **Görev ID: USER-004 - İletişim Kanalı Yönetimi RPC'leri (`AddContact`, `DeleteContact`)**
    -   **Açıklama:** Mevcut bir kullanıcıya yeni iletişim kanalları eklemek veya mevcut olanları silmek için RPC'ler oluştur.

-   [ ] **Görev ID: USER-005 - Listeleme ve Sayfalama RPC'leri (`ListUsers`, `ListTenants`)**
    -   **Açıklama:** Yönetici panelleri için kullanıcıları ve kiracıları listeleyen, sayfalama (`pagination`) destekli RPC'ler oluştur.

---

### **FAZ 3: Yetkilendirme ve Gelişmiş Özellikler**

-   [ ] **Görev ID: USER-006 - Rol Yönetimi**
    -   **Açıklama:** `roles` ve `user_roles` tabloları ekleyerek, kullanıcılara "admin", "agent", "supervisor" gibi roller atama yeteneği ekle.