import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Copy, Monitor } from "lucide-react";
import { useState } from "react";
import { api, type Schemas } from "@/api/client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { useI18n } from "@/lib/i18n";

type User = Schemas["User"];
type Session = Schemas["Session"];

// user null bo'lsa panel yopiq.
export function UserSessionsSheet({ user, onClose }: { user: User | null; onClose: () => void }) {
  const { t } = useI18n();
  return (
    <Sheet open={user !== null} onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="sm:max-w-md" closeLabel={t("common.close")}>
        {user && (
          <>
            <SheetHeader className="border-b">
              <SheetTitle>{t("sessions.title")}</SheetTitle>
              <SheetDescription>
                {user.name} · {user.email}
              </SheetDescription>
            </SheetHeader>
            <SessionList user={user} />
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}

function SessionList({ user }: { user: User }) {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const queryKey = ["admin", "users", user.id, "sessions"];
  const sessions = useQuery({
    queryKey,
    queryFn: () => api<Session[]>(`/users/${user.id}/sessions`),
  });
  const terminate = useMutation({
    mutationFn: (sid: string) => api<void>(`/users/${user.id}/sessions/${sid}`, { method: "DELETE" }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey }),
  });

  if (sessions.isPending) return null;
  if (sessions.isError) return <p className="px-4 text-sm text-destructive">{sessions.error.message}</p>;

  return (
    <div className="flex flex-col gap-3 overflow-y-auto px-4 pb-4">
      {terminate.isError && <p className="text-sm text-destructive">{terminate.error.message}</p>}
      {sessions.data.length === 0 ? (
        <p className="py-8 text-center text-sm text-muted-foreground">{t("sessions.empty")}</p>
      ) : (
        sessions.data.map((s) => (
          <SessionCard
            key={s.id}
            session={s}
            pending={terminate.isPending && terminate.variables === s.id}
            onTerminate={() => terminate.mutate(s.id!)}
          />
        ))
      )}
    </div>
  );
}

function SessionCard({
  session: s,
  pending,
  onTerminate,
}: {
  session: Session;
  pending: boolean;
  onTerminate: () => void;
}) {
  const { locale, t } = useI18n();
  const [confirming, setConfirming] = useState(false);
  const date = (v?: string) => (v ? new Date(v).toLocaleString(locale) : "—");
  const device = [s.device?.os, s.device?.client].filter(Boolean).join(" · ");

  return (
    <div className="flex flex-col gap-3 rounded-lg border p-3">
      <div className="flex items-start gap-3">
        <Monitor className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="truncate text-sm font-medium">{s.device?.name || t("sessions.unknownDevice")}</span>
            {s.current && <Badge variant="secondary">{t("sessions.current")}</Badge>}
          </div>
          {device && <div className="truncate text-xs text-muted-foreground">{device}</div>}
        </div>
      </div>

      <dl className="grid grid-cols-[auto_1fr] items-center gap-x-4 gap-y-1 text-xs">
        <dt className="text-muted-foreground">{t("sessions.id")}</dt>
        <dd className="flex min-w-0 items-center gap-1">
          <span className="truncate font-mono select-all">{s.id}</span>
          <CopyButton value={s.id ?? ""} />
        </dd>
        <dt className="text-muted-foreground">{t("sessions.startedAt")}</dt>
        <dd>
          {date(s.started_at)} · {s.started_ip}
        </dd>
        <dt className="text-muted-foreground">{t("sessions.lastActive")}</dt>
        <dd>
          {date(s.current_at)} · {s.current_ip}
        </dd>
        <dt className="text-muted-foreground">{t("sessions.expiresAt")}</dt>
        <dd>{date(s.expires_at)}</dd>
      </dl>

      {/* Joriy session'ni bu yerdan tugatib bo'lmaydi: buning uchun "Chiqish" bor. */}
      {!s.current && (
        <div className="flex justify-end gap-2">
          {confirming ? (
            <>
              <Button variant="ghost" size="sm" onClick={() => setConfirming(false)} disabled={pending}>
                {t("common.cancel")}
              </Button>
              <Button variant="destructive" size="sm" onClick={onTerminate} disabled={pending}>
                {t("sessions.confirmTerminate")}
              </Button>
            </>
          ) : (
            <Button variant="outline" size="sm" onClick={() => setConfirming(true)}>
              {t("sessions.terminate")}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

// Session ID audit uchun: loglarda shu ID bo'yicha qidiriladi.
function CopyButton({ value }: { value: string }) {
  const { t } = useI18n();
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {}
  }

  const label = copied ? t("common.copied") : t("common.copy");
  return (
    <Button variant="ghost" size="icon-xs" aria-label={label} title={label} onClick={copy}>
      {copied ? <Check /> : <Copy />}
    </Button>
  );
}
