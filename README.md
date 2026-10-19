# auth-service

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

Lokal sozlamalar `config/` papkasida: statik sozlamalar `config/.env.local` da (ularni `make` yuklaydi), limitlar va token muddatlari `config/config.yaml` da. Bazada admin bo'lmasa, server `admin@localhost` adminini yaratadi. Xatlar Mailpit'ga boradi: http://localhost:8025.

## Rejimlar

Rejim `APP_ENV` orqali tanlanadi:

| `APP_ENV` | Qachon | Swagger UI |
|---|---|---|
| `dev` | lokal ishlab chiqish (standart) | yoqilgan |
| `test` | test muhiti | yoqilgan |
| `prod` | production | o'chirilgan, yoqib bo'lmaydi |

Swagger UI: `http://localhost:5000/swagger-ui/`. Fayllar git'da saqlanmaydi, ularni `make swagger-ui` yuklaydi. Swagger UI yoqilgan bo'lsa-yu, fayllar yuklanmagan bo'lsa, server ishga tushmaydi.

## Sozlamalar

Sozlamalar ikki xil (25-qaror, [docs/decisions/system.md](docs/decisions/system.md)):

| | Fayl | Qanday beriladi | O'zgarsa |
|---|---|---|---|
| **Statik** | `config/.env.local` (app, db, jwt, swagger, sign-in, admin, smtp) | environment variable | restart kerak (`make reload`) |
| **Dinamik** | `config/config.yaml` (limitlar, token muddatlari) | YAML fayl, yo'li `CONFIG_FILE` da | server faylni o'zi qayta o'qiydi, restart kerak emas |

Berilmagan qiymatlar uchun standart qiymat olinadi.

Server env faylni o'zi o'qimaydi: lokal rejimda `config/.env.local` ni `make` yuklaydi, prod da o'zgaruvchilar container environment orqali beriladi. `config/config.yaml` ni esa server o'qiydi va kuzatadi. Prod da u Kubernetes ConfigMap'dan ulanadi. Fayl xato bo'lsa, server eski qiymatlar bilan ishlashda davom etadi va xatoni log'ga yozadi.

## Buyruqlar

| Buyruq | Tavsif |
|---|---|
| `make build` | web UI va `bin/auth-service` binary'ni yig'ish |
| `make dev` | live rejim: web http://localhost:5173. Web o'zgarishlari darhol ko'rinadi (Vite), Go kodi o'zgarsa server qayta ishga tushadi (air, `.air.toml`). Emaildagi link ham 5173 ga keladi. Ctrl+C bilan to'xtaydi |
| `make start` | build, kalit yaratish va serverni fonda ishga tushirish (log: `logs/auth-service.log`) |
| `make stop` / `make status` | serverni to'xtatish / holatini ko'rish |
| `make reload` | graceful restart: joriy so'rovlar tugaydi, server yangi env bilan qayta ishga tushadi (`config/config.yaml` uchun kerak emas) |
| `make services-up` / `services-down` / `services-status` | `docker/compose.yaml` dagi servicelar: PostgreSQL, Mailpit (UI: http://localhost:8025) |
| `make keys` | `JWT_PRIVATE_KEY_PATH` ga RSA private key yaratish (mavjud bo'lsa tegmaydi) |
| `make swagger-ui` | Swagger UI fayllarini yuklash |
| `make web-install` | web UI paketlarini o'rnatish (`npm ci`) |
| `make web-dev` | Vite dev server (http://localhost:5173), API so'rovlari Go serverga proxy qilinadi |
| `make web-build` | web UI'ni `web/dist` ga build qilish |
| `make web-types` | `docs/openapi.json` dan `web/src/api/schema.d.ts` ni qayta yaratish |
| `make test` / `make test INTEGRATION=1` | unit testlar / unit + integration (testcontainers, Docker kerak). `config/.env.local` yuklanmaydi |
