# 👤 Sentiric User Service

[![Status](https://img.shields.io/badge/status-active-success.svg)]()
[![Language](https://img.shields.io/badge/language-Go-blue.svg)]()
[![Protocol](https://img.shields.io/badge/protocol-gRPC_(mTLS)-green.svg)]()

**Sentiric User Service**, Sentiric platformundaki tüm kimlik ve varlık yönetiminin merkezi ve "tek doğruluk kaynağıdır". Yüksek performans ve eşzamanlılık için **Go** ile yazılmıştır.

## 🎯 Temel Sorumluluklar

Bu servis, platformun "Omnichannel" vizyonunu destekleyen, ölçeklenebilir bir veri modeli üzerine kurulmuştur:

1.  **Profil Yönetimi (`users`):** Bir varlığın (bir arayan, bir ajan veya bir yönetici) temel profilini yönetir. Bu profil, iletişim kanallarından bağımsızdır ve benzersiz bir `UUID` ile tanımlanır.
2.  **İletişim Kanalı Yönetimi (`contacts`):** Bir kullanıcının sahip olabileceği birden fazla iletişim kanalını (telefon numarası, WhatsApp ID, e-posta adresi vb.) yönetir ve bunları ana kullanıcı profiline bağlar.
3.  **Kimlik Doğrulama ve Yetkilendirme:** Gelen bir iletişim bilgisine (`contact_value`) dayanarak, bu kişinin platformda kim olduğunu bulur ve `dialplan-service` gibi diğer servislere bu bilgiyi sunar.
4.  **CRUD Operasyonları:** Yönetici paneli (`dashboard-ui`) ve CLI gibi araçların, kullanıcıları ve iletişim kanallarını oluşturması, okuması, güncellemesi ve silmesi (CRUD) için güvenli gRPC endpoint'leri sağlar.

## 🛠️ Teknoloji Yığını

*   **Dil:** Go
*   **Servisler Arası İletişim:** gRPC (mTLS ile güvenli hale getirilmiş)
*   **Veritabanı:** PostgreSQL (`pgx` kütüphanesi ile)
*   **Loglama:** `zerolog` ile yapılandırılmış, ortama duyarlı loglama.
*   **API Sözleşmeleri:** `sentiric-contracts` reposunda tanımlanan Protobuf dosyaları.

## 🔌 API Etkileşimleri (Server For)

Bu servis, diğer iç (backend) servislere gRPC üzerinden hizmet verir.

*   **`sentiric-dialplan-service` (gRPC):** Bir arama geldiğinde, arayanın kim olduğunu ve hangi `tenant`'a ait olduğunu doğrulamak için `FindUserByContact` RPC'sini çağırır.
*   **`sentiric-api-gateway-service` (gRPC):** Yönetici panelinden gelen kullanıcı yönetimi (CRUD) isteklerini bu servise yönlendirir.
*   **`sentiric-agent-service` (gRPC):** İş akışları sırasında kullanıcı detaylarına veya yetkilerine erişmek için bu servisi çağırabilir.

## 🚀 Yerel Geliştirme

Bu servis, platformun bir parçası olarak `sentiric-infrastructure` reposundaki merkezi `docker-compose` dosyası ile çalıştırılmak üzere tasarlanmıştır.

1.  **Bağımlılıkları Yükle:**
    ```bash
    go mod tidy
    ```
2.  **Ortam Değişkenlerini Ayarla:**
    `.env.example` dosyasını `.env` olarak kopyalayın ve `POSTGRES_URL` gibi gerekli değişkenleri doldurun.
3.  **Servisi Çalıştır:**
    ```bash
    go run main.go
    ```

## 🤝 Katkıda Bulunma

Katkılarınızı bekliyoruz! Lütfen projenin ana [Sentiric Governance](https://github.com/sentiric/sentiric-governance) reposundaki kodlama standartlarına ve katkıda bulunma rehberine göz atın.
