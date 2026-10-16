import { useI18n } from "@/lib/i18n";

export function SettingsPage() {
  const { t } = useI18n();
  return (
    <div className="flex flex-col gap-6">
      <p className="text-sm text-muted-foreground">{t("common.comingSoon")}</p>
    </div>
  );
}
