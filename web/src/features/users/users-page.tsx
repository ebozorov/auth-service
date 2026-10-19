import { useQuery } from "@tanstack/react-query";
import { createColumnHelper } from "@tanstack/react-table";
import { MonitorSmartphone, Pencil, Plus } from "lucide-react";
import { useMemo, useState } from "react";
import { api, type Schemas } from "@/api/client";
import { DataTable } from "@/components/data-table/data-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n";
import { UserFormSheet, type UserFormTarget } from "./user-form-sheet";
import { UserSessionsSheet } from "./user-sessions-sheet";

type User = Schemas["User"];

const col = createColumnHelper<User>();

export function UsersPage() {
  const { locale, t } = useI18n();
  const [sessionsOf, setSessionsOf] = useState<User | null>(null);
  const [formTarget, setFormTarget] = useState<UserFormTarget>(null);
  const users = useQuery({
    queryKey: ["admin", "users"],
    queryFn: () => api<User[]>("/users"),
  });

  const columns = useMemo(
    () => [
      col.accessor("name", { header: t("users.name") }),
      col.accessor("email", { header: t("users.email") }),
      col.accessor("role", {
        header: t("users.role"),
        cell: (c) => <Badge variant={c.getValue() === "admin" ? "default" : "secondary"}>{c.getValue()}</Badge>,
      }),
      col.accessor("enabled", {
        header: t("users.status"),
        cell: (c) =>
          c.getValue() ? (
            <Badge variant="outline">{t("users.enabled")}</Badge>
          ) : (
            <Badge variant="destructive">{t("users.disabled")}</Badge>
          ),
      }),
      col.accessor("created_at", {
        header: t("users.createdAt"),
        cell: (c) => (c.getValue() ? new Date(c.getValue()!).toLocaleDateString(locale) : ""),
      }),
      col.display({
        id: "actions",
        cell: (c) => (
          <div className="flex justify-end gap-1">
            <Button variant="ghost" size="sm" onClick={() => setFormTarget(c.row.original)}>
              <Pencil />
              {t("users.edit")}
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setSessionsOf(c.row.original)}>
              <MonitorSmartphone />
              {t("users.sessions")}
            </Button>
          </div>
        ),
      }),
    ],
    [locale, t],
  );

  return (
    <div className="flex flex-col gap-6">
      {users.isError ? (
        <p className="text-sm text-destructive">{users.error.message}</p>
      ) : (
        <DataTable
          columns={columns}
          data={users.data ?? []}
          searchPlaceholder={t("users.search")}
          toolbar={
            <Button onClick={() => setFormTarget("new")}>
              <Plus />
              {t("users.add")}
            </Button>
          }
        />
      )}
      <UserFormSheet target={formTarget} onClose={() => setFormTarget(null)} />
      <UserSessionsSheet user={sessionsOf} onClose={() => setSessionsOf(null)} />
    </div>
  );
}
