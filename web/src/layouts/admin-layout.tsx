import { useQuery } from "@tanstack/react-query";
import { KeyRound, LogOut, Settings, Users } from "lucide-react";
import { Navigate, NavLink, Outlet, useNavigate } from "react-router";
import { api, logout, type Schemas } from "@/api/client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const nav = [
  { to: "users", label: "Foydalanuvchilar", icon: Users },
  { to: "clients", label: "Clientlar", icon: KeyRound },
  { to: "settings", label: "Sozlamalar", icon: Settings },
];

export function AdminLayout() {
  const navigate = useNavigate();
  const me = useQuery({
    queryKey: ["me"],
    queryFn: () => api<Schemas["User"]>("/auth/me"),
  });

  if (me.isPending) return null;
  if (me.isError) return <Navigate to="/login" replace />;

  async function onLogout() {
    await logout();
    navigate("/login", { replace: true });
  }

  return (
    <div className="flex min-h-svh">
      <aside className="flex w-60 flex-col border-r bg-muted/40 p-4">
        <div className="mb-6 px-2 text-lg font-semibold">auth-api</div>
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
              {label}
            </NavLink>
          ))}
        </nav>
        <div className="border-t pt-4">
          <div className="truncate px-2 text-sm font-medium">{me.data.name}</div>
          <div className="mb-2 truncate px-2 text-xs text-muted-foreground">{me.data.email}</div>
          <Button variant="ghost" size="sm" className="w-full justify-start" onClick={onLogout}>
            <LogOut />
            Chiqish
          </Button>
        </div>
      </aside>
      <main className="flex-1 p-8">
        <Outlet />
      </main>
    </div>
  );
}
