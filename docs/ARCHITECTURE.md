# Arxitektura

auth-service qanday tuzilgani: kirish oqimi, endpointlar, kod tuzilmasi, JWT va ma'lumotlar bazasi. Nima uchun aynan shunday qilingani: [DECISION.md](DECISION.md). API'ning to'liq tavsifi: [openapi.json](openapi.json).

## Kirish oqimi

1. Qurilma `POST /auth/sign-in` ga email (va bo'lsa, qurilma ma'lumotini) yuboradi. Javobda `device_secret` qaytadi (brauzerda `HttpOnly` cookie).
2. Emailga magic link boradi.
3. Link faqat so'rov yuborilgan qurilmada ishlaydi, chunki token `device_secret` bilan birga tekshiriladi. Boshqa joyda ochilsa hech narsa bo'lmaydi. Bu firibgarlarga qarshi chora.
   - **Brauzer:** `GET /auth/sign-in/link?token=...`, secret cookie'dan olinadi. Muvaffaqiyatda refresh token `HttpOnly` cookie bo'lib o'rnatiladi va brauzer `SIGN_IN_REDIRECT_URL` ga yo'naltiriladi. Web ilova access tokenni `POST /auth/refresh` dan oladi.
   - **Ilova yoki curl:** `POST /auth/sign-in/link`, token va secret body'da, tokenlar body'da qaytadi.

Email ro'yxatda bo'lmasa ham javob bir xil qaytadi, faqat xat yuborilmaydi.

## Endpointlar

Barcha API endpointlari `/api/v1` prefiksi ostida: quyidagi jadvallarda prefiks yozilmagan, masalan `/auth/sign-in` aslida `/api/v1/auth/sign-in`. Qolgan yo'llar web UI'ga tegishli (19 va 20-qarorlar).

### Auth

| Metod | Endpoint | Tavsif |
|---|---|---|
| `POST` | `/auth/sign-in` | Email kiritish, magic link yuborish |
| `GET` | `/auth/sign-in/link?token=...` | Magic link (brauzer): session yaratadi, refresh token cookie'ga, redirect |
| `POST` | `/auth/sign-in/link` | Magic link (ilova/curl, token va secret body'da), session yaratadi |
| `POST` | `/auth/refresh` | Tokenlarni yangilash (refresh token cookie'dan yoki body'dan) |
| `POST` | `/auth/logout` | Joriy sessionni tugatish, refresh token cookie'sini o'chirish |
| `GET` | `/auth/me` | Joriy foydalanuvchi |

### Admin

Faqat `role = admin` uchun. Birinchi admin `INITIAL_ADMIN_EMAIL` dan yaratiladi.

| Metod | Endpoint | Tavsif |
|---|---|---|
| `GET` | `/admin/users` | Userlar ro'yxati |
| `POST` | `/admin/users` | User qo'shish (email, ism, rol) |
| `PATCH` | `/admin/users/{id}` | Ism, rol, `enabled` ni o'zgartirish |
| `GET` | `/admin/users/{id}/sessions` | Userning qurilmalari |
| `DELETE` | `/admin/users/{id}/sessions/{sid}` | Userning qurilmasini tugatish |

### Sessionlar

Session = qurilma. Ro'yxatda har bir qurilmaning nomi, IP va oxirgi faollik vaqti ko'rinadi.

| Metod | Endpoint | Tavsif |
|---|---|---|
| `GET` | `/sessions` | Faol qurilmalar ro'yxati |
| `DELETE` | `/sessions/{id}` | Bitta qurilmani tugatish |
| `DELETE` | `/sessions` | Joriydan boshqa barcha qurilmalarni tugatish |

### Public

Swagger'da ko'rinmaydi.

| Metod | Endpoint | Tavsif |
|---|---|---|
| `GET` | `/api/v1/` | API haqida ma'lumot, health, uptime |
| `GET` | `/api/v1/openapi.json` | API hujjati |
| `GET` | `/.well-known/jwks.json` | JWT tekshirish uchun public key (prefikssiz) |
| `GET` | `/.well-known/openid-configuration` | Issuer va `jwks_uri` (prefikssiz) |
| `GET` | `/swagger-ui/` | Swagger UI, faqat `dev` / `test` (prefikssiz) |

## Tuzilma

Package by feature: har bir endpoint guruhi o'z papkasida.

```
cmd/server/main.go     kirish nuqtasi: config yuklaydi, app.New, signal kutadi
internal/
├── app/               serverni yig'adi (app.New): baza, migrationlar, route'lar
├── config.go          statik sozlamalar (env)
├── dynamic.go         dinamik sozlamalar (config/config.yaml, o'zgarsa qayta o'qiladi)
├── database.go        ulanish, migrationlar
├── identity.go        joriy user va session (request ichida)
├── id.go              ID yaratish
├── token.go           tasodifiy token va uning hash'i
├── api.go             API prefiksi (/api/v1)
├── auth/              /auth: sign-in, magic link, refresh, JWT, IP limit
├── session/           /sessions: qurilmalar, arxivga ko'chirish
├── admin/             /admin: userlar va ularning qurilmalari
├── user/              users jadvali (handler'siz)
├── mailer/            SMTP orqali email
└── home/              /api/v1/, /api/v1/openapi.json, /.well-known/*, /swagger-ui/
web/                   web UI (React): build web/dist ga tushadi va binary ichiga embed qilinadi
docs/
├── ARCHITECTURE.md    arxitektura (shu fayl)
├── DECISION.md        arxitektura qarorlari ro'yxati
├── decisions/         qarorlar kategoriya bo'yicha (system, flows, api, database, models, web)
└── openapi.json       API hujjati
migrations/            SQL migrationlar (sxemaning manbasi)
test/integration/      integration testlar, alohida Go moduli (testcontainers)
config/                .env.local (statik, make yuklaydi), config.yaml (dinamik, server o'qiydi)
docker/                compose.yaml (lokal servicelar)
swagger-ui/            Swagger UI fayllari
certs/                 RSA private key (standart joyi)
bin/                   binary, pid
logs/                  loglar
```

## Swagger UI

Fayllar `swagger-ui/` papkasida (`make swagger-ui`), `docs/openapi.json` ni ko'rsatadi.

| `APP_ENV` | `SWAGGER_UI_ENABLED` standart qiymati |
|---|---|
| `dev`, `test` | `true` |
| `prod` | `false` (yoqib bo'lmaydi) |

## JWT

Access token `RS256` bilan imzolanadi. Boshqa servislar tokenni `/.well-known/jwks.json` dagi public key bilan tekshiradi:

- `alg` = `RS256`
- `kid` — JWKS dagi kalit bilan mos
- `iss` = `JWT_ISSUER`
- `exp` — muddati o'tmagan

Private key `JWT_PRIVATE_KEY_PATH` da (standart `certs/private.pem`), `make keys` yaratadi. Public key alohida faylda saqlanmaydi: server uni private key'dan oladi va `/.well-known/jwks.json` da beradi.

## Ma'lumotlar bazasi

Sxemaning manbasi: `migrations/`. Migrationlar server ishga tushganda avtomatik qo'llanadi.

| Jadval | Vazifasi |
|---|---|
| `users` | xodimlar: email, ism, `role`, `enabled` |
| `signin_requests` | kutilayotgan magic link so'rovlari; link ishlatilganda `sessions` ga ko'chadi |
| `sessions` | faol sessionlar = qurilmalar (`device JSONB`); tugaganda `ended_sessions` ga ko'chadi |
| `ended_sessions` | tugagan sessionlar va qanday tugagani (`ended JSONB`) |

Ma'lumot jadvaldan jadvalga bitta ID bilan ko'chadi (DECISION.md, 16-qaror). Ikkala ko'chirish SQL'i `internal/session/service.go` da: `Start` va `archive`.
