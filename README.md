# Rate Limiting API

Implementasi **Leaky Bucket Algorithm** untuk rate limiting API dengan Go + Redis + HTMX Dashboard.

## Quick Start

```bash
# 1. Start Redis
docker run -d -p 6379:6379 redis

# 2. Run server
go run main.go

# 3. Access dashboard
open http://localhost:8080/dashboard/

# 4. Test rate limiting (limit: 10 requests)
for i in {1..15}; do curl http://localhost:8080/api/ping; done
```

## Fitur

✅ **Leaky Bucket Algorithm** - Request rate limiting   
✅ **Redis Backend** - Distributed state management   
✅ **Gin Middleware** - Easy integration dengan routes   
✅ **Interactive Dashboard** - Monitor & manage rate limits   
✅ **Flexible Key Functions** - By IP, API Key, User ID   
✅ **20+ Unit Tests** - Full test coverage   

## Endpoints

### Dashboard
- `GET /dashboard/` - Main dashboard
- `GET /dashboard/status` - Your rate limit status
- `GET /dashboard/keys` - All active keys
- `POST /dashboard/reset` - Reset rate limit
- `POST /dashboard/test` - Test request

### API (Rate Limited)
- `GET /api/ping` - Simple ping
- `GET /api/data` - Protected data (rate limited)

### Health
- `GET /health` - Health check

## Configuration

Edit `main.go`:
```go
// Capacity: 10 requests, LeakRate: 2/sec, TTL: 1 hour
rateLimiter := limiter.NewLeakyBucket(10, 2, time.Hour)
```

## How It Works

```
Request → Bucket [████████░░] → Allow/Deny
                      ↓↓ Leak (2/sec) ↓↓
```

- Bucket kapasitas **10 requests**
- Setiap request mengisi **1 unit**
- Bucket bocor **2 units/sec**
- Jika penuh → **429 Too Many Requests**

## Documentation

Lihat [DOCUMENTATION.md](DOCUMENTATION.md) untuk:
- Setup dan instalasi lengkap
- Error handling & troubleshooting
- Custom configuration
- Production deployment
- Implementasi algoritma lain

## Test

```bash
cd internal/limiter && go test -v
cd internal/middleware && go test -v
```

All 20 tests pass ✓

## Technology Stack

- **Go 1.20+** - Backend
- **Gin** - Web framework
- **Redis** - State storage
- **HTMX** - Dynamic frontend
- **Tailwind CSS** - Styling

## License

MIT

