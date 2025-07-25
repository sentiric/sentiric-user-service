# --- AŞAMA 1: Derleme (Builder) ---
# Go'nun resmi imajını temel al
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Önce sadece bağımlılık dosyalarını kopyala (önbellekleme için)
COPY go.mod go.sum ./
RUN go mod download

# Şimdi kaynak kodunu ve üretilmiş gRPC kodunu kopyala
COPY . .

# Uygulamayı derle (statik olarak, C kütüphaneleri olmadan)
RUN CGO_ENABLED=0 GOOS=linux go build -o /user-service .

# --- AŞAMA 2: Çalıştırma (Runtime) ---
# Sadece derlenmiş binary'i içeren minimal bir scratch imajı kullan
FROM scratch

WORKDIR /

# Derlenmiş uygulamayı builder aşamasından kopyala
COPY --from=builder /user-service .

# gRPC portunu belirt
EXPOSE 50053

# Konteyner başladığında uygulamayı çalıştır
ENTRYPOINT ["/user-service"]