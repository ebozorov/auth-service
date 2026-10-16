import { Outlet } from "react-router";

export function AuthLayout() {
  return (
    <main className="flex min-h-svh items-center justify-center bg-muted p-4">
      <div className="w-full max-w-sm">
        <Outlet />
      </div>
    </main>
  );
}
