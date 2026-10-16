import { Outlet } from "react-router";
import { LanguageMenu, ThemeMenu } from "./navbar";

// Kirishdan oldin ham til va mavzuni tanlash mumkin.
export function AuthLayout() {
  return (
    <main className="relative flex min-h-svh items-center justify-center bg-muted p-4">
      <div className="absolute top-4 right-4 flex items-center gap-1">
        <LanguageMenu />
        <ThemeMenu />
      </div>
      <div className="w-full max-w-sm">
        <Outlet />
      </div>
    </main>
  );
}
