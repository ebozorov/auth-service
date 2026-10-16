import { createBrowserRouter } from "react-router";
import { SignInPage } from "@/features/auth/sign-in-page";
import { ClientsPage } from "@/features/clients/clients-page";
import { DashboardPage } from "@/features/dashboard/dashboard-page";
import { SettingsPage } from "@/features/settings/settings-page";
import { UsersPage } from "@/features/users/users-page";
import { AdminLayout } from "@/layouts/admin-layout";
import { AuthLayout } from "@/layouts/auth-layout";
import type { MessageKey } from "@/lib/i18n";

// Navbar sahifa nomini shu yerdan oladi (useMatches).
export type RouteHandle = { title?: MessageKey };

export const router = createBrowserRouter(
  [
    {
      element: <AuthLayout />,
      children: [{ path: "login", element: <SignInPage /> }],
    },
    {
      element: <AdminLayout />,
      children: [
        { index: true, element: <DashboardPage />, handle: { title: "nav.dashboard" } satisfies RouteHandle },
        { path: "users", element: <UsersPage />, handle: { title: "nav.users" } satisfies RouteHandle },
        { path: "clients", element: <ClientsPage />, handle: { title: "nav.clients" } satisfies RouteHandle },
        { path: "settings", element: <SettingsPage />, handle: { title: "nav.settings" } satisfies RouteHandle },
      ],
    },
  ],
);
