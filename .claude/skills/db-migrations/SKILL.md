---
name: db-migrations
description: auth-service'ning migrations/ papkasidagi SQL fayllar bilan database darajasida ishlash — sxemani tahlil qilish, jadval/ustun/indeks/constraint/view bo'yicha qaror qabul qilish, yangi migratsiya yozish va uni tekshirish. Go kodiga qaramasdan, faqat SQL va docs/decisions/database.md asosida fikrlanadi. "migration", "migratsiya", "sxema", "jadval", "ustun", "indeks", "constraint", "materialized view", "database qarori" so'zlari kelganda ishlatiladi.
---

# Migratsiyalar bilan ishlash (database darajasida)

Bu skill sxema haqida **database nuqtai nazaridan** fikrlash uchun. Ma'lumotning to'g'riligi ilova kodiga emas, DB'ning o'ziga (tur, `NOT NULL`, `CHECK`, `UNIQUE`, FK, indeks) tayanishi kerak: ilova o'zgaradi, bir nechta pod bo'ladi, kimdir `psql` dan qo'lda yozadi — DB esa har doim bitta.

## Asosiy qoida: Go kodini o'qima

- Manbalar faqat:
  - `migrations/*.sql` — sxemaning yagona manbasi;
  - `docs/decisions/database.md` — database qarorlari va ochiq savollar;
  - `docs/DECISION.md` — barcha qarorlar ro'yxati va raqamlash qoidasi;
  - kerak bo'lsa boshqa `docs/decisions/*.md` (masalan, limitlar — `flows.md`, dashboard — `web.md`), lekin faqat DB'ga tegishli qismi uchun.
- `internal/`, `cmd/` va boshqa Go fayllarini **o'qima va grep qilma**. Savol "kod buni qanday ishlatadi" degan joyga borsa, kodni ochma: farazni aniq yoz ("ilova X ni Y tarzida so'raydi deb hisoblaymiz") va qarorni shu farazga bog'la, yoki userdan so'ra.
- Maqsad: sxema kodsiz ham o'zini tushuntirsin va himoya qilsin. Agar biror invariantni faqat kod ta'minlayotgan bo'lsa — bu DB darajasidagi qaror uchun nomzod.

## Ish tartibi: reja → savol-javob → tasdiq → o'zgartirish

1. **Reja.** Tahlildan keyin qilinadigan ishlar ro'yxati ko'rsatiladi (quyidagi "Output formati"). Oxirida tasdiq so'raladi.
2. **Savol-javob.** User tasdiqlamaguncha suhbat savol-javob tarzida davom etadi: user savol beradi, variant tanlaydi, bandni o'zgartiradi yoki olib tashlaydi. Har o'zgarishdan keyin yangilangan reja qayta ko'rsatiladi va yana tasdiq so'raladi.
3. **Tasdiq.** Faqat userning aniq tasdig'i ("ha", "tasdiqlayman", "qil", "bajar" va h.k.) hisoblanadi. Jim turish, "yaxshi fikr", savol yoki bitta bandga rozilik — butun reja uchun tasdiq emas. User faqat ayrim bandlarni tasdiqlasa ("1 va 3 ni qil"), faqat o'shalar bajariladi.
4. **O'zgartirish.** Tasdiqdan keyingina `migrations/` (va user rozi bo'lsa `docs/`) fayllari o'zgartiriladi — aynan tasdiqlangan reja bo'yicha, ortiqcha narsa qo'shilmaydi. Bajarish paytida rejadan chetga chiqish kerak bo'lib qolsa (masalan, sinovda xato chiqsa), to'xtab, yangi rejani ko'rsatib, qayta tasdiq so'raladi.
5. **Hisobot.** O'zgartirishdan keyin "Bajarildi" formati bilan natija beriladi.

Tasdiqgacha hech qanday fayl yozilmaydi, o'chirilmaydi va vaqtinchalik bazada ham hech narsa ishga tushirilmaydi.

## Output formati

### Reja (tasdiqgacha)

```
## Hozirgi holat
<sxemaning qisqa xulosasi: jadvallar, asosiy bog'lanishlar — 2-4 qator>

## Topilganlar
### Xato / nomuvofiqlik
1. <qisqa sarlavha> — <jadval.ustun>
   Muammo: <nima noto'g'ri va qanday holatda buziladi>
   Taklif: <SQL parchasi>

### Qaror kerak
2. <savol>
   a) <variant> — <afzallik / kamchilik>
   b) <variant> — <afzallik / kamchilik>
   Tavsiya: <qaysi va nega>

### Kichik yaxshilashlar
3. <bir qatorda> — <SQL parchasi>

## Qilinadigan ishlar
| # | Fayl | O'zgarish | Band |
|---|---|---|---|
| 1 | migrations/000002_xxx.up.sql | yangi fayl: ... | 1, 3 |
| 2 | migrations/000002_xxx.down.sql | yangi fayl: ... | 1, 3 |
| 3 | docs/decisions/database.md | N-qaror qo'shiladi | 2 |

Mavjud ma'lumotga ta'siri: <eski qatorlar yangi constraint'ga mosmi, nima tekshiriladi>
Ochiq qolgan savollar: <javob kerak bo'lganlar, bo'lmasa "yo'q">

**Tasdiqlaysizmi?** Hammasini, ayrim bandlarni (raqam bilan) yoki o'zgartirish kiritib.
```

- "Qaror kerak" bandlarida variant tanlanmaguncha u band "Qilinadigan ishlar" ga tavsiya etilgan variant bilan kiradi va bu aniq yoziladi.
- Har bir SQL parchasi to'liq va ishlaydigan bo'lsin (`ALTER TABLE ...` butunligicha), "va hokazo" yo'q.
- Savol-javob paytida qisqa javob beriladi; reja faqat o'zgarganda qayta to'liq ko'rsatiladi.

### Bajarildi (tasdiqdan keyin)

```
## Bajarildi
| Fayl | Nima qilindi |
|---|---|
| ... | ... |

## Tekshiruv
- up → down → up: <o'tdi / xato + xabar matni / o'tkazib yuborildi va nega>
- Constraint sinovlari: <qaysi noto'g'ri INSERT rad etildi>

## Qolgan ishlar
<tasdiqlanmagan bandlar va ochiq savollar, bo'lmasa "yo'q">
```

Commit qilinmaydi — user alohida so'rasa qilinadi.

## Tahlil tartibi

1. `migrations/` dagi barcha fayllarni raqam tartibida o'qi va hozirgi sxemani boshida yig'.
2. `docs/decisions/database.md` ni o'qi; sxema bilan hujjat orasidagi farqni birinchi bo'lib ayt.
3. Har bir jadval uchun quyidagi savollar ro'yxatidan o't.
4. Natijani "Output formati" dagi **Reja** shaklida ber va tasdiq so'ra.

### Har bir jadval uchun savollar

- **Ustun turlari:** `TEXT` vs `BYTEA` (hashlar), `INET`, `TIMESTAMPTZ` (`TIMESTAMP` emas), `JSONB` ichidagi narsa alohida ustun bo'lishi kerakmasmi (filtr, guruhlash, indeks kerak bo'lsa — ustun).
- **NULL:** har bir nullable ustun uchun "qachon NULL bo'ladi?" — javob bo'lmasa, `NOT NULL`.
- **CHECK:** faqat kritik invariantlar uchun taklif qilinadi (buzilsa xavfsizlik yoki ma'lumot to'g'riligi buziladi). Oddiy input validation (bo'sh satr, format, vaqt tartibi, JSONB shakli, hash uzunligi) ilovada qoladi va taklif qilinmaydi.
- **UNIQUE va kalitlar:** biznes-kalit (email) unique'mi; normalizatsiya (`lower`) DB'da tekshiriladimi.
- **FK va `ON DELETE`:** `CASCADE` nimani o'chirib yuboradi? Arxiv/audit jadvallari uchun odatda `RESTRICT`. JSONB ichidagi ID'lar FK'siz qolishini hisobga ol.
- **Indekslar:** har bir kutilayotgan so'rov (`WHERE`, `ORDER BY`, tozalash, dashboard) indeks bilan qoplanganmi; ortiqcha indeks yo'qmi. Kichik jadvalda seq scan normal — buni ham ayt.
- **Vaqt:** vaqt DB'ning `now()` dan olinishi ma'qul (bir nechta pod soati farq qiladi). Muddatlar qat'iymi yoki sirpanuvchimi — sxema izohi bilan mosmi.
- **O'sish:** jadval cheksiz o'sadimi; saqlash muddati, tozalash, partitioning.
- **Bir vaqtdagi so'rovlar:** invariant bir nechta qatorga bog'liq bo'lsa (masalan, "kamida bitta enabled admin"), uni faqat ilova tekshirsa race bo'ladi — trigger, constraint yoki lock kerakmi.

## Loyiha konventsiyalari (migratsiyalarda saqlanadi)

- **Vosita:** `golang-migrate`, PostgreSQL 17. Migratsiyalar server ishga tushganda avtomatik qo'llanadi (`m.Up()`).
- **Fayl nomi:** `NNNNNN_nom.up.sql` va juft `NNNNNN_nom.down.sql` (6 xonali raqam, `snake_case` nom). Har bir `up` ning `down` i bo'ladi va teskari tartibda bekor qiladi (`IF EXISTS` bilan).
- **Fayllar tartibi:** har bir jadval (yoki bir-biriga bog'liq jadvallar guruhi) uchun alohida fayl, FK tartibida, `000000` dan boshlab. Ro'yxat: `docs/decisions/database.md`.
- **Prod holati:** `docs/decisions/database.md` dagi "Prod holati" bo'limiga qara. Loyiha prod'ga chiqmagan bo'lsa, mavjud migratsiya fayllari to'g'ridan-to'g'ri tahrirlanadi. Chiqqan bo'lsa, chiqqan migratsiya o'zgartirilmaydi, yangi raqamli fayl qo'shiladi.
- **Bir faylda bitta mantiqiy o'zgarish.** Fayl bitta multi-statement so'rov sifatida bajariladi, shuning uchun `CREATE INDEX CONCURRENTLY` va `REFRESH ... CONCURRENTLY` boshqa buyruqlar bilan bir faylda ishlamaydi — ular alohida faylga.
- **ID:** `BIGINT PRIMARY KEY`, qiymatni ilova beradi (Sonyflake, 14-qaror), `DEFAULT` yo'q.
- **Vaqt:** `TIMESTAMPTZ`, yaratilish vaqti `DEFAULT now()`.
- **Ustun nomlari:** `requested_*` (so'rov), `started_*` (session boshi), `current_*` (oxirgi holat; IP va vaqt birga). `ended_sessions` ustunlari `sessions` bilan 1:1 bir xil nomda (`token_hash` dan tashqari) — `sessions` ga ustun qo'shilsa, `ended_sessions` ga ham qo'shiladi.
- **Jadvallar uchligi** (16-qaror): `signin_requests` → `sessions` → `ended_sessions`, bitta ID, jadvallar orasida FK yo'q.
- **Indeks nomi:** `<jadval>_<ustunlar>_idx`, masalan `sessions_user_id_idx`.
- **Izohlar:** har bir `CREATE TABLE` ustida inglizcha izoh: jadval vazifasi, keyin har bir ustun `ustun: ma'no, e.g. <misol qiymat>`. Yangi ustun/jadval ham shu uslubda izohlanadi.
- **Materialized view'lar** (23-qaror): har birida `UNIQUE` indeks (`CONCURRENTLY` uchun). View ustunlari jadvalga bog'liq, shuning uchun jadval ustuni o'zgarsa, migratsiya view'ni `DROP` qilib qayta yaratadi.

## Migratsiyani tekshirish

Tasdiqdan keyin migratsiya yozilgach, asosiy DB'ga tegmasdan vaqtinchalik bazada sina (lokal konteyner: `auth-service-postgres`, user `auth`):

```bash
DB=scratch_$(date +%s)
docker exec auth-service-postgres psql -U auth -d auth -c "CREATE DATABASE $DB"
for f in migrations/*.up.sql;   do docker exec -i auth-service-postgres psql -U auth -d $DB -v ON_ERROR_STOP=1 < "$f"; done
for f in $(ls -r migrations/*.down.sql); do docker exec -i auth-service-postgres psql -U auth -d $DB -v ON_ERROR_STOP=1 < "$f"; done
for f in migrations/*.up.sql;   do docker exec -i auth-service-postgres psql -U auth -d $DB -v ON_ERROR_STOP=1 < "$f"; done
docker exec auth-service-postgres psql -U auth -d $DB -c '\d+'
docker exec auth-service-postgres psql -U auth -d auth -c "DROP DATABASE $DB"
```

- up → down → up xatosiz o'tishi kerak.
- Yangi `CHECK`/`NOT NULL` uchun noto'g'ri qatorni `INSERT` qilib, rad etilishini ko'rsat.
- Mavjud ma'lumotli jadvalga constraint qo'shilsa: avval eski qatorlar bunga mosmi — tekshiruv `SELECT` ini ham yoz.
- Konteyner ishlamasa (`make services-up`), tekshiruvni o'tkazib yuborganingni aniq ayt.

## Qarorni hujjatlash

Qaror tasdiqlangach (hujjat o'zgarishi ham "Qilinadigan ishlar" ro'yxatida bo'lsa):

- `docs/decisions/database.md` ga yangi bo'lim: `## N. Sarlavha` — N `docs/DECISION.md` dagi keyingi bo'sh global raqam. Hujjat o'zbek tilida, qisqa gaplar, "nima" va "nega".
- `docs/DECISION.md` jadvaliga qator qo'shiladi.
- Hal bo'lgan ochiq savol `## Ochiq savollar` dan olib tashlanadi, yangisi qo'shiladi.
- SQL izohlari (inglizcha) va `database.md` bir-biriga zid bo'lmasin.
