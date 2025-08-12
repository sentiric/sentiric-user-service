# 👤 Sentiric User Service - Görev Listesi

Bu belge, `user-service`'in geliştirme yol haritasını ve önceliklerini tanımlar.

---

### Faz 1: Temel Varlık Yönetimi (Mevcut Durum)

Bu faz, servisin temel kullanıcı bulma ve oluşturma görevlerini yerine getirmesini hedefler.

-   [x] **gRPC Sunucusu:** `user.proto`'da tanımlanan RPC'leri implemente eden sunucu.
-   [x] **Veritabanı Entegrasyonu:** PostgreSQL'e bağlanma ve sorgu yapma.
-   [x] **`FindUserByContact` RPC:** Bir iletişim bilgisine göre kullanıcıyı ve tüm iletişim kanallarını getirme.
-   [x] **`CreateUser` RPC:** Yeni bir kullanıcı profili ve ona bağlı ilk iletişim kanalını atomik bir işlemle (transaction) oluşturma.
-   [x] **`GetUser` RPC:** Bir `user_id`'ye göre kullanıcıyı ve tüm kanallarını getirme.

---

### Faz 2: Tam CRUD ve Gelişmiş Yönetim (Sıradaki Öncelik)

Bu faz, servisi `dashboard-ui` üzerinden tam teşekküllü bir kullanıcı yönetimi merkezi haline getirmeyi hedefler.

-   [ ] **Görev ID: USER-001 - `UpdateUser` RPC**
    -   **Açıklama:** Bir kullanıcının adını veya tipini güncellemek için bir RPC ekle.
    -   **Durum:** ⬜ Planlandı.

-   [ ] **Görev ID: USER-002 - `DeleteUser` RPC**
    -   **Açıklama:** Bir kullanıcıyı ve ona bağlı tüm iletişim kanallarını güvenli bir şekilde (soft delete veya hard delete) silen bir RPC ekle.
    -   **Durum:** ⬜ Planlandı.

-   [ ] **Görev ID: USER-003 - İletişim Kanalı Yönetimi RPC'leri**
    -   **Açıklama:** Mevcut bir kullanıcıya yeni bir iletişim kanalı eklemek (`AddContact`), bir kanalı güncellemek (`UpdateContact`) veya silmek (`DeleteContact`) için RPC'ler ekle.
    -   **Durum:** ⬜ Planlandı.

---

### Faz 3: Yetkilendirme ve Roller

Bu faz, servise daha granüler erişim kontrolü yetenekleri kazandırmayı hedefler.

-   [ ] **Görev ID: USER-004 - Rol Yönetimi**
    -   **Açıklama:** `roles` ve `user_roles` tabloları ekleyerek, kullanıcılara "admin", "agent", "supervisor" gibi roller atama yeteneği ekle.
    -   **Durum:** ⬜ Planlandı.