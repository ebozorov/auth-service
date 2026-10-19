# Database bo'yicha qarorlar

Jadvallar, ustunlar va view'lar. Sxemaning manbasi: `migrations/`. Barcha qarorlar ro'yxati: [DECISION.md](../DECISION.md).

## Jadvallar

| Jadval | Vazifasi |
|---|---|
| `actions` | amallar, bir belgili kalit: `c`, `r`, `u`, `d` (24-qaror) |
| `resources` | permission himoya qiladigan narsalar: `user`, `role`, ... (24-qaror) |
| `permissions` | bitta resursdagi bitta amal: `user:c` (24-qaror) |
| `roles` | rollar: permissionlar to'plami va token uchun yig'ilgan `permissions` (24-qaror) |
| `role_permissions` | rol ↔ permission (24-qaror) |
| `users` | xodimlar (email, ism, `role`, `enabled`); `role` — `roles` ga FK (24-qaror) |
| `signin_requests` | kutilayotgan magic link so'rovlari va qurilma ma'lumoti |
| `sessions` | faol sessionlar = qurilmalar |
| `ended_sessions` | tugagan sessionlar arxivi: session ma'lumoti va qachon, nima uchun, kim tomonidan tugatilgani |

Migratsiyalar har bir jadval (yoki bir-biriga bog'liq jadvallar guruhi) uchun alohida fayl, FK tartibida, `000000` dan boshlab:

| Fayl | Jadvallar |
|---|---|
| `000000_actions` | `actions` |
| `000001_resources` | `resources` |
| `000002_permissions` | `permissions` |
| `000003_roles` | `roles`, `role_permissions` |
| `000004_users` | `users` |
| `000005_sessions` | `signin_requests`, `sessions`, `ended_sessions` (16-qaror) |

**Prod holati:** loyiha hali prod'ga chiqmagan. Shuning uchun sxema o'zgarganda mavjud migratsiya fayllari to'g'ridan-to'g'ri tahrirlanadi, yangi raqamli fayl qo'shilmaydi. Prod'ga chiqqach, bu yerga sana yoziladi. Shundan keyin chiqqan migratsiyalar o'zgartirilmaydi, har bir o'zgarish yangi raqamli faylda bo'ladi.

## 5. Session = device (Telegram kabi)

- Alohida `devices` jadvali yo'q. Har bir session bitta qurilma hisoblanadi.
- Qurilma ma'lumoti `sessions.device` (`JSONB`) ustunida saqlanadi. U session ochilganda signin so'rovidan ko'chadi (16-qaror).
- Qurilmalar ro'yxati — userning faol sessionlari. Qurilmani tugatish — sessionni tugatish.
- Tugatilgan session `ended_sessions` jadvaliga ko'chiriladi va `sessions` dan o'chiriladi. `sessions` da faqat faol sessionlar qoladi (16-qaror).

## 16. Jadvallar uchligi: `signin_requests` → `sessions` → `ended_sessions`

Ma'lumot jadvaldan jadvalga ko'chib o'tadi. Har bir bosqichda kerakli ma'lumot keyingi jadvalga nusxalanadi va oldingi jadvaldan o'chiriladi. Shuning uchun bitta qator bir vaqtda faqat bitta jadvalda bo'ladi.

| Jadval | Nima saqlanadi | Qachon o'chadi |
|---|---|---|
| `signin_requests` | kutilayotgan so'rov: link token hash, `device_secret` hash, qurilma, `requested_ip`, `requested_at` | link ishlatilib session ochilganda; ishlatilmasa, tozalashda |
| `sessions` | faol session: refresh token hash, qurilma, `started_ip` / `started_at`, `current_ip` / `current_at`, muddat | session tugaganda |
| `ended_sessions` | arxiv: `sessions` ustunlari (`token_hash` dan tashqari) va bitta `ended` JSON ustuni (qachon, nima sababdan, kim va qaysi session tugatgani) | o'chmaydi |

- **Bitta ID:** `signin_requests.id = sessions.id = ended_sessions.id`. ID ilovada yaratiladi (14-qaror, [system](system.md)) va jadvaldan jadvalga o'zgarmasdan ko'chadi. JWT'dagi `sid` ham shu ID. Jadvallar o'rtasida FK yo'q, chunki oldingi qator har doim o'chirilgan bo'ladi.
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
- **Tozalash:** server har soatda muddati o'tgan sessionlarni `ended_sessions` ga ko'chiradi (`ended.reason = 'expired'`) va ishlatilmagan eski signin so'rovlarini o'chiradi.
- **User limiti** (8-qaror, [flows](flows.md)) `signin_requests` dagi qatorlarni sanaydi. Ishlatilgan so'rovlar darhol o'chgani uchun limit faqat kutilayotgan va ishlatilmagan so'rovlarni hisoblaydi. Bu yetarli: limit email spam'iga qarshi, muvaffaqiyatli kirish esa spam emas.
- **Ustun nomlari:** so'rov vaqti `requested_*`, session boshlanishi `started_*`, sessionning hozirgi holati `current_*` (IP va vaqt birga yangilanadi). Arxivda ham shu nomlar qoladi.
- **ID ichidagi vaqt** email kiritilgan paytni bildiradi. Session boshlangan aniq vaqt `started_at` da.

## 23. Dashboard ma'lumotlari materialized view'da

- Dashboard'ga (18-qaror, [web](web.md)) chiqadigan ma'lumotlar **materialized view** qilinadi. Maqsad: dashboard tez ochilishi va kutib qolmasligi. Hisob-kitob (sanash, guruhlash) so'rov paytida emas, oldindan bajariladi. Dashboard so'rovi tayyor qatorlarni o'qiydi, shuning uchun uning tezligi jadvallar o'sishiga bog'liq emas.
- View'lar fonda yangilanadi: `REFRESH MATERIALIZED VIEW CONCURRENTLY`. `CONCURRENTLY` yangilanish paytida o'qishni to'xtatmaydi, buning uchun har bir view'da `UNIQUE` indeks bo'lishi kerak.
- Ma'lumot yangilanishlar orasida biroz eskirgan bo'ladi. Dashboard uchun bu maqbul, va dashboard ma'lumot qachon yangilanganini ko'rsatadi.

## 24. Rollar va permissionlar

Qaysi rolga qanday permission berilganini belgilaydi. Userning permissionlari access token'da `permissions` claim'i bo'lib boradi. Migratsiyalar: `000000_actions` … `000004_users`.

- **Amal** (`actions`) — resurs bilan nima qilish mumkinligi. Kaliti bitta belgi (`a-z`, `0-9`), migratsiyada beriladi: `c` create, `r` read, `u` update, `d` delete. **Resurs** (`resources`) — permission himoya qiladigan narsa: `user`, `role`, keyinchalik `order`, `payment`.
- **Permission** — bitta resursdagi **bitta amal**, bitta qator: `resource` va `action` ustunlari `resources` va `actions` ga FK. Kaliti `resurs:amal`, masalan `user:c` ("Create users"); `key = resource || ':' || action` ni `CHECK` tekshiradi. Har bir permissionning `name` va `description` i bor. Ishlatilayotgan resurs yoki amalni FK o'chirishga qo'ymaydi.
- `actions`, `resources` va `permissions` ni **faqat migratsiya** to'ldiradi: permissionni servis kodi tekshiradi, shuning uchun uni dasturchi qo'shadi. Admin **rollarni** yaratadi va ularga permission biriktiradi (`role_permissions`).
- **Userda bitta rol**: `users.role` — `roles (key)` ga FK, standart qiymati `'user'`. Rol ma'lumoti faqat shu ustunda, alohida `user_roles` jadvali yo'q.
- **Rolni o'chirish:** unga biriktirilgan userlar bo'lsa o'chirib bo'lmaydi (`users.role` → `RESTRICT`, `users_role_idx`), aks holda istalgan rol o'chiriladi; permission bog'lanishlari rol bilan birga o'chadi (`role_permissions` → `CASCADE`). Maxsus (system) rollar yo'q. `admin` va `user` — oddiy rollar: `admin` ga barcha permissionlar seed qilingan, `user` bo'sh. `user` roli o'chirilsa, rolsiz qo'shilgan user FK xatosi bilan rad etiladi.
- **Kalitlar `TEXT`** (`actions.key` — `CHAR(1)`), Sonyflake emas (14-qaror, [system](system.md)): bu kichik lug'at jadvallar, qatorlari migratsiyada seed qilinadi va kalit tokenga to'g'ridan-to'g'ri yoziladi. Kalit o'zgarmaydi, ko'rinadigan nom `name` da.
- **`roles.permissions` oldindan yig'iladi** — token'ga yoziladigan tayyor qiymat: rolning permissionlari resurs bo'yicha birlashtirilgan, harflar `actions.sort_order` tartibida, resurslar alifbo tartibida, masalan `{role:r,user:cru}`.
  - Ustunni faqat `role_permissions` dagi `AFTER INSERT OR DELETE` trigger yozadi: rolning `permissions` ini `role_permission_keys` bilan qayta hisoblaydi va `updated_at` ni yangilaydi. Permission o'chib CASCADE bo'lganda ham shu trigger ishlaydi.
  - `roles` da trigger yo'q: `name`, `description` o'zgarishi permissionlarga ta'sir qilmaydi.
  - `role_permissions` qatorlari faqat qo'shiladi va o'chiriladi, `UPDATE` qilinmaydi — aks holda ustun eskicha qoladi. `roles.permissions` ga qo'lda yozilmaydi.
  - `actions.sort_order` ni o'zgartiradigan migratsiya oxirida `UPDATE roles SET permissions = role_permission_keys(key);` bilan barcha rollar qayta hisoblanadi.
- **Token** uchun bitta o'qish yetadi: `"permissions": ["role:crud", "user:crud"]`. Permission o'zgarishi keyingi refresh'da, ko'pi bilan `ACCESS_TOKEN_TTL` ichida kuchga kiradi.

  ```sql
  SELECT r.permissions FROM users u JOIN roles r ON r.key = u.role WHERE u.id = $1;
  ```

## Ochiq savollar

- Dashboard view'lari qancha vaqtda bir yangilanadi va buni nima bajaradi (server ichidagi ticker yoki `pg_cron`)? Prod'da bir nechta pod bo'lsa, yangilashni faqat bittasi bajarishi kerak.
- `user` roliga boshlang'ich permission kerakmi? Hozir u bo'sh.
- Access token'ga `permissions` dan tashqari `roles` claim'i ham kerakmi?
