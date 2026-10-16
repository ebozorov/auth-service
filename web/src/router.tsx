import { createBrowserRouter, Navigate } from "react-router";
import { SignInPage } from "@/features/auth/sign-in-page";
import { ClientsPage } from "@/features/clients/clients-page";
import { SettingsPage } from "@/features/settings/settings-page";
import { UsersPage } from "@/features/users/users-page";
import { AdminLayout } from "@/layouts/admin-layout";
import { AuthLayout } from "@/layouts/auth-layout";

export const router = createBrowserRouter(
  [
    {
      element: <AuthLayout />,
      children: [{ path: "login", element: <SignInPage /> }],
    },
    {
      element: <AdminLayout />,
      children: [
        { index: true, element: <Navigate to="users" replace /> },
        { path: "users", element: <UsersPage /> },
        { path: "clients", element: <ClientsPage /> },
        { path: "settings", element: <SettingsPage /> },
      ],
    },
  ],
);
