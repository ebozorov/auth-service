# Database bo'yicha qarorlar

Jadvallar, ustunlar va view'lar. Sxemaning manbasi: `migrations/`. Barcha qarorlar ro'yxati: [DECISION.md](../DECISION.md).

## Jadvallar

| Jadval | Vazifasi |
|---|---|
| `users` | xodimlar (email, ism, `role`, `enabled`); `role` — `admin` yoki `user` (24-qaror) |
| `signin_requests` | kutilayotgan magic link so'rovlari va qurilma ma'lumoti |
| `sessions` | faol sessionlar = qurilmalar |
| `ended_sessions` | tugagan sessionlar arxivi: session ma'lumoti va qachon, nima uchun, kim tomonidan tugatilgani |

Migratsiyalar har bir jadval (yoki bir-biriga bog'liq jadvallar guruhi) uchun alohida fayl, FK tartibida, `000000` dan boshlab:

| Fayl | Jadvallar |
|---|---|
| `000000_users` | `users` |
| `000001_sessions` | `signin_requests`, `sessions`, `ended_sessions` (16-qaror) |

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

auth-service faqat **authentication** bilan shug'ullanadi: user kimligini aniqlaydi. Permissionlarni boshqarish bu servisning vazifasi emas.

- **Rol hardcoded:** `users.role` — `admin` yoki `user`, `CHECK (role IN ('admin', 'user'))`, standart qiymati `'user'`. Alohida `roles` jadvali yo'q.
- **`admin` — to'liq ruxsat:** admin amallari kodda `role == admin` tekshiruvi bilan ochiladi. `user` faqat tizimga kiradi.
- **Access token'da permissionlar yo'q:** permissionlar ro'yxati cheksiz o'sishi mumkin, token payload'i esa kichik bo'lishi kerak. Go kodida permissionlar qismi yo'q.
- **`actions`, `resources`, `permissions`, `roles`, `role_permissions`** jadvallari va ularning seed ma'lumotlari `../permissions-service/migrations/` ga ko'chirildi. Rollar va permissionlar o'sha servisda hal qilinadi.

## Ochiq savollar

- Dashboard view'lari qancha vaqtda bir yangilanadi va buni nima bajaradi (server ichidagi ticker yoki `pg_cron`)? Prod'da bir nechta pod bo'lsa, yangilashni faqat bittasi bajarishi kerak.
