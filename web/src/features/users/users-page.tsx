import { useQuery } from "@tanstack/react-query";
import { createColumnHelper } from "@tanstack/react-table";
import { api, type Schemas } from "@/api/client";
import { DataTable } from "@/components/data-table/data-table";
import { Badge } from "@/components/ui/badge";

type User = Schemas["User"];

const col = createColumnHelper<User>();

const columns = [
  col.accessor("name", { header: "Ism" }),
  col.accessor("email", { header: "Email" }),
  col.accessor("role", {
    header: "Rol",
    cell: (c) => <Badge variant={c.getValue() === "admin" ? "default" : "secondary"}>{c.getValue()}</Badge>,
  }),
  col.accessor("enabled", {
    header: "Holat",
    cell: (c) =>
      c.getValue() ? <Badge variant="outline">faol</Badge> : <Badge variant="destructive">o'chirilgan</Badge>,
  }),
  col.accessor("created_at", {
    header: "Qo'shilgan",
    cell: (c) => (c.getValue() ? new Date(c.getValue()!).toLocaleDateString() : ""),
  }),
];

export function UsersPage() {
  const users = useQuery({
    queryKey: ["admin", "users"],
    queryFn: () => api<User[]>("/admin/users"),
  });

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold">Foydalanuvchilar</h1>
      {users.isError ? (
        <p className="text-sm text-destructive">{users.error.message}</p>
      ) : (
        <DataTable columns={columns} data={users.data ?? []} searchPlaceholder="Ism yoki email..." />
      )}
    </div>
  );
}
