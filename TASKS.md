# 👤 Sentiric User Service - Geliştirme Yol Haritası (v4.1)

Bu belge, `user-service`'in geliştirme görevlerini projenin genel fazlarına uygun olarak listeler.

---



### **FAZ 2: Platformun Yönetilebilir Hale Getirilmesi (Mevcut Öncelik)**

**Amaç:** `dashboard-ui` gibi yönetim araçlarının, platformdaki tüm kullanıcıları ve kiracıları tam olarak yönetebilmesini sağlamak.
**Ön Koşul:** `sentiric-contracts` v1.8.4+ sürümünün yayınlanmış olması.

-   [ ] **Görev ID: USER-002 - `UpdateUser` RPC'si**
    -   **Durum:** ⬜ **Bloklandı** (CT-002 bekleniyor)
    -   **Kabul Kriterleri:**
        -   [ ] `server/grpc.go` içine yeni `UpdateUser` RPC metodu implemente edilmeli.
        -   [ ] Metod, `FieldMask`'i kullanarak dinamik bir SQL `UPDATE` sorgusu oluşturmalı ve sadece istenen alanları (`name`, `user_type`, `preferred_language_code`) güncellemeli.
        -   [ ] Başarı durumunda, güncellenmiş tam `User` nesnesini (`fetchUserByID` ile) döndürmeli.
        -   [ ] Kullanıcı bulunamazsa `NOT_FOUND` hatası vermeli.

-   [ ] **Görev ID: USER-003 - `DeleteUser` RPC'si**
    -   **Durum:** ⬜ **Bloklandı** (CT-002 bekleniyor)
    -   **Kabul Kriterleri:**
        -   [ ] `server/grpc.go` içine yeni `DeleteUser` RPC metodu implemente edilmeli.
        -   [ ] Metod, `user_id`'ye göre bir `DELETE FROM users WHERE id = $1` sorgusu çalıştırmalı.
        -   [ ] Veritabanındaki `ON DELETE CASCADE` kuralı sayesinde, kullanıcıya ait tüm `contacts` kayıtları otomatik olarak silinmeli.
        -   [ ] Başarılı silme işleminden sonra `DeleteUserResponse` dönmeli.
        -   [ ] Kullanıcı bulunamazsa `NOT_FOUND` hatası vermeli.

-   [ ] **Görev ID: USER-004A - `AddContact` RPC'si**
    -   **Durum:** ⬜ **Bloklandı** (CT-002 bekleniyor)
    -   **Kabul Kriterleri:**
        -   [ ] `server/grpc.go` içine yeni `AddContact` RPC metodu implemente edilmeli.
        -   [ ] Metod, belirtilen `user_id`'ye yeni bir `contact` eklemeli.
        -   [ ] `(contact_type, contact_value)` kombinasyonunun benzersizliğini (unique) ihlal ederse `ALREADY_EXISTS` hatası vermeli.
        -   [ ] Başarı durumunda, güncellenmiş tam `User` nesnesini döndürmeli.

-   [ ] **Görev ID: USER-004B - `UpdateContact` ve `DeleteContact` RPC'leri**
    -   **Durum:** ⬜ **Bloklandı** (CT-002 bekleniyor)
    -   **Kabul Kriterleri:**
        -   [ ] `UpdateContact` RPC'si, bir `contact` kaydının bilgilerini (`contact_value`, `is_primary`) `FieldMask` kullanarak güncellemeli.
        -   [ ] `DeleteContact` RPC'si, `contact_id`'ye göre bir `contact` kaydını silmeli.
        -   [ ] Her iki işlem de başarı durumunda güncellenmiş tam `User` nesnesini döndürmeli.

-   [ ] **Görev ID: USER-005 - Listeleme ve Sayfalama RPC'leri**
    -   **Durum:** ⬜ **Planlandı** (Bu görev için kontrat değişikliği gerekebilir, şimdilik bekliyor)

---

### **FAZ 3: Yetkilendirme ve Gelişmiş Özellikler**
-   [ ] **Görev ID: USER-006 - Rol Yönetimi**
    -   **Durum:** ⬜ **Planlandı**