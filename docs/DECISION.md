# Qarorlar

Amaldagi arxitektura qarorlari. Bu fayldan oldingi qarorlar amalda emas. Sxemaning manbasi: `migrations/`, API hujjati: `docs/openapi.json`.

Sana: 2026-10-16

Qarorlar kategoriya bo'yicha [`decisions/`](decisions/) papkasida yuritiladi. Bu fayl ularning ro'yxati.

- Raqamlar global va o'zgarmaydi: kod va hujjatlardagi `DECISION.md #N` havolalari shu ro'yxat orqali topiladi.
- Yangi qaror keyingi bo'sh raqamni oladi, o'z kategoriyasi fayliga yoziladi va quyidagi ro'yxatga qo'shiladi.
- Qaror bir nechta kategoriyaga tegishli bo'lsa, qismlarga ajratiladi va qismlar bir-biriga havola beradi.

## Kategoriyalar

| Fayl | Nima haqida |
|---|---|
| [system.md](decisions/system.md) | tizim: loyiha tuzilishi, ishga tushirish, sozlamalar, infratuzilma |
| [flows.md](decisions/flows.md) | flow: sign-in oqimi, qurilmaga bog'lash, himoya |
| [api.md](decisions/api.md) | API: endpointlar, yo'llar, JSON formati |
| [database.md](decisions/database.md) | database: jadvallar, ustunlar, view'lar |
| [models.md](decisions/models.md) | model: Go kodidagi `type ... struct` turlari |
| [web.md](decisions/web.md) | web (UI): sahifalar, routing, stek |

## Barcha qarorlar

| # | Qaror | Kategoriya |
|---|---|---|
| 1 | Tizim kim uchun | [system](decisions/system.md) |
| 2 | Identity — faqat email | [flows](decisions/flows.md) |
| 3 | Parolsiz kirish: faqat magic link | [flows](decisions/flows.md) |
| 4 | Magic link faqat so'rov yuborilgan qurilmada ishlaydi | [flows](decisions/flows.md) |
| 5 | Session = device (Telegram kabi) | [database](decisions/database.md) |
| 6 | Qurilma ma'lumoti qachon va qanday olinadi | [flows](decisions/flows.md) |
| 7 | Mavjud bo'lmagan yoki o'chirilgan user | [flows](decisions/flows.md) |
| 8 | Brute force himoyasi | [flows](decisions/flows.md) |
| 9 | Admin va userlarni kiritish (Keycloak kabi) | [system](decisions/system.md) |
| 10 | Limitlar — environment variable orqali | [system](decisions/system.md) |
| 11 | IP bo'yicha limit auth-service ichida | [system](decisions/system.md) |
| 12 | Brauzerda tokenlar: `HttpOnly` cookie va redirect | [flows](decisions/flows.md) |
| 13 | Email: SMTP orqali, auth-service'ning o'zi yuboradi | [system](decisions/system.md) |
| 14 | ID: Sonyflake | [system](decisions/system.md) |
| 15 | Rejimlar: dev, test va prod | [system](decisions/system.md) |
| 16 | Jadvallar uchligi: `signin_requests` → `sessions` → `ended_sessions` | [database](decisions/database.md) |
| 17 | API va web UI bitta binary ichida | [system](decisions/system.md) |
| 18 | Dashboard: hamma uchun | [web](decisions/web.md) |
| 19 | API `/api/v1` ostida | [api](decisions/api.md) |
| 20 | Web UI routing va stek | [web](decisions/web.md) |
| 21 | ID'lar JSON'da string | [api](decisions/api.md) |
| 22 | ID turlari: `int64` va `internal.ID` | [models](decisions/models.md) |
| 23 | Dashboard ma'lumotlari materialized view'da | [database](decisions/database.md) |

## Ochiq savollar

Har bir kategoriya faylining oxirida:

- [system.md](decisions/system.md#ochiq-savollar) — prod Kubernetes tafsilotlari.
- [database.md](decisions/database.md#ochiq-savollar) — dashboard view'larini yangilash.
