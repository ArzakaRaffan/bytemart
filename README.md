# ByteMart - Event-Driven System dengan RabbitMQ

## Deskripsi Proyek

ByteMart adalah sistem e-commerce berbasis microservices yang mengimplementasikan arsitektur event-driven menggunakan RabbitMQ sebagai message broker. Sistem ini mendemonstrasikan bagaimana komponen aplikasi dapat berkomunikasi secara asynchronous tanpa saling bergantung satu sama lain.

Sistem terdiri dari tiga service utama: Order Service sebagai producer yang berjalan di port 3001, Notification Service sebagai consumer pertama di port 3002, dan Inventory Service sebagai consumer kedua di port 3003. Ketiganya berkomunikasi melalui RabbitMQ menggunakan exchange bertipe topic bernama `bytemart.events`.

---

## Tech Stack

- **Backend**: Go 1.22 + Fiber v2
- **Message Broker**: RabbitMQ 3.12 (topic exchange)
- **Database**: PostgreSQL 16 (3 database terpisah per service)
- **ORM**: GORM
- **Infrastructure**: Docker

---

## Cara Menjalankan

### Requirements
- Go 1.22+
- Docker Desktop
- PowerShell (Windows)

### Langkah 1 Jalankan RabbitMQ & PostgreSQL

```powershell
cd bytemart/docker
docker compose up -d
```

### Langkah 2 Jalankan Order Service (Terminal 1)

```powershell
cd bytemart/order-service
$env:DB_HOST="localhost"; $env:DB_PORT="5433"; $env:DB_USER="bytemart"; $env:DB_PASSWORD="arzaka22"; $env:DB_NAME="bytemart_orders"
go run main.go
```

### Langkah 3 Jalankan Notification Service (Terminal 2)

```powershell
cd bytemart/notification-service
$env:DB_HOST="localhost"; $env:DB_PORT="5433"; $env:DB_USER="bytemart"; $env:DB_PASSWORD="arzaka22"; $env:DB_NAME="bytemart_notifications"
go run main.go
```

### Langkah 4 Jalankan Inventory Service (Terminal 3)

```powershell
cd bytemart/inventory-service
$env:DB_HOST="localhost"; $env:DB_PORT="5433"; $env:DB_USER="bytemart"; $env:DB_PASSWORD="arzaka22"; $env:DB_NAME="bytemart_inventory"
go run main.go
```

---

## Testing

### Kirim Order (Terminal 4)

```powershell
# Order 1
Invoke-RestMethod -Method POST -Uri "http://localhost:3001/api/orders" `
  -ContentType "application/json" `
  -Body '{"user_id":"budi","product_id":"PROD-001","quantity":2,"total":20000000}'

# Order 2
Invoke-RestMethod -Method POST -Uri "http://localhost:3001/api/orders" `
  -ContentType "application/json" `
  -Body '{"user_id":"siti","product_id":"PROD-002","quantity":1,"total":500000}'

# Order 3
Invoke-RestMethod -Method POST -Uri "http://localhost:3001/api/orders" `
  -ContentType "application/json" `
  -Body '{"user_id":"andi","product_id":"PROD-003","quantity":5,"total":2500000}'
```

### Cek Hasil

```powershell
# Lihat semua order
Invoke-RestMethod -Uri "http://localhost:3001/api/orders"

# Lihat notifikasi yang masuk
Invoke-RestMethod -Uri "http://localhost:3002/api/notifications"

# Lihat stok terkini
Invoke-RestMethod -Uri "http://localhost:3003/api/stock"

# Lihat log perubahan stok
Invoke-RestMethod -Uri "http://localhost:3003/api/stock-logs"
```

### Hasil yang Diharapkan

Saat satu order dikirim, dua terminal consumer langsung merespons secara bersamaan tanpa Order Service perlu memanggil mereka secara langsung.

Terminal Notification Service:
```
Notification added for [budi]: Hello, budi! Order #dc16d1f0 for 2 items worth Rp20000000 has been received.
```

Terminal Inventory Service:
```
Stock of PROD-001 reduced by 2, remaining: 98
```

---

## Penjelasan Mekanisme Event-Driven

### 1. Producer Mengirim Event

Ketika user membuat order baru via `POST /api/orders`, Order Service menyimpan data ke database lalu mempublish event `order.created` ke exchange `bytemart.events` di RabbitMQ. Publish dilakukan di goroutine terpisah sehingga response langsung dikirim ke client tanpa menunggu consumer selesai memproses.

```go
// Publish dilakukan di background, tidak memblokir response
go func() {
    publisher.Publish("order.created", event)
}()

// Response langsung dikirim ke client
return c.Status(201).JSON(...)
```

### 2. Event Didistribusikan ke Queue

RabbitMQ menerima event dan mendistribusikannya ke dua queue berbeda berdasarkan routing key `order.created`. Queue `notification.queue` diterima oleh Notification Service, dan `inventory.queue` diterima oleh Inventory Service. Karena menggunakan topic exchange, satu event bisa diterima oleh banyak consumer secara independen dan paralel tanpa perubahan apapun di sisi producer.

### 3. Consumer Memproses Event dengan Ack/Nack

Setiap consumer menggunakan mekanisme acknowledgement untuk menjamin pesan tidak hilang:

```
Berhasil proses  → msg.Ack()               → pesan dihapus dari queue
Gagal (DB error) → msg.Nack(requeue=true)  → pesan dikembalikan ke queue untuk dicoba ulang
Gagal (parse)    → msg.Nack(requeue=false) → pesan dibuang
```

Tanpa Ack, pesan akan tetap dianggap "belum diproses" oleh RabbitMQ dan akan dikirim ulang terus-menerus, menyebabkan duplikasi data.

---

## Perbedaan Asynchronous vs Request-Response

### Komunikasi Request-Response (Synchronous)

Pada komunikasi synchronous biasa, Order Service harus memanggil Notification Service dan Inventory Service secara langsung dan menunggu respons masing-masing sebelum bisa membalas client. Total waktu response bergantung pada kecepatan semua service yang dipanggil. Jika salah satu service mati, seluruh request gagal. Menambah consumer baru berarti harus mengubah kode Order Service.

### Komunikasi Event-Driven (Asynchronous)

Dengan event-driven, Order Service cukup mempublish satu event ke RabbitMQ dan langsung selesai. Waktu response ke client hanya sekitar 20ms terlepas dari berapa banyak consumer yang terlibat. Jika salah satu consumer mati, event tetap tersimpan di queue dan diproses otomatis saat consumer kembali online. Menambah consumer baru tidak memerlukan perubahan apapun di Order Service cukup subscribe ke queue yang sama.


### Disclosure Penggunaan AI
Dalam pengerjaan proyek ini, saya menggunakan bantuan AI (Claude by Anthropic) secara terbatas, yaitu sebagai alat bantu debugging ketika menghadapi error konfigurasi RabbitMQ dan Docker, serta untuk memperjelas konsep-konsep teknis seperti mekanisme Ack/Nack dan perbedaan komunikasi synchronous vs asynchronous. Seluruh kode ditulis dan diketik secara mandiri, dipahami baris per baris sebelum diimplementasikan, dan diuji langsung di environment lokal. AI tidak digunakan untuk men-generate kode secara langsung maupun menulis laporan ini.