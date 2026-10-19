import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, type FormEvent } from "react";
import { api, type Schemas } from "@/api/client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { useI18n } from "@/lib/i18n";

type User = Schemas["User"];
type Role = NonNullable<User["role"]>;

// null: panel yopiq, "new": qo'shish, User: tahrirlash.
export type UserFormTarget = User | "new" | null;

export function UserFormSheet({ target, onClose }: { target: UserFormTarget; onClose: () => void }) {
  const { t } = useI18n();
  const user = target === "new" ? null : target;
  return (
    <Sheet open={target !== null} onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="sm:max-w-md" closeLabel={t("common.close")}>
        {target !== null && (
          <>
            <SheetHeader className="border-b">
              <SheetTitle>{user ? t("users.editTitle") : t("users.addTitle")}</SheetTitle>
              <SheetDescription>{user ? `${user.name} · ${user.email}` : t("users.addDescription")}</SheetDescription>
            </SheetHeader>
            {/* key: boshqa user ochilganda forma qaytadan boshlanadi. */}
            <UserForm key={user?.id ?? "new"} user={user} onDone={onClose} />
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}

function UserForm({ user, onDone }: { user: User | null; onDone: () => void }) {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [email, setEmail] = useState(user?.email ?? "");
  const [name, setName] = useState(user?.name ?? "");
  const [role, setRole] = useState<Role>(user?.role ?? "user");
  const [enabled, setEnabled] = useState(user?.enabled ?? true);

  const save = useMutation({
    mutationFn: () =>
      user
        ? api<User>(`/users/${user.id}`, {
            method: "PATCH",
            body: JSON.stringify({ name, role, enabled } satisfies Schemas["UpdateUserRequest"]),
          })
        : api<User>("/users", {
            method: "POST",
            body: JSON.stringify({ email, name, role } satisfies Schemas["CreateUserRequest"]),
          }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "users"] });
      onDone();
    },
  });

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    save.mutate();
  }

  return (
    <form onSubmit={onSubmit} className="flex flex-1 flex-col overflow-y-auto">
      <div className="flex flex-col gap-5 px-4">
        <div className="flex flex-col gap-2">
          <Label htmlFor="user-email">{t("users.email")}</Label>
          <Input
            id="user-email"
            type="email"
            required
            autoFocus={!user}
            disabled={user !== null}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          {user && <p className="text-xs text-muted-foreground">{t("users.emailReadonly")}</p>}
        </div>

        <div className="flex flex-col gap-2">
          <Label htmlFor="user-name">{t("users.name")}</Label>
          <Input
            id="user-name"
            required
            autoFocus={user !== null}
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>

        <fieldset className="flex flex-col gap-2">
          <legend className="mb-2 text-sm font-medium">{t("users.role")}</legend>
          <div className="grid grid-cols-2 gap-2">
            {(["user", "admin"] as const).map((r) => (
              <Button
                key={r}
                type="button"
                variant={role === r ? "default" : "outline"}
                aria-pressed={role === r}
                onClick={() => setRole(r)}
              >
                {r === "admin" ? t("users.roleAdmin") : t("users.roleUser")}
              </Button>
            ))}
          </div>
        </fieldset>

        {user && (
          <label className="flex items-start gap-3 rounded-lg border p-3">
            <input
              type="checkbox"
              className="mt-0.5 size-4 accent-primary"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
            />
            <span className="flex flex-col gap-1">
              <span className="text-sm font-medium">{t("users.active")}</span>
              <span className="text-xs text-muted-foreground">{t("users.activeHint")}</span>
            </span>
          </label>
        )}

        {save.isError && <p className="text-sm text-destructive">{save.error.message}</p>}
      </div>

      <SheetFooter className="flex-row justify-end border-t">
        <Button type="button" variant="ghost" onClick={onDone} disabled={save.isPending}>
          {t("common.cancel")}
        </Button>
        <Button type="submit" disabled={save.isPending}>
          {user ? t("common.save") : t("users.add")}
        </Button>
      </SheetFooter>
    </form>
  );
}
