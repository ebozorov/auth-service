# API bo'yicha qarorlar

Endpointlar, yo'llar va JSON formati. API'ning to'liq tavsifi: [openapi.json](../openapi.json). Barcha qarorlar ro'yxati: [DECISION.md](../DECISION.md).

## 19. API `/api/v1` ostida

- Barcha API endpointlari `/api/v1` ostida. Cookie path'lari ham shu prefiks bilan: `device_secret` uchun `/api/v1/auth/sign-in`, refresh token uchun `/api/v1/auth`.
- `/api/v1` ostidagi noma'lum yo'l JSON 404 qaytaradi. Qolgan GET yo'llar web UI'ga tegishli (20-qaror, [web](web.md)).
- `/.well-known/*` prefikssiz: discovery issuer manzili yonida bo'lishi kerak (OpenID standarti). `/swagger-ui/` ham prefikssiz, u API emas.

## 21. ID'lar JSON'da string

- API'da (JSON javoblar va JWT `sid`) ID **string** bo'lib yuboriladi: `"id": "41784020480034629"`. Qiymat 2^53 dan katta (14-qaror, [system](system.md)), JavaScript JSON sonni `double` ga o'girib yaxlitlaydi (`…629` → `…630`).
- So'rovlarda server `"123"` ni ham, `123` ni ham qabul qiladi (masalan, `sid` son bo'lgan eski tokenlar). Go kodida qanday qilingani: 22-qaror ([models](models.md)).
