import { useMutation } from "@tanstack/react-query";
import { MailCheck } from "lucide-react";
import { useState, type FormEvent } from "react";
import { API_PREFIX, ApiError, type Schemas } from "@/api/client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { detectDevice } from "@/lib/device";
import { useI18n } from "@/lib/i18n";

// device_secret HttpOnly cookie bo'lib keladi, magic link faqat shu brauzerda ishlaydi.
// Qurilma ma'lumoti brauzerning o'zidan olinadi: server User-Agent'dan aniqroq bilolmaydi.
async function signIn(email: string) {
  const body: Schemas["SignInRequest"] = { email, device: await detectDevice() };
  const res = await fetch(`${API_PREFIX}/auth/sign-in`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const data: Schemas["Error"] | null = await res.json().catch(() => null);
    throw new ApiError(res.status, data?.error ?? res.statusText);
  }
}

export function SignInPage() {
  const { t } = useI18n();
  const [email, setEmail] = useState("");
  const mutation = useMutation({ mutationFn: signIn });

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    mutation.mutate(email);
  }

  if (mutation.isSuccess) {
    const [before, after] = t("signIn.sent").split("{email}");
    return (
      <Card>
        <CardHeader className="items-center text-center">
          <MailCheck className="mb-2 size-10 text-muted-foreground" />
          <CardTitle>{t("signIn.checkTitle")}</CardTitle>
          <CardDescription>{before}<span className="font-medium text-foreground">{email}</span>{after}</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("signIn.title")}</CardTitle>
        <CardDescription>{t("signIn.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="email">{t("signIn.email")}</Label>
            <Input
              id="email"
              type="email"
              required
              autoFocus
              placeholder="ali@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
          {mutation.isError && <p className="text-sm text-destructive">{mutation.error.message}</p>}
          <Button type="submit" disabled={mutation.isPending}>
            {t("signIn.submit")}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
