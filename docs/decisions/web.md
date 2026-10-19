# Web (UI) bo'yicha qarorlar

Web UI: sahifalar, routing va stek. Barcha qarorlar ro'yxati: [DECISION.md](../DECISION.md).

## 18. Dashboard va userlar bo'limi: hamma uchun

- Tizimga kirgandan keyin birinchi sahifa dashboard (`/`). Chap paneldagi loyiha nomi ham shu yerga qaytaradi.
- Dashboard **hamma uchun**, faqat admin uchun emas: tizimdan ishxona xodimlari foydalanadi va tizimdagi umumiy kartinani ko'rish hammaga teng ochiq.
- Userlar bo'limi ham hamma uchun ochiq: auth-service ishxona xodimlari uchun, shuning uchun har bir user barcha userlarning ma'lumotlarini (email, rol, holat, qurilmalar va IP) ko'radi. `user` bilan `admin` farqi faqat o'zgartirishda (24-qaror, [database](database.md)): `user` uchun qo'shish, tahrirlash va qurilmani tugatish tugmalari ko'rsatilmaydi, server ham bu amallarni 403 bilan rad etadi.
- Kim qaysi userlarni ko'rishi (masalan, faqat o'z bo'limini) keyingi bosqich masalasi: ochiq savollarda ([database](database.md)).
- Ma'lumotlar materialized view'dan olinadi (23-qaror, [database](database.md)).

## 20. Web UI routing va stek

- `/api/v1` dan tashqari GET yo'llarga web UI javob beradi (19-qaror, [api](api.md)). Fayl bo'lmagan yo'lga `index.html` qaytadi, routing'ni React bajaradi.
- Web: React, Vite, Tailwind va shadcn/ui. Build binary ichiga embed qilinadi (17-qaror, [system](system.md)).
