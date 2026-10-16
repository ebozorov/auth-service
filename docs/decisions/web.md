# Web (UI) bo'yicha qarorlar

Web UI: sahifalar, routing va stek. Barcha qarorlar ro'yxati: [DECISION.md](../DECISION.md).

## 18. Dashboard: hamma uchun

- Tizimga kirgandan keyin birinchi sahifa dashboard (`/`). Chap paneldagi loyiha nomi ham shu yerga qaytaradi.
- Dashboard **hamma uchun**, faqat admin uchun emas: tizimdan ishxona xodimlari foydalanadi va tizimdagi umumiy kartinani ko'rish hammaga teng ochiq. Shuning uchun dashboard'ga faqat har bir xodim ko'rishi mumkin bo'lgan ma'lumot chiqadi. Boshqa userlarning shaxsiy ma'lumotlari (email, IP, qurilmalar) admin bo'limlarida qoladi.
- Ma'lumotlar materialized view'dan olinadi (23-qaror, [database](database.md)).

## 20. Web UI routing va stek

- `/api/v1` dan tashqari GET yo'llarga web UI javob beradi (19-qaror, [api](api.md)). Fayl bo'lmagan yo'lga `index.html` qaytadi, routing'ni React bajaradi.
- Web: React, Vite, Tailwind va shadcn/ui. Build binary ichiga embed qilinadi (17-qaror, [system](system.md)).
