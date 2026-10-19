import { useQuery } from "@tanstack/react-query";
import { KeyRound, Settings, Users } from "lucide-react";
import { useState } from "react";
import { Link, Navigate, NavLink, Outlet } from "react-router";
import { api, type Schemas } from "@/api/client";
import { useI18n } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { Navbar } from "./navbar";

const nav = [
  { to: "users", label: "nav.users", icon: Users },
  { to: "clients", label: "nav.clients", icon: KeyRound },
  { to: "settings", label: "nav.settings", icon: Settings },
] as const;

// Yopiq panel faqat ikonkalar qatori bo'lib qoladi; holat qayta ochilganda ham saqlanadi.
const SIDEBAR_KEY = "sidebar-collapsed";

function storedCollapsed() {
  try {
    return localStorage.getItem(SIDEBAR_KEY) === "1";
  } catch {
    return false;
  }
}

// Joriy user. Layout uni yuklaydi, sahifalar keshdan oladi.
export function useMe() {
  return useQuery({
    queryKey: ["me"],
    queryFn: () => api<Schemas["User"]>("/auth/me"),
  });
}

export function AdminLayout() {
  const { t } = useI18n();
  const [collapsed, setCollapsed] = useState(storedCollapsed);
  const me = useMe();

  if (me.isPending) return null;
  if (me.isError) return <Navigate to="/login" replace />;

  function toggleSidebar() {
    const next = !collapsed;
    try {
      localStorage.setItem(SIDEBAR_KEY, next ? "1" : "0");
    } catch {}
    setCollapsed(next);
  }

  return (
    <div className="flex min-h-svh">
      <aside
        className={cn(
          "sticky top-0 flex h-svh shrink-0 flex-col overflow-hidden border-r bg-muted/40 p-2 transition-[width] duration-200",
          collapsed ? "w-14" : "w-60",
        )}
      >
        <Link
          to="/"
          title={collapsed ? t("nav.dashboard") : undefined}
          className="mb-4 flex h-10 items-center rounded-md px-2.5 text-lg font-semibold whitespace-nowrap hover:bg-accent hover:text-accent-foreground"
        >
          {collapsed ? "a" : "auth-service"}
        </Link>
        <nav className="flex flex-1 flex-col gap-1">
          {nav.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              title={collapsed ? t(label) : undefined}
              className={({ isActive }) =>
                cn(
                  "flex h-9 items-center gap-2 rounded-md px-3 text-sm whitespace-nowrap text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                  isActive && "bg-accent font-medium text-accent-foreground",
                )
              }
            >
              <Icon className="size-4 shrink-0" />
              <span className={cn(collapsed && "sr-only")}>{t(label)}</span>
            </NavLink>
          ))}
        </nav>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <Navbar user={me.data} sidebarOpen={!collapsed} onToggleSidebar={toggleSidebar} />
        <main className="flex-1 p-8">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
