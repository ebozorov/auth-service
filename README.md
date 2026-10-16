# auth-api

Ishxona xodimlari uchun autentifikatsiya API. Parol yo'q: xodim email kiritadi va emailga kelgan magic link orqali kiradi. Har bir session bitta qurilma hisoblanadi, uni istalgan vaqtda tugatish mumkin.

## Hujjatlar

| Fayl | Mazmuni |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | arxitektura: kirish oqimi, endpointlar, kod tuzilmasi, JWT, ma'lumotlar bazasi |
| [docs/DECISION.md](docs/DECISION.md) | arxitektura qarorlari va ularning sabablari |
| [docs/openapi.json](docs/openapi.json) | API hujjati (OpenAPI 3) |

## Talablar

- Go 1.26+
- Node.js 22+ (faqat web UI build uchun)
- Docker (PostgreSQL va Mailpit uchun)
- `make`, `openssl`, `curl`

## Tez boshlash

```sh
make services-up       # tashqi servicelar (docker/compose.yaml)
make web-install       # web UI paketlari (bir marta)
make swagger-ui        # Swagger UI fayllari (bir marta, dev/test da kerak)
make start             # build + kalit + server: web http://localhost:5000, API http://localhost:5000/api/v1
```

Server ishga tushganda migrationlar avtomatik qo'llanadi. To'xtatish: `make stop`.

Lokal sozlamalar `.env.local` da, ularni `make` yuklaydi. Bazada admin bo'lmasa, server `admin@localhost` adminini yaratadi. Xatlar Mailpit'ga boradi: http://localhost:8025.

## Rejimlar

Rejim `APP_ENV` orqali tanlanadi:

| `APP_ENV` | Qachon | Swagger UI |
|---|---|---|
| `dev` | lokal ishlab chiqish (standart) | yoqilgan |
| `test` | test muhiti | yoqilgan |
| `prod` | production | o'chirilgan, yoqib bo'lmaydi |

Swagger UI: `http://localhost:5000/swagger-ui/`. Fayllar git'da saqlanmaydi, ularni `make swagger-ui` yuklaydi. Swagger UI yoqilgan bo'lsa-yu, fayllar yuklanmagan bo'lsa, server ishga tushmaydi.

## Sozlamalar

Barcha sozlamalar environment variable orqali beriladi. Ro'yxati `.env.local` da (app, db, jwt, swagger, sign-in, admin, smtp guruhlari). Berilmagan qiymatlar uchun standart qiymat olinadi. Limitlar va sign-in sozlamalari: [docs/DECISION.md](docs/DECISION.md).

Server hech qanday env faylni o'zi o'qimaydi. Lokal rejimda `.env.local` ni `make` yuklaydi, prod da o'zgaruvchilar container environment orqali beriladi.

## Buyruqlar

| Buyruq | Tavsif |
|---|---|
| `make build` | web UI va `bin/auth-api` binary'ni yig'ish |
| `make start` | build, kalit yaratish va serverni fonda ishga tushirish (log: `logs/auth-api.log`) |
| `make stop` / `make status` | serverni to'xtatish / holatini ko'rish |
| `make reload` | graceful restart: joriy so'rovlar tugaydi, server yangi env bilan qayta ishga tushadi |
| `make services-up` / `services-down` / `services-status` | `docker/compose.yaml` dagi servicelar: PostgreSQL, Mailpit (UI: http://localhost:8025) |
| `make keys` | `JWT_PRIVATE_KEY_PATH` ga RSA private key yaratish (mavjud bo'lsa tegmaydi) |
| `make swagger-ui` | Swagger UI fayllarini yuklash |
| `make web-install` | web UI paketlarini o'rnatish (`npm ci`) |
| `make web-dev` | Vite dev server (http://localhost:5173), API so'rovlari Go serverga proxy qilinadi |
| `make web-build` | web UI'ni `web/dist` ga build qilish |
| `make web-types` | `docs/openapi.json` dan `web/src/api/schema.d.ts` ni qayta yaratish |
| `make test` / `make test INTEGRATION=1` | unit testlar / unit + integration (testcontainers, Docker kerak). `.env.local` yuklanmaydi |
