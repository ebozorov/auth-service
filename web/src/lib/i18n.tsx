import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";

const uz = {
  "nav.users": "Foydalanuvchilar",
  "nav.clients": "Clientlar",
  "nav.settings": "Sozlamalar",
  "common.comingSoon": "Bu bo'lim keyinroq qo'shiladi.",
  "common.close": "Yopish",
  "common.cancel": "Bekor qilish",

  "navbar.sidebar": "Panelni ochish/yopish",
  "navbar.language": "Til",
  "navbar.theme": "Mavzu",
  "navbar.profile": "Profil",
  "navbar.logout": "Chiqish",
  "theme.light": "Yorug'",
  "theme.dark": "Qorong'i",
  "theme.system": "Tizim",

  "signIn.title": "Kirish",
  "signIn.description": "Emailingizni kiriting, kirish linkini yuboramiz.",
  "signIn.email": "Email",
  "signIn.submit": "Link yuborish",
  "signIn.checkTitle": "Emailingizni tekshiring",
  "signIn.sent": "{email} manziliga kirish linki yuborildi. Linkni shu brauzerda oching.",

  "users.name": "Ism",
  "users.email": "Email",
  "users.role": "Rol",
  "users.status": "Holat",
  "users.createdAt": "Qo'shilgan",
  "users.enabled": "faol",
  "users.disabled": "o'chirilgan",
  "users.search": "Ism yoki email...",
  "users.sessions": "Sessionlar",
  "users.add": "Qo'shish",
  "users.edit": "Tahrirlash",
  "users.addTitle": "Yangi foydalanuvchi",
  "users.addDescription": "Foydalanuvchi email orqali kirish linki bilan tizimga kiradi.",
  "users.editTitle": "Foydalanuvchini tahrirlash",
  "users.roleUser": "Foydalanuvchi",
  "users.roleAdmin": "Admin",
  "users.active": "Faol",
  "users.activeHint": "O'chirilsa, barcha sessionlari tugatiladi.",
  "users.emailReadonly": "Emailni o'zgartirib bo'lmaydi.",
  "common.save": "Saqlash",

  "sessions.title": "Sessionlar",
  "sessions.empty": "Faol session yo'q",
  "sessions.current": "Joriy",
  "sessions.unknownDevice": "Noma'lum qurilma",
  "sessions.startedAt": "Kirgan",
  "sessions.lastActive": "Oxirgi faollik",
  "sessions.expiresAt": "Tugaydi",
  "sessions.terminate": "Tugatish",
  "sessions.confirmTerminate": "Ha, tugatish",

  "table.search": "Qidirish...",
  "table.empty": "Ma'lumot yo'q",
};

export type MessageKey = keyof typeof uz;

const ru: Record<MessageKey, string> = {
  "nav.users": "Пользователи",
  "nav.clients": "Клиенты",
  "nav.settings": "Настройки",
  "common.comingSoon": "Этот раздел появится позже.",
  "common.close": "Закрыть",
  "common.cancel": "Отмена",

  "navbar.sidebar": "Открыть/закрыть панель",
  "navbar.language": "Язык",
  "navbar.theme": "Тема",
  "navbar.profile": "Профиль",
  "navbar.logout": "Выйти",
  "theme.light": "Светлая",
  "theme.dark": "Тёмная",
  "theme.system": "Системная",

  "signIn.title": "Вход",
  "signIn.description": "Введите email, мы отправим ссылку для входа.",
  "signIn.email": "Email",
  "signIn.submit": "Отправить ссылку",
  "signIn.checkTitle": "Проверьте почту",
  "signIn.sent": "Ссылка для входа отправлена на {email}. Откройте её в этом браузере.",

  "users.name": "Имя",
  "users.email": "Email",
  "users.role": "Роль",
  "users.status": "Статус",
  "users.createdAt": "Добавлен",
  "users.enabled": "активен",
  "users.disabled": "отключён",
  "users.search": "Имя или email...",
  "users.sessions": "Сессии",
  "users.add": "Добавить",
  "users.edit": "Изменить",
  "users.addTitle": "Новый пользователь",
  "users.addDescription": "Пользователь входит по ссылке, отправленной на email.",
  "users.editTitle": "Изменить пользователя",
  "users.roleUser": "Пользователь",
  "users.roleAdmin": "Админ",
  "users.active": "Активен",
  "users.activeHint": "При отключении все его сессии будут завершены.",
  "users.emailReadonly": "Email изменить нельзя.",
  "common.save": "Сохранить",

  "sessions.title": "Сессии",
  "sessions.empty": "Нет активных сессий",
  "sessions.current": "Текущая",
  "sessions.unknownDevice": "Неизвестное устройство",
  "sessions.startedAt": "Вход",
  "sessions.lastActive": "Последняя активность",
  "sessions.expiresAt": "Истекает",
  "sessions.terminate": "Завершить",
  "sessions.confirmTerminate": "Да, завершить",

  "table.search": "Поиск...",
  "table.empty": "Нет данных",
};

const en: Record<MessageKey, string> = {
  "nav.users": "Users",
  "nav.clients": "Clients",
  "nav.settings": "Settings",
  "common.comingSoon": "This section is coming later.",
  "common.close": "Close",
  "common.cancel": "Cancel",

  "navbar.sidebar": "Toggle sidebar",
  "navbar.language": "Language",
  "navbar.theme": "Theme",
  "navbar.profile": "Profile",
  "navbar.logout": "Sign out",
  "theme.light": "Light",
  "theme.dark": "Dark",
  "theme.system": "System",

  "signIn.title": "Sign in",
  "signIn.description": "Enter your email and we'll send you a sign-in link.",
  "signIn.email": "Email",
  "signIn.submit": "Send link",
  "signIn.checkTitle": "Check your email",
  "signIn.sent": "A sign-in link was sent to {email}. Open it in this browser.",

  "users.name": "Name",
  "users.email": "Email",
  "users.role": "Role",
  "users.status": "Status",
  "users.createdAt": "Added",
  "users.enabled": "active",
  "users.disabled": "disabled",
  "users.search": "Name or email...",
  "users.sessions": "Sessions",
  "users.add": "Add",
  "users.edit": "Edit",
  "users.addTitle": "New user",
  "users.addDescription": "The user signs in with a link sent to their email.",
  "users.editTitle": "Edit user",
  "users.roleUser": "User",
  "users.roleAdmin": "Admin",
  "users.active": "Active",
  "users.activeHint": "Disabling ends all of their sessions.",
  "users.emailReadonly": "Email cannot be changed.",
  "common.save": "Save",

  "sessions.title": "Sessions",
  "sessions.empty": "No active sessions",
  "sessions.current": "Current",
  "sessions.unknownDevice": "Unknown device",
  "sessions.startedAt": "Signed in",
  "sessions.lastActive": "Last active",
  "sessions.expiresAt": "Expires",
  "sessions.terminate": "Terminate",
  "sessions.confirmTerminate": "Yes, terminate",

  "table.search": "Search...",
  "table.empty": "No data",
};

const messages = { uz, ru, en };

export type Locale = keyof typeof messages;

export const locales: { value: Locale; label: string }[] = [
  { value: "uz", label: "O'zbekcha" },
  { value: "ru", label: "Русский" },
  { value: "en", label: "English" },
];

const STORAGE_KEY = "locale";

function storedLocale(): Locale {
  try {
    const v = localStorage.getItem(STORAGE_KEY);
    if (v && v in messages) return v as Locale;
  } catch {}
  return "uz";
}

type Translate = (key: MessageKey) => string;

const I18nContext = createContext<{ locale: Locale; setLocale: (l: Locale) => void; t: Translate } | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(storedLocale);

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  function setLocale(l: Locale) {
    try {
      localStorage.setItem(STORAGE_KEY, l);
    } catch {}
    setLocaleState(l);
  }

  const t: Translate = useCallback((key) => messages[locale][key], [locale]);

  return <I18nContext value={{ locale, setLocale, t }}>{children}</I18nContext>;
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n: I18nProvider yo'q");
  return ctx;
}
