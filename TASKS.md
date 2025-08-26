# 👤 Sentiric User Service - Geliştirme Yol Haritası (v4.0)

Bu belge, `user-service`'in geliştirme görevlerini projenin genel fazlarına uygun olarak listeler.

---

### **FAZ 1: Temel Varlık Yönetimi (Mevcut Durum)**

**Amaç:** Diğer servislerin temel çağrı akışını tamamlayabilmesi için gereken minimum kullanıcı bulma ve oluşturma yeteneklerini sağlamak.

-   [x] **Görev ID: USER-000A - Temel gRPC Sunucusu ve Veritabanı Entegrasyonu**
    -   **Durum:** ✅ **Tamamlandı**
    -   **Kabul Kriterleri:** Servis, mTLS ile güvenli bir gRPC sunucusu sunar ve PostgreSQL'e başarılı bir şekilde bağlanır.

-   [x] **Görev ID: USER-000B - `FindUserByContact` RPC'si**
    -   **Durum:** ✅ **Tamamlandı**
    -   **Kabul Kriterleri:** Bir iletişim bilgisine (`contact_value`) göre kullanıcıyı ve tüm iletişim kanallarını (`contacts`) başarılı bir şekilde döndürür. Kullanıcı bulunamazsa `NOT_FOUND` hatası verir.

-   [x] **Görev ID: USER-000C - `CreateUser` RPC'si**
    -   **Durum:** ✅ **Tamamlandı**
    -   **Kabul Kriterleri:** Yeni bir kullanıcıyı ve ona bağlı ilk iletişim kanalını tek bir atomik işlemle (transaction) oluşturur. Başarı durumunda oluşturulan tam `User` nesnesini döndürür.

-   [x] **Görev ID: USER-000D - `GetUser` RPC'si**
    -   **Durum:** ✅ **Tamamlandı**
    -   **Kabul Kriterleri:** Bir `user_id`'ye göre kullanıcıyı ve tüm iletişim kanallarını başarılı bir şekilde döndürür.

---

### **FAZ 2: Platformun Yönetilebilir Hale Getirilmesi (Sıradaki Öncelik)**

**Amaç:** `dashboard-ui` gibi yönetim araçlarının, platformdaki tüm kullanıcıları ve kiracıları tam olarak yönetebilmesini sağlamak.

-   [ ] **Görev ID: USER-001 - `UpdateUser` RPC'si**
    -   **Açıklama:** Bir kullanıcının adını, tipini veya tercih ettiği dili güncellemek için bir RPC ekle.
    -   **Kabul Kriterleri:**
        -   [ ] RPC, güncellenecek alanları içeren bir `UpdateUserRequest` mesajı almalı.
        -   [ ] Sadece gönderilen alanlar (`name`, `user_type` vb.) güncellenmeli (kısmi güncelleme).
        -   [ ] Başarı durumunda güncellenmiş tam `User` nesnesini döndürmeli.

-   [ ] **Görev ID: USER-002 - `DeleteUser` RPC'si**
    -   **Açıklama:** Bir kullanıcıyı ve ona bağlı tüm varlıkları güvenli bir şekilde silen bir RPC ekle.
    -   **Kabul Kriterleri:**
        -   [ ] RPC, silinecek `user_id`'yi almalı.
        -   [ ] Veritabanı `ON DELETE CASCADE` özelliği sayesinde, kullanıcı silindiğinde tüm `contacts` kayıtları da otomatik olarak silinmeli.
        -   [ ] Başarılı silme işleminden sonra boş bir yanıt (`Empty`) dönmeli.

-   [ ] **Görev ID: USER-003 - İletişim Kanalı Yönetimi RPC'leri (`AddContact`, `DeleteContact`)**
    -   **Açıklama:** Mevcut bir kullanıcıya yeni iletişim kanalları eklemek veya mevcut olanları silmek için RPC'ler oluştur.
    -   **Kabul Kriterleri:**
        -   [ ] `AddContact(user_id, contact_type, contact_value)` RPC'si implemente edilmeli.
        -   [ ] `DeleteContact(contact_id)` RPC'si implemente edilmeli.
        -   [ ] Her iki işlem de başarı durumunda güncellenmiş tam `User` nesnesini döndürmeli.

-   [ ] **Görev ID: USER-005 - Listeleme ve Sayfalama RPC'leri (`ListUsers`, `ListTenants`)**
    -   **Açıklama:** Yönetici panelleri için kullanıcıları ve kiracıları listeleyen, sayfalama (`pagination`) destekli RPC'ler oluştur.
    -   **Kabul Kriterleri:**
        -   [ ] `ListUsers(tenant_id, page_size, page_token)` RPC'si implemente edilmeli.
        -   [ ] `ListTenants(page_size, page_token)` RPC'si implemente edilmeli.
        -   [ ] Yanıtlar, sonuç listesini ve bir sonraki sayfaya geçmek için bir `next_page_token` içermeli.

---

### **FAZ 3: Yetkilendirme ve Gelişmiş Özellikler**

**Amaç:** Servise daha granüler erişim kontrolü ve güvenlik yetenekleri kazandırmak.

-   [ ] **Görev ID: USER-004 - Rol Yönetimi**
    -   **Açıklama:** `roles` ve `user_roles` tabloları ekleyerek, kullanıcılara "admin", "agent", "supervisor" gibi roller atama yeteneği ekle.
    -   **Durum:** ⬜ Planlandı.