# Tizim bo'yicha qarorlar

Loyiha tuzilishi, ishga tushirish, sozlamalar va infratuzilma. Barcha qarorlar ro'yxati: [DECISION.md](../DECISION.md).

## 1. Tizim kim uchun

Ishxona xodimlari uchun ichki tizim. Userlar oldindan kiritiladi. O'zi ro'yxatdan o'tish (sign-up) yo'q.

## 9. Admin va userlarni kiritish (Keycloak kabi)

- Har bir userda bitta rol bo'ladi (`users.role`): `admin` (to'liq ruxsat) yoki `user`. Rollar hardcoded, permissionlar bu servisda yo'q: 24-qaror ([database](database.md)).
- Birinchi admin server ishga tushganda yaratiladi. Bazada birorta ham admin bo'lmasa, `INITIAL_ADMIN_EMAIL` (va `INITIAL_ADMIN_NAME`) dan foydalaniladi. Admin bor bo'lsa, bu o'zgaruvchilar e'tiborga olinmaydi. `APP_ENV=dev` da `INITIAL_ADMIN_EMAIL` standart qiymati `admin@localhost` (server `make` siz ishga tushirilganda ham ishlashi uchun), test va prod da standart qiymat yo'q.
- Birinchi admin ham boshqalar kabi email orqali magic link bilan kiradi.
- Keyingi userlarni admin API orqali qo'shadi, o'zgartiradi, o'chiradi (`enabled = false`) va ularning sessionlarini tugatadi.

## 10. Limitlar — environment variable orqali

Barcha limitlar env orqali sozlanadi. Sozlanmagan bo'lsa, quyidagi standart qiymatlar olinadi. Jadvaldagi limitlar va muddatlar (`SIGN_IN_REQUEST_TTL` … `REFRESH_TOKEN_TTL`) 25-qaror bilan dinamik sozlamalar fayliga ko'chadi, `SIGN_IN_REDIRECT_URL` va `PUBLIC_URL` env'da qoladi:

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
- Tizim Kubernetes'da bir nechta instance (pod) bo'lib ishlaydi va har bir pod o'zicha hisoblaydi. Shuning uchun amaldagi limit taxminan `SIGN_IN_IP_LIMIT × podlar soni` bo'ladi. Bu qabul qilingan: user limiti (8-qaror, [flows](flows.md)) Postgres'da hisoblanadi va hamma podlar uchun umumiy.

## 13. Email: SMTP orqali, auth-service'ning o'zi yuboradi

- Xatni auth-service'ning o'zi yuboradi, broker yoki alohida mail servis ishlatilmaydi. SMTP sozlamalari env orqali beriladi: `SMTP_HOST`, `SMTP_PORT` (`587`), `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`.
- `SMTP_HOST` bo'sh bo'lsa (dev/test), xat yuborilmaydi, link logga yoziladi. `prod` da `SMTP_HOST` majburiy.
- Xat fonda yuboriladi, shunda javob vaqtidan email tizimda bor-yo'qligini bilib bo'lmaydi.

## 14. ID: Sonyflake

- ID'larni ilova **Sonyflake** usulida yaratadi (`github.com/sony/sonyflake`). Tizim Kubernetes'da bir nechta pod bo'lib ishlaydi va ID butun klaster bo'ylab noyob bo'lishi kerak.
- ID tuzilishi (63 bit):

| Qism | Bit | Ma'nosi |
|---|---|---|
| vaqt | 39 | boshlang'ich sanadan beri 10 ms birliklarda, ~174 yil yetadi |
| ketma-ketlik | 8 | bitta pod 10 ms ichida 256 tagacha ID beradi (~25 600 ID/soniya) |
| machine ID | 16 | podning private IP manzilining oxirgi 16 biti |

- Machine ID podning IP manzilidan olinadi. Kubernetes'da har bir pod klaster ichida o'z IP'siga ega, shuning uchun ID'lar butun klaster uchun noyob bo'ladi.
- Shart: pod tarmog'i (pod CIDR) `/16` dan katta bo'lmasligi kerak. Aks holda ikki podning IP oxirgi 16 biti bir xil chiqib qolishi mumkin. Klaster sozlanayotganda shu tekshiriladi.
- ID qiymati 2^53 dan katta, shuning uchun API'da string bo'lib yuboriladi (21-qaror, [api](api.md)). Go kodidagi turlari: 22-qaror ([models](models.md)).

## 15. Rejimlar: dev, test va prod

`APP_ENV` uchta qiymatni qabul qiladi:

| Rejim | Nima uchun | Sozlama qayerdan |
|---|---|---|
| `dev` | lokal ishlab chiqish (`make start`) | `config/.env.local` fayli (`make` yuklaydi): o'zimiz ko'rsatgan servislar (Postgres, SMTP). Dinamik sozlamalar: `config/config.yaml` (25-qaror) |
| `test` | avtomatik testlar (`make test`) | `config/.env.local` yuklanmaydi, hammasi localhost. Integration testlarga kerakli servislarni testcontainers ko'taradi |
| `prod` | Kubernetes, tafsilotlari hali aniqlanmagan | container environment. Prod tekshiruvlari himoya sifatida qoladi: Swagger taqiqlangan, `SMTP_HOST` majburiy |

- `make test` hech qanday env faylga bog'liq emas va dev bazasiga ulanmaydi.
- Server env fayl o'qimaydi. Repoda faqat `config/.env.local` (lokal dev sozlamalari) saqlanadi, uni `make` yuklaydi. `.env` fayli ishlatilmaydi. Server o'zi o'qiydigan yagona fayl — dinamik sozlamalar fayli (25-qaror).

### Unit va integration testlar

Integration testlar build tag bilan ajratiladi:

- Unit testlar kod yonida, oddiy `*_test.go` fayllarda. Tashqi servis ishlatmaydi.
- Integration testlar alohida `test/integration/` papkasida. Ular tizimni tashqaridan, client kabi tekshiradi: HTTP so'rov yuboradi va javobni tekshiradi (sign-in → link → refresh → logout, admin amallari va hokazo).
- Har bir integration fayl boshida `//go:build integration` bo'ladi. Tag bo'lmasa `go test ./...` bu papkani ham ishga tushirib, Docker talab qilib qolardi.
- Bitta `TestMain` testcontainers orqali Postgres va Mailpit'ni ko'taradi, barcha integration testlar shulardan foydalanadi. Buning uchun Docker kerak.
- Integration testlar `Config` ni env orqali emas, struct sifatida o'zi yig'adi: baza va SMTP manzillari faqat konteynerlardan olinadi. `make test` esa `config/.env.local` dagi o'zgaruvchilarni test process'iga bermaydi.
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

## 17. API va web UI bitta binary ichida

auth-service o'z web UI'si bilan birga keladi (Keycloak kabi): login sahifalari va admin konsol bitta binary ichida.

- `web/dist` `embed` orqali binary ichiga kiradi, Node faqat build vaqtida kerak.
- API yo'llari: 19-qaror ([api](api.md)). Web UI routing'i va steki: 20-qaror ([web](web.md)).

## 25. Dinamik sozlamalar: fayl va qayta yuklash

Ba'zi sozlamalarni server qayta ishga tushmasdan o'zgartirish kerak, masalan hujum paytida limitni pasaytirish. Buning uchun sozlamalar ikki guruhga bo'linadi.

| Guruh | Qayerda | O'zgarsa |
|---|---|---|
| **Statik** | env (10-, 13-, 15-qarorlar) | restart kerak |
| **Dinamik** | YAML fayl | server faylni qayta o'qiydi, restart kerak emas |

- **Dinamik sozlamalar** — faqat limitlar va muddatlar:

| Kalit | Standart | Oldingi env |
|---|---|---|
| `sign_in_request_ttl` | `10m` | `SIGN_IN_REQUEST_TTL` |
| `sign_in_user_limit` | `5` | `SIGN_IN_USER_LIMIT` |
| `sign_in_user_window` | `1h` | `SIGN_IN_USER_WINDOW` |
| `sign_in_ip_limit` | `20` | `SIGN_IN_IP_LIMIT` |
| `sign_in_ip_window` | `1h` | `SIGN_IN_IP_WINDOW` |
| `access_token_ttl` | `15m` | `ACCESS_TOKEN_TTL` |
| `refresh_token_ttl` | `720h` | `REFRESH_TOKEN_TTL` |

- **Qolganlari statik:** baza, port, JWT kaliti, SMTP, `APP_ENV`, `PUBLIC_URL`, `SIGN_IN_REDIRECT_URL`, Swagger. Ular ulanishlar va server tuzilishiga ta'sir qiladi, ishlab turgan serverda almashtirish xavfli.
- **Dinamik sozlama faqat faylda.** Env'dagi eski nomlar o'qilmaydi. Aks holda env fayldagi qiymatni bosib, fayl o'zgarishi hech narsa qilmay qolardi. Bitta sozlama — bitta manba.
- **Fayllar `config/` papkasida:** `config/.env.local` — statik sozlamalar (lokal dev, `make` yuklaydi), `config/config.yaml` — dinamik sozlamalar (server o'zi o'qiydi).
- **Fayl yo'li:** `CONFIG_FILE` env (standart `config/config.yaml`). Fayl bo'lmasa yoki unda kalit bo'lmasa, standart qiymat olinadi, shuning uchun fayl bo'sh yoki qisman bo'lishi mumkin. Standart qiymatlar `internal/dynamic.go` da (`DefaultDynamic`).
- **Qayta yuklash:** server fayl turgan **papkani** kuzatadi (`fsnotify`), faylning o'zini emas: Kubernetes ConfigMap faylni symlink almashtirish orqali yangilaydi. Ketma-ket hodisalar qisqa kutish (debounce) bilan bitta qayta o'qishga birlashadi.
- **Xato fayl serverni to'xtatmaydi:** yangi fayl o'qiladi va tekshiriladi (musbat son, musbat muddat, noma'lum kalit yo'q — kalit nomidagi xato ko'rinsin). Xato bo'lsa, server eski qiymatlar bilan ishlashda davom etadi va xatoni log'ga yozadi. Ishga tushishda xato fayl esa server ishga tushmaydi.
- **Kod qiymatni har safar o'qiydi:** sozlamalar `atomic.Pointer` ichida, yangilanish butun struct'ni bir zumda almashtiradi. Kod qiymatni ishga tushishda nusxalab olmaydi, kerak bo'lganda oxirgi qiymatni oladi. O'zgarish log'ga eski → yangi qiymat bilan yoziladi.
- **O'zgarish faqat yangi narsalarga ta'sir qiladi:** muddat yaratilishda hisoblanadi (`expires_at`, token `exp`). `access_token_ttl` yoki `refresh_token_ttl` o'zgarsa, mavjud token va sessionlar eski muddat bilan qoladi. Limit va oyna o'zgarishi keyingi tekshiruvdan boshlab kuchga kiradi.
- **Kutubxona:** `fsnotify` va `gopkg.in/yaml.v3`, loader o'zimizniki. `viper`/`koanf` ishlatilmaydi: sozlamalar soni kam va `internal/config.go` allaqachon qo'lda yozilgan.
- **Prod:** fayl Kubernetes ConfigMap'dan papka sifatida ulanadi (`subPath` bilan emas, aks holda fayl yangilanmaydi). ConfigMap o'zgargach, kubelet faylni ~1 daqiqa ichida yangilaydi. Har bir pod faylni o'zi qayta o'qiydi, shuning uchun podlar qisqa vaqt turli qiymatda ishlashi mumkin. Bu qabul qilingan.

## Ochiq savollar

- Dinamik sozlamalar git'da saqlanib, ConfigMap'ga avtomatik yetkaziladimi (ArgoCD, Flux) yoki qo'lda `kubectl apply` qilinadimi?
- Prod Kubernetes'da ishlaydi. Qolgan tafsilotlar (qaysi SMTP server, ingress, secretlar) hali aniqlanmagan.
