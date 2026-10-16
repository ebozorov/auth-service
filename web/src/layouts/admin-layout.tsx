import { useQuery } from "@tanstack/react-query";
import { KeyRound, Settings, Users } from "lucide-react";
import { Navigate, NavLink, Outlet } from "react-router";
import { api, type Schemas } from "@/api/client";
import { useI18n } from "@/lib/i18n";
import { cn } from "@/lib/utils";
import { Navbar } from "./navbar";

const nav = [
  { to: "users", label: "nav.users", icon: Users },
  { to: "clients", label: "nav.clients", icon: KeyRound },
  { to: "settings", label: "nav.settings", icon: Settings },
] as const;

export function AdminLayout() {
  const { t } = useI18n();
  const me = useQuery({
    queryKey: ["me"],
    queryFn: () => api<Schemas["User"]>("/auth/me"),
  });

  if (me.isPending) return null;
  if (me.isError) return <Navigate to="/login" replace />;

  return (
    <div className="flex min-h-svh">
      <aside className="sticky top-0 flex h-svh w-60 flex-col border-r bg-muted/40 p-4">
        <div className="mb-6 px-2 text-lg font-semibold">auth-service</div>
        <nav className="flex flex-1 flex-col gap-1">
          {nav.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                  isActive && "bg-accent font-medium text-accent-foreground",
                )
              }
            >
              <Icon className="size-4" />
              {t(label)}
            </NavLink>
          ))}
        </nav>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <Navbar user={me.data} />
        <main className="flex-1 p-8">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
