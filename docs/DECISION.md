# Qarorlar

Amaldagi arxitektura qarorlari. Bu fayldan oldingi qarorlar amalda emas. Sxemaning manbasi: `migrations/`, API hujjati: `docs/openapi.json`.

Sana: 2026-10-15

## 1. Tizim kim uchun

Ishxona xodimlari uchun ichki tizim. Userlar oldindan kiritiladi. O'zi ro'yxatdan o'tish (sign-up) yo'q.

## 2. Identity — faqat email

- Tizimga kirish faqat email orqali.
- Email `users.email` ustunida saqlanadi (`UNIQUE`, doim kichik harfda). Alohida `identities` jadvali yo'q.

## 3. Parolsiz kirish: faqat magic link

- Parol, TOTP, OTP kod va `credentials` jadvali yo'q.
- User email kiritadi va emailga magic link boradi. Kirishning boshqa yo'li yo'q.
- Har bir so'rov `signin_requests` jadvalida bitta qator. Link tokenining faqat hash'i saqlanadi.
- Link token tasodifiy va uzun (kamida 32 bayt), shuning uchun uni taxmin qilib bo'lmaydi. Link bir martalik va muddati cheklangan.

## 4. Magic link faqat so'rov yuborilgan qurilmada ishlaydi

Bu firibgarlarga (scam) qarshi chora. Firibgar xodimning emailini o'z qurilmasida kiritib, xodimdan "kelgan linkni bosing" deb so'rashi mumkin. Link bosilsa ham, firibgarga hech narsa berilmaydi.

- `POST /auth/sign-in` javobida so'rov yuborgan qurilmaga `device_secret` beriladi. Bazada uning faqat hash'i (`signin_requests.secret_hash`) saqlanadi.
  - Brauzerda bu secret `HttpOnly` cookie bo'ladi (`SameSite=Lax`, shunda emaildagi linkni bosganda ham yuboriladi).
  - Ilova yoki curl/Postman secret'ni o'zi saqlaydi.
- Magic link faqat token va o'sha `device_secret` birga kelsa ishlaydi. Session shu qurilmada yaratiladi.
  - Brauzer: link `GET /auth/sign-in/link?token=...` ga olib boradi, secret cookie'dan olinadi. Tokenlar cookie va redirect orqali beriladi (12-qaror).
  - Ilova: link (universal/deep link) ilovani ochadi, ilova `POST /auth/sign-in/link` ga token va secret'ni yuboradi.
- Link boshqa qurilma yoki brauzerda ochilsa, hech narsa sodir bo'lmaydi. So'rov o'zgarmaydi, faqat "Bu link faqat kirish so'ralgan qurilmada ishlaydi" degan xabar chiqadi.
- Polling va "boshqa qurilmada tasdiqlash" yo'q.

<!--
Keyinchalik qo'shish mumkin (claude.ai kabi): link boshqa qurilmada ochilsa,
o'sha ekranda tasdiqlash kodi ko'rsatiladi va user uni so'rov yuborilgan
qurilmaga kiritadi. Hozircha qo'shilmagan: firibgar "ekrandagi kodni aytib
bering" deb so'rashi mumkin, OTP ham aynan shu sabab bilan olib tashlangan.
-->
- OTP olib tashlangani uchun "kodni aytib bering" turidagi firibgarlik ham yo'q: linkni boshqaga uzatish foyda bermaydi.

## 5. Session = device (Telegram kabi)

- Alohida `devices` jadvali yo'q. Har bir session bitta qurilma hisoblanadi.
- Qurilma ma'lumoti `sessions.device` (`JSONB`) ustunida saqlanadi. U session ochilganda signin so'rovidan ko'chadi (16-qaror).
- Qurilmalar ro'yxati — userning faol sessionlari. Qurilmani tugatish — sessionni tugatish.
- Tugatilgan session `ended_sessions` jadvaliga ko'chiriladi va `sessions` dan o'chiriladi. `sessions` da faqat faol sessionlar qoladi (16-qaror).

## 6. Qurilma ma'lumoti qachon va qanday olinadi

- Qurilma ma'lumoti email kiritish qadamida olinadi va `signin_requests.device` da saqlanadi. Session ochilganda u `sessions.device` ga, session tugaganda `ended_sessions.device` ga ko'chadi.
- Client yubormasa (masalan, curl yoki Postman) ma'lumot `User-Agent` dan olinadi.
- `source` maydoni ma'lumot qayerdan kelganini ko'rsatadi:

```json
{"name": "iPhone 15", "os": "iOS 18", "client": "MyApp 2.4", "source": "client"}
{"name": "", "os": "", "client": "curl 8.5.0", "source": "user_agent"}
```

## 7. Mavjud bo'lmagan yoki o'chirilgan user

- Email bazada bo'lmasa yoki `enabled = false` bo'lsa, `signin_requests` ga yozilmaydi va xat yuborilmaydi.
- Javob baribir bir xil: *"Agar bu email ro'yxatda bo'lsa, link yuborildi"*. Bunday holda ham `device_secret` qaytariladi (tasodifiy, hech narsaga bog'lanmagan). Tashqaridan qaysi email tizimda borligini bilib bo'lmaydi.
- User `enabled = false` qilinsa, uning barcha sessionlari tugatiladi (`ended.reason = 'disabled'`).

## 8. Brute force himoyasi

- **Linkni taxmin qilish:** token uzun va tasodifiy, ustiga `device_secret` ham kerak, shuning uchun taxmin qilib bo'lmaydi. Noto'g'ri link urinishlari ham IP limitiga hisoblanadi.
- **Ko'p so'rov:** bitta userga vaqt birligida cheklangan son so'rov (`user_id, created_at` indeksi orqali), IP bo'yicha ham limit.
- **Login qilgan qurilma:** sessionni tugatish orqali.

## 9. Admin va userlarni kiritish (Keycloak kabi)

- Userlarda `role` bo'ladi: `admin` yoki `user`.
- Birinchi admin server ishga tushganda yaratiladi. Bazada birorta ham admin bo'lmasa, `INITIAL_ADMIN_EMAIL` (va `INITIAL_ADMIN_NAME`) dan foydalaniladi. Admin bor bo'lsa, bu o'zgaruvchilar e'tiborga olinmaydi. `APP_ENV=dev` da `INITIAL_ADMIN_EMAIL` standart qiymati `admin@localhost` (server `make` siz ishga tushirilganda ham ishlashi uchun), test va prod da standart qiymat yo'q.
- Birinchi admin ham boshqalar kabi email orqali magic link bilan kiradi.
- Keyingi userlarni admin API orqali qo'shadi, o'zgartiradi, o'chiradi (`enabled = false`) va ularning sessionlarini tugatadi.

## 10. Limitlar — environment variable orqali

Barcha limitlar env orqali sozlanadi. Sozlanmagan bo'lsa, quyidagi standart qiymatlar olinadi:

| O'zgaruvchi | Standart | Ma'nosi |
|---|---|---|
| `SIGN_IN_REQUEST_TTL` | `10m` | magic link amal qilish muddati |
| `SIGN_IN_USER_LIMIT` | `5` | bitta userga `SIGN_IN_USER_WINDOW` ichida so'rovlar soni |
| `SIGN_IN_USER_WINDOW` | `1h` | |
| `SIGN_IN_IP_LIMIT` | `20` | bitta IP'dan `SIGN_IN_IP_WINDOW` ichida so'rovlar va noto'g'ri link urinishlari soni |
| `SIGN_IN_IP_WINDOW` | `1h` | |
| `ACCESS_TOKEN_TTL` | `15m` | access token muddati |
| `REFRESH_TOKEN_TTL` | `720h` | session (refresh token) muddati |
| `SIGN_IN_REDIRECT_URL` | `/` | brauzerda link ochilgandan keyin yo'naltiriladigan manzil |
| `PUBLIC_URL` | `http://localhost:5000` | tashqaridan ko'rinadigan manzil, emaildagi link shu bilan boshlanadi |

## 11. IP bo'yicha limit auth-service ichida

- IP limit auth-service'ning o'zida, xotirada (in-memory) hisoblanadi. Tashqi servis (Redis va boshqalar) ishlatilmaydi.
- Proxy ortida ishlaganda haqiqiy IP `TRUST_PROXY=true` bo'lsagina olinadi.
- Server qayta ishga tushsa, hisoblagich nolga tushadi.
- Tizim Kubernetes'da bir nechta instance (pod) bo'lib ishlaydi va har bir pod o'zicha hisoblaydi. Shuning uchun amaldagi limit taxminan `SIGN_IN_IP_LIMIT × podlar soni` bo'ladi. Bu qabul qilingan: user limiti (8-qaror) Postgres'da hisoblanadi va hamma podlar uchun umumiy.

## 12. Brauzerda tokenlar: `HttpOnly` cookie va redirect

claude.ai'dagi kabi: link so'rov yuborilgan brauzerda ochilsa, user avtomatik tizimga kiradi.

- `GET /auth/sign-in/link?token=...` token va `device_secret` cookie'sini tekshiradi.
- Ikkalasi mos kelsa:
  - session yaratiladi;
  - refresh token `HttpOnly`, `Secure`, `SameSite=Lax` cookie bo'lib o'rnatiladi, JavaScript uni o'qiy olmaydi;
  - `device_secret` cookie'si o'chiriladi;
  - brauzer `SIGN_IN_REDIRECT_URL` ga yo'naltiriladi.
- Web ilova access tokenni `POST /auth/refresh` orqali oladi, refresh token cookie'dan olinadi. Access token faqat xotirada saqlanadi.
- Mos kelmasa (boshqa qurilma, muddati o'tgan yoki ishlatilgan link), brauzer `SIGN_IN_REDIRECT_URL` ga xato belgisi bilan yo'naltiriladi va session yaratilmaydi.
- Ilova va curl'da hech narsa o'zgarmaydi: `POST /auth/sign-in/link` tokenlarni body'da qaytaradi, `POST /auth/refresh` refresh tokenni body'dan oladi.

## 13. Email: SMTP orqali, auth-service'ning o'zi yuboradi

- Xatni auth-service'ning o'zi yuboradi, broker yoki alohida mail servis ishlatilmaydi. SMTP sozlamalari env orqali beriladi: `SMTP_HOST`, `SMTP_PORT` (`587`), `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`.
- `SMTP_HOST` bo'sh bo'lsa (dev/test), xat yuborilmaydi, link logga yoziladi. `prod` da `SMTP_HOST` majburiy.
- Xat fonda yuboriladi, shunda javob vaqtidan email tizimda bor-yo'qligini bilib bo'lmaydi.

## 14. ID va tozalash

- ID'larni ilova **Sonyflake** usulida yaratadi (`github.com/sony/sonyflake`). Tizim Kubernetes'da bir nechta pod bo'lib ishlaydi va ID butun klaster bo'ylab noyob bo'lishi kerak.
- ID tuzilishi (63 bit):

| Qism | Bit | Ma'nosi |
|---|---|---|
| vaqt | 39 | boshlang'ich sanadan beri 10 ms birliklarda, ~174 yil yetadi |
| ketma-ketlik | 8 | bitta pod 10 ms ichida 256 tagacha ID beradi (~25 600 ID/soniya) |
| machine ID | 16 | podning private IP manzilining oxirgi 16 biti |

- Machine ID podning IP manzilidan olinadi. Kubernetes'da har bir pod klaster ichida o'z IP'siga ega, shuning uchun ID'lar butun klaster uchun noyob bo'ladi.
- Shart: pod tarmog'i (pod CIDR) `/16` dan katta bo'lmasligi kerak. Aks holda ikki podning IP oxirgi 16 biti bir xil chiqib qolishi mumkin. Klaster sozlanayotganda shu tekshiriladi.
- Server har soatda muddati o'tgan sessionlarni `ended_sessions` ga ko'chiradi (`ended.reason = 'expired'`) va ishlatilmagan eski signin so'rovlarini o'chiradi (16-qaror).

## 15. Rejimlar: dev, test va prod

`APP_ENV` uchta qiymatni qabul qiladi:

| Rejim | Nima uchun | Sozlama qayerdan |
|---|---|---|
| `dev` | lokal ishlab chiqish (`make start`) | `.env.local` fayli (`make` yuklaydi): o'zimiz ko'rsatgan servislar (Postgres, SMTP) |
| `test` | avtomatik testlar (`make test`) | `.env.local` yuklanmaydi, hammasi localhost. Integration testlarga kerakli servislarni testcontainers ko'taradi |
| `prod` | Kubernetes, tafsilotlari hali aniqlanmagan | container environment. Prod tekshiruvlari himoya sifatida qoladi: Swagger taqiqlangan, `SMTP_HOST` majburiy |

- `make test` hech qanday env faylga bog'liq emas va dev bazasiga ulanmaydi.
- Server env fayl o'qimaydi. Repoda faqat `.env.local` (lokal dev sozlamalari) saqlanadi, uni `make` yuklaydi. `.env` fayli ishlatilmaydi.

### Unit va integration testlar

Integration testlar build tag bilan ajratiladi:

- Unit testlar kod yonida, oddiy `*_test.go` fayllarda. Tashqi servis ishlatmaydi.
- Integration testlar alohida `test/integration/` papkasida. Ular tizimni tashqaridan, client kabi tekshiradi: HTTP so'rov yuboradi va javobni tekshiradi (sign-in → link → refresh → logout, admin amallari va hokazo).
- Har bir integration fayl boshida `//go:build integration` bo'ladi. Tag bo'lmasa `go test ./...` bu papkani ham ishga tushirib, Docker talab qilib qolardi.
- Bitta `TestMain` testcontainers orqali Postgres va Mailpit'ni ko'taradi, barcha integration testlar shulardan foydalanadi. Buning uchun Docker kerak.
- Integration testlar `Config` ni env orqali emas, struct sifatida o'zi yig'adi: baza va SMTP manzillari faqat konteynerlardan olinadi. `make test` esa `.env.local` dagi o'zgaruvchilarni test process'iga bermaydi.
- Testlar serverni import qila olishi uchun server `main.go` dan exported funksiyaga chiqariladi (masalan, `app.New(cfg)`).

```
test/integration/
├── main_test.go        TestMain: Postgres, Mailpit konteynerlari va server
├── signin_test.go
├── sessions_test.go
└── admin_test.go
```

Flag bo'lmasa faqat unit testlar ishlaydi. Integration testlar flag bilan yoqiladi:

```sh
make test                  # go test ./...                     faqat unit
make test INTEGRATION=1    # go test -tags=integration ./...   unit + integration
```

## 16. Jadvallar uchligi: `signin_requests` → `sessions` → `ended_sessions`

Ma'lumot jadvaldan jadvalga ko'chib o'tadi. Har bir bosqichda kerakli ma'lumot keyingi jadvalga nusxalanadi va oldingi jadvaldan o'chiriladi. Shuning uchun bitta qator bir vaqtda faqat bitta jadvalda bo'ladi.

| Jadval | Nima saqlanadi | Qachon o'chadi |
|---|---|---|
| `signin_requests` | kutilayotgan so'rov: link token hash, `device_secret` hash, qurilma, `requested_ip`, `requested_at` | link ishlatilib session ochilganda; ishlatilmasa, tozalashda |
| `sessions` | faol session: refresh token hash, qurilma, `started_ip` / `started_at`, `current_ip` / `current_at`, muddat | session tugaganda |
| `ended_sessions` | arxiv: `sessions` ustunlari (`token_hash` dan tashqari) va bitta `ended` JSON ustuni (qachon, nima sababdan, kim va qaysi session tugatgani) | o'chmaydi |

- **Bitta ID:** `signin_requests.id = sessions.id = ended_sessions.id`. ID ilovada yaratiladi (14-qaror) va jadvaldan jadvalga o'zgarmasdan ko'chadi. JWT'dagi `sid` ham shu ID. Jadvallar o'rtasida FK yo'q, chunki oldingi qator har doim o'chirilgan bo'ladi.
- **Session ochilganda** (link ishlatilganda) bitta SQL buyrug'i bajariladi: `signin_requests` dan `DELETE ... RETURNING`, so'ng `sessions` ga `INSERT`. `id` va `device` ko'chadi, `started_ip` va `started_at` linkni ochgan paytdagi IP va vaqt bo'ladi. So'rov o'chirilgani uchun link ikkinchi marta ishlamaydi, alohida `used_at` ustuni kerak emas.
- **Session tugaganda** ham bitta SQL buyrug'i bajariladi: `sessions` dan `DELETE ... RETURNING`, so'ng `ended_sessions` ga `INSERT`. `ended_sessions` ustunlari `sessions` dagi bilan bir xil nomda (`token_hash` dan tashqari). Ko'chirish SQL'i ustunlarni aniq sanab yozadi (`internal/session/service.go`). `sessions` ga yangi ustun qo'shilsa, u `ended_sessions` ga va shu SQL'ga ham qo'shiladi.
- **Tugash ma'lumotlari** bitta `ended` JSON ustunida saqlanadi:

  ```json
  {"at": "2026-10-20T18:30:00+05:00", "reason": "terminated", "by": 587364927401230337, "by_session": 587364927401230999}
  ```

  - `at` — qachon tugadi;
  - `reason` — nima sababdan: `logout`, `terminated`, `expired`, `disabled` (DB'da `CHECK` bilan tekshiriladi);
  - `by` — kim tugatdi (user ID); tizim tugatgan bo'lsa (`expired`) `null`;
  - `by_session` — qaysi session (qurilma) orqali tugatildi, masalan boshqa qurilmadan "terminate" bosilganda o'sha qurilmaning session ID'si; tizim tugatgan bo'lsa `null`.
- **Arxivga ko'chmaydigan ma'lumotlar:** so'rov vaqtidagi `requested_ip` va `requested_at`. Ular faqat `signin_requests` da yashaydi va session ochilganda yo'qoladi. `token_hash` ham arxivga ko'chmaydi: tugagan sessionning refresh tokeni hech kimga kerak emas.
- **Jadval oddiy, `INHERITS` ishlatilmaydi:** aks holda `SELECT FROM sessions` arxivdagi qatorlarni ham qaytarardi.
- **Ishlatilmagan signin so'rovlari** (link bosilmagan, muddati o'tgan yoki rad etilgan) session'ga o'tmaydi, `ended_sessions` ga ham yozilmaydi. Tozalash ularni o'chiradi. Qatorlar user limiti oynasi davomida saqlanadi, chunki limit ularni sanaydi.
- **User limiti** (8-qaror) `signin_requests` dagi qatorlarni sanaydi. Ishlatilgan so'rovlar darhol o'chgani uchun limit faqat kutilayotgan va ishlatilmagan so'rovlarni hisoblaydi. Bu yetarli: limit email spam'iga qarshi, muvaffaqiyatli kirish esa spam emas.
- **Ustun nomlari:** so'rov vaqti `requested_*`, session boshlanishi `started_*`, sessionning hozirgi holati `current_*` (IP va vaqt birga yangilanadi). Arxivda ham shu nomlar qoladi.
- **ID ichidagi vaqt** email kiritilgan paytni bildiradi. Session boshlangan aniq vaqt `started_at` da.

## 17. API `/api/v1` ostida, qolgan yo'llar web UI

auth-service o'z web UI'si bilan birga keladi (Keycloak kabi): login sahifalari va admin konsol bitta binary ichida.

- Barcha API endpointlari `/api/v1` ostida. Cookie path'lari ham shu prefiks bilan: `device_secret` uchun `/api/v1/auth/sign-in`, refresh token uchun `/api/v1/auth`.
- Qolgan GET yo'llarga web UI javob beradi. Fayl bo'lmagan yo'lga `index.html` qaytadi, routing'ni React bajaradi. `/api/v1` ostidagi noma'lum yo'l esa JSON 404 bo'lib qoladi.
- `/.well-known/*` prefikssiz: discovery issuer manzili yonida bo'lishi kerak (OpenID standarti). `/swagger-ui/` ham prefikssiz, u API emas.
- Web: React, Vite, Tailwind va shadcn/ui. `web/dist` `embed` orqali binary ichiga kiradi, Node faqat build vaqtida kerak.

## Jadvallar

| Jadval | Vazifasi |
|---|---|
| `users` | xodimlar (email, ism, `role`, `enabled`) |
| `signin_requests` | kutilayotgan magic link so'rovlari va qurilma ma'lumoti |
| `sessions` | faol sessionlar = qurilmalar |
| `ended_sessions` | tugagan sessionlar arxivi: session ma'lumoti va qachon, nima uchun, kim tomonidan tugatilgani |

## Ochiq savollar

- Sonyflake ID 63 bitli, JavaScript esa faqat 53 bitgacha aniq o'qiydi. JSON javoblarda ID qanday beriladi: string sifatida (`"id": "123..."`) yoki son sifatida qoldiriladi?
- Prod Kubernetes'da ishlaydi. Qolgan tafsilotlar (qaysi SMTP server, ingress, secretlar) hali aniqlanmagan.
