# Sentiric User Service - Görev Listesi ve Yol Haritası

Bu belge, `user-service`'in geliştirme önceliklerini ve gelecekte yapılacak önemli görevleri takip eder.

---

## Faz 1: Çekirdek Altyapı Entegrasyonu (Mevcut Odak)

Bu fazın amacı, servisin platformun geri kalanıyla konuşabilen, çalışan bir gRPC iskeleti haline getirilmesidir.

-   [x] **Temel gRPC Sunucusu Implementasyonu**
    -   **Açıklama:** `user.proto`'da tanımlanan `AuthenticateUser` RPC'sini implemente eden bir gRPC sunucusu oluşturuldu.
    -   **Durum:** ✅ Tamamlandı.
    -   **Not:** Bu ilk versiyon, hız ve bağımsız geliştirme için veritabanı yerine **hafızadaki mock verileri** kullanmaktadır.

---

## Faz 2: Üretime Hazırlık (Sıradaki Öncelik)

Bu faz, servisi gerçek dünya verileriyle çalışacak, güvenilir ve yönetilebilir hale getirmeyi hedefler.

-   [ ] **PostgreSQL Veritabanı Entegrasyonu**
    -   **Görev ID:** `user-task-001`
    -   **Açıklama:** Mevcut hafızadaki mock kullanıcı listesini, kalıcı bir PostgreSQL veritabanı bağlantısıyla değiştir. `AuthenticateUser` RPC'si, gelen telefon numarasını `customers` tablosunda sorgulamalıdır.
    -   **Kabul Kriterleri:**
        -   [ ] Veritabanı bağlantı detayları ortam değişkenlerinden (`DATABASE_URL`) okunmalıdır.
        -   [ ] Gerekli tablo şeması `sentiric-db-models` reposunda tanımlanmalı veya burada belgelenmelidir.
        -   [ ] Başarılı ve başarısız veritabanı sorguları için loglama eklenmelidir.
    -   **Durum:** ⬜ Planlandı.

-   [ ] **CRUD Operasyonları için gRPC Endpoint'leri Ekleme**
    -   **Görev ID:** `user-task-002`
    -   **Açıklama:** Yönetim panelinin (`dashboard-ui`) kullanıcıları yönetebilmesi için `CreateUser`, `GetUser`, `UpdateUser`, `DeleteUser` gibi RPC'leri `user.proto`'ya ve servise ekle.
    -   **Durum:** ⬜ Planlandı.

-   [ ] **Yapılandırılmış Loglamayı Geliştirme**
    -   **Görev ID:** `user-task-003`
    -   **Açıklama:** Tüm loglara `trace_id` ve `tenant_id` gibi, dağıtık sistemlerde hata ayıklamayı kolaylaştıracak bağlam bilgilerini ekle.
    -   **Durum:** ⬜ Planlandı.