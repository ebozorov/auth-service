# Flow bo'yicha qarorlar

Kirish (sign-in) oqimi, qurilmaga bog'lash va himoya. Barcha qarorlar ro'yxati: [DECISION.md](../DECISION.md).

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
- **Ko'p so'rov:** bitta userga vaqt birligida cheklangan son so'rov (`user_id, created_at` indeksi orqali), IP bo'yicha ham limit (11-qaror, [system](system.md)). Limit qiymatlari: 10-qaror ([system](system.md)).
- **Login qilgan qurilma:** sessionni tugatish orqali.

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
- Cookie path'lari: 19-qaror ([api](api.md)).
