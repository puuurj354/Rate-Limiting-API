# Rate Limiting API - Dokumentasi

## Deskripsi Project

Implementasi lengkap **Leaky Bucket Algorithm** untuk rate limiting API menggunakan:
- **Go** dengan framework Gin
- **Redis** untuk menyimpan state rate limiter
- **HTMX** + **Tailwind CSS** untuk dashboard interaktif

## Struktur Project

```
.
├── internal/
│   ├── limiter/           # Core rate limiting logic
│   │   ├── limiter.go       # Interface RateLimiter
│   │   ├── leaky_bucket.go  # Implementasi Leaky Bucket
│   │   └── leaky_bucket_test.go
│   ├── middleware/         # Gin middleware
│   │   ├── ratelimit.go     # Rate limiting middleware
│   │   └── ratelimit_test.go
│   ├── dashboard/         # Dashboard handlers
│   │   └── handler.go
│   └── storage/           # Redis integration
│       └── redis.go
├── templates/             # HTML templates
│   ├── index.html
│   └── partials/
│       ├── status.html
│       ├── keys.html
│       └── test_result.html
├── main.go               # Entry point
├── go.mod
└── README.md
```

## Setup & Instalasi

### 1. Prerequisites
- Go 1.20+
- Redis 6.0+

### 2. Install Dependencies

```bash
go mod download
go mod tidy
```

### 3. Start Redis

```bash
# Dengan Docker
docker run -d --name redis -p 6379:6379 redis:latest

# Atau gunakan Redis yang sudah terinstall
redis-server
```

### 4. Run Application

```bash
cd /home/purnama/Documents/Rate-Limiting-API
go run main.go
```

Server akan berjalan di `http://localhost:8080`

## Fitur & Endpoints

### Dashboard (Tanpa Rate Limiting)
| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| GET | `/dashboard/` | Halaman dashboard utama |
| GET | `/dashboard/status` | Status rate limiter untuk IP Anda |
| GET | `/dashboard/status/json` | Status dalam format JSON |
| GET | `/dashboard/keys` | Daftar semua keys yang sedang di-track |
| GET | `/dashboard/keys/json` | Daftar keys dalam JSON |
| POST | `/dashboard/reset` | Reset rate limit untuk key tertentu |
| POST | `/dashboard/test` | Test request untuk demo |

### API Routes (Dengan Rate Limiting)
| Method | Endpoint | Deskripsi | Limit |
|--------|----------|-----------|-------|
| GET | `/api/ping` | Simple ping endpoint | 10 req/IP |
| GET | `/api/data` | Contoh protected data | 10 req/IP |

### Health Check (Tanpa Rate Limiting)
| Method | Endpoint | Response |
|--------|----------|----------|
| GET | `/health` | `{"status": "healthy"}` |

## Konfigurasi Rate Limiting

Edit `main.go` untuk mengubah konfigurasi:

```go
// Default: 10 requests, 2 leak/sec, TTL 1 hour
rateLimiter := limiter.NewLeakyBucket(
    10,              // Capacity (request max)
    2,               // LeakRate (requests that "leak" per second)
    time.Hour,       // TTL (time-to-live di Redis)
)
```

## Cara Kerja Leaky Bucket Algorithm

### Konsep
- Bucket memiliki **kapasitas terbatas** (default: 10 requests)
- Setiap request "mengisi" bucket dengan 1 unit
- Bucket "bocor" pada rate tertentu (default: 2 per detik)
- Jika bucket penuh → request ditolak (429 Too Many Requests)

### Contoh Timeline
```
Time 0:00 → Request 1-9 diterima (bucket fill: 9)
Time 0:00 → Request 10 diterima (bucket full: 10) ⬛⬛⬛⬛⬛⬛⬛⬛⬛⬛
Time 0:01 → Request 11 DITOLAK! Bucket masih penuh setelah leak

Time 0:30 → Leak 60 units (30 detik × 2/sec), bucket kosong
Time 0:30 → Request baru diterima lagi
```

## Test Implementasi

### Run Unit Tests

```bash
# Test limiter package
cd internal/limiter && go test -v

# Test middleware package
cd internal/middleware && go test -v
```

### Manual Testing

```bash
# Test 15 requests (limit adalah 10)
for i in {1..15}; do 
    curl http://localhost:8080/api/ping | jq .
done

# Test dengan JSON response
curl http://localhost:8080/dashboard/status/json | jq .

# List semua active keys
curl http://localhost:8080/dashboard/keys/json | jq .

# Reset rate limit untuk IP specific
curl -X POST http://localhost:8080/dashboard/reset \
  -d "key=127.0.0.1"
```

## Custom Rate Limiting per Endpoint

### By IP Address (Default)
```go
apiGroup.Use(middleware.RateLimit(rateLimiter))
```

### By API Key
```go
apiGroup.Use(middleware.RateLimitByAPIKey(rateLimiter, "X-API-Key"))
// Client harus kirim: curl -H "X-API-Key: mykey" http://localhost:8080/api/...
```

### By User ID
```go
// Asumsikan user ID sudah di-set di context
authMiddleware := func(c *gin.Context) {
    c.Set("user_id", "user123")
    c.Next()
}

apiGroup.Use(authMiddleware)
apiGroup.Use(middleware.RateLimitByUserID(rateLimiter, "user_id"))
```

## Middleware Kustom

Buat custom middleware dengan konfigurasi sendiri:

```go
customMiddleware := middleware.RateLimitWithConfig(
    middleware.RateLimitConfig{
        Limiter: rateLimiter,
        KeyFunc: func(c *gin.Context) string {
            return c.Request.Header.Get("X-Custom-ID")
        },
        ErrHandler: func(c *gin.Context) {
            c.JSON(429, gin.H{"error": "Too many requests"})
            c.Abort()
        },
    },
)

apiGroup.Use(customMiddleware)
```

## Response Headers

Setiap response dari API yang di-rate-limit akan menyertakan:
```
X-RateLimit-Remaining: 5
```

Ini menunjukkan berapa capacity yang tersisa.

## Troubleshooting

### Error: "address already in use"
```bash
# Kill existing processes
pkill -f "go run main.go"
```

### Error: "Failed to connect to Redis"
```bash
# Pastikan Redis running
redis-cli ping  # Should return PONG

# Jika tidak running, start Redis
redis-server
```

### Rate limiting tidak bekerja
1. Pastikan Redis terkoneksi dengan benar
2. Cek apakah TTL sudah expire (default 1 jam)
3. Reset rate limiter: `curl -X POST http://localhost:8080/dashboard/reset -d "key=127.0.0.1"`

## Performance Notes

- **Current**: ~105µs per request (terukur di `/health`)
- **Memory**: Redis menyimpan 2 keys per rate limiter (water level + timestamp)
- **TTL**: Default 1 hour - ubah sesuai kebutuhan untuk menghemat memory

## Production Deployment

1. **Disable Debug Mode**
   ```go
   gin.SetMode(gin.ReleaseMode)
   ```

2. **Set Trusted Proxies**
   ```go
   r.SetTrustedProxies([]string{"10.0.0.0/8"})
   ```

3. **Gunakan Environment Variables**
   ```go
   redis_url := os.Getenv("REDIS_URL")
   port := os.Getenv("PORT")
   ```

4. **Add Proper Logging**
   ```go
   import "log"
   ```

## Implementasi Algoritma Lain

Untuk menambah algoritma rate limiting lain (Token Bucket, Sliding Window, dll), cukup implement `RateLimiter` interface:

```go
type MyCustomLimiter struct {
    // fields...
}

func (m *MyCustomLimiter) Allow(ctx context.Context, key string) (bool, float64, error) {
    // implementasi
}

func (m *MyCustomLimiter) Reset(ctx context.Context, key string) error {
    // implementasi
}

func (m *MyCustomLimiter) GetStatus(ctx context.Context, key string) (*Status, error) {
    // implementasi
}
```

## Test Coverage

- **Limiter**: 12 tests (NewLeakyBucket, Allow, Reset, GetStatus, Error handling)
- **Middleware**: 8 tests (RateLimit, KeyFunc, APIKey, CustomHandler)
- **Total**: 20 tests, semua PASS ✓

## References

- [Leaky Bucket Algorithm](https://en.wikipedia.org/wiki/Leaky_bucket)
- [Gin Web Framework](https://gin-gonic.com/)
- [Redis Documentation](https://redis.io/documentation)
- [HTMX Documentation](https://htmx.org/)
