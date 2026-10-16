import type { components } from "./schema";

export type Schemas = components["schemas"];

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

// Access token faqat xotirada turadi. Refresh token HttpOnly cookie'da (path /api/v1/auth),
// JS uni ko'rmaydi: brauzer /auth/refresh ga o'zi qo'shib yuboradi.
export const API_PREFIX = "/api/v1";

let accessToken: string | null = null;
let refreshing: Promise<boolean> | null = null;

// Bir vaqtda bir nechta 401 kelsa ham refresh bitta bo'ladi.
function refresh(): Promise<boolean> {
  refreshing ??= fetch(`${API_PREFIX}/auth/refresh`, { method: "POST", credentials: "include" })
    .then(async (res) => {
      if (!res.ok) {
        accessToken = null;
        return false;
      }
      const data: Schemas["TokenResponse"] = await res.json();
      accessToken = data.access_token ?? null;
      return accessToken !== null;
    })
    .finally(() => {
      refreshing = null;
    });
  return refreshing;
}

async function send(path: string, init: RequestInit): Promise<Response> {
  const headers = new Headers(init.headers);
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  if (init.body) headers.set("Content-Type", "application/json");
  return fetch(API_PREFIX + path, { ...init, headers, credentials: "include" });
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  if (!accessToken) await refresh();

  let res = await send(path, init);
  if (res.status === 401 && (await refresh())) {
    res = await send(path, init);
  }

  if (!res.ok) {
    const body: Schemas["Error"] | null = await res.json().catch(() => null);
    throw new ApiError(res.status, body?.error ?? res.statusText);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export async function logout() {
  await api("/auth/logout", { method: "POST" }).catch(() => {});
  accessToken = null;
}
