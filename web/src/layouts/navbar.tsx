import { Languages, LogOut, Monitor, Moon, PanelLeft, Sun } from "lucide-react";
import { useNavigate } from "react-router";
import { logout, type Schemas } from "@/api/client";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { locales, useI18n, type Locale } from "@/lib/i18n";
import { useTheme, type Theme } from "@/lib/theme";

export function LanguageMenu() {
  const { locale, setLocale, t } = useI18n();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" aria-label={t("navbar.language")}>
          <Languages />
          <span className="uppercase">{locale}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuRadioGroup value={locale} onValueChange={(v) => setLocale(v as Locale)}>
          {locales.map((l) => (
            <DropdownMenuRadioItem key={l.value} value={l.value}>
              {l.label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

const themes = [
  { value: "light", label: "theme.light", icon: Sun },
  { value: "dark", label: "theme.dark", icon: Moon },
  { value: "system", label: "theme.system", icon: Monitor },
] as const;

export function ThemeMenu() {
  const { theme, setTheme } = useTheme();
  const { t } = useI18n();
  const Icon = themes.find((x) => x.value === theme)!.icon;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" aria-label={t("navbar.theme")}>
          <Icon />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuRadioGroup value={theme} onValueChange={(v) => setTheme(v as Theme)}>
          {themes.map(({ value, label, icon: ItemIcon }) => (
            <DropdownMenuRadioItem key={value} value={value}>
              <ItemIcon />
              {t(label)}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

// "Ali Valiyev" -> "AV"; ism bo'lmasa emailning birinchi harfi.
function initials(user: Schemas["User"]) {
  const parts = (user.name ?? "").trim().split(/\s+/).filter(Boolean);
  const letters = parts.length ? parts.slice(0, 2).map((p) => p[0]) : [user.email?.[0] ?? "?"];
  return letters.join("").toUpperCase();
}

function ProfileMenu({ user }: { user: Schemas["User"] }) {
  const navigate = useNavigate();
  const { t } = useI18n();

  async function onLogout() {
    await logout();
    navigate("/login", { replace: true });
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={t("navbar.profile")}
          className="rounded-full outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50"
        >
          <Avatar>
            <AvatarFallback className="text-xs font-medium">{initials(user)}</AvatarFallback>
          </Avatar>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel className="font-normal">
          <div className="truncate text-sm font-medium">{user.name}</div>
          <div className="truncate text-xs text-muted-foreground">{user.email}</div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem variant="destructive" onSelect={onLogout}>
          <LogOut />
          {t("navbar.logout")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function Navbar({ user, onToggleSidebar }: { user: Schemas["User"]; onToggleSidebar: () => void }) {
  const { t } = useI18n();
  return (
    <header className="sticky top-0 z-10 flex h-14 items-center gap-1 border-b bg-background/80 px-4 backdrop-blur">
      <Button variant="ghost" size="icon" className="mr-auto" aria-label={t("navbar.sidebar")} onClick={onToggleSidebar}>
        <PanelLeft />
      </Button>
      <LanguageMenu />
      <ThemeMenu />
      <div className="ml-2">
        <ProfileMenu user={user} />
      </div>
    </header>
  );
}
