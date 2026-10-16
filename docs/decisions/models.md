# Model bo'yicha qarorlar

Go kodidagi model turlari (`type Model struct {}`) va ularning JSON ko'rinishi. Barcha qarorlar ro'yxati: [DECISION.md](../DECISION.md).

## 22. ID turlari: `int64` va `internal.ID`

API'da ID string bo'lib yuboriladi (21-qaror, [api](api.md)). Serverda ID `int64` bo'lib qoladi.

- Faqat yoziladigan javob modellarida `int64` + `json:",string"` tegi (ortiqcha allocation yo'q).
- JSON'dan o'qiladigan joylarda `internal.ID`: `"123"` ni ham, `123` ni ham qabul qiladi (masalan, `sid` son bo'lgan eski tokenlar).
- Bazadagi JSON (`ended_sessions.ended`) o'zgarmaydi: u API'ga chiqmaydi, Postgres sonni aniq saqlaydi.
