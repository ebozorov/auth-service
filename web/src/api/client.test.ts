import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

// client.ts access tokenni modul ichida saqlaydi, shuning uchun har bir test modulni yangidan yuklaydi.
let client: typeof import("./client");
let fetchMock: ReturnType<typeof vi.fn>;

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

// Har bir chaqiruvga yangi Response qaytaradi (Response body bir marta o'qiladi).
function route(handler: (url: string, init: RequestInit) => Response | Promise<Response>) {
  fetchMock.mockImplementation((url: string, init: RequestInit = {}) => Promise.resolve(handler(url, init)));
}

const calls = (suffix: string) => fetchMock.mock.calls.filter(([url]) => url.endsWith(suffix));
const authHeader = (call: unknown[]) => new Headers((call[1] as RequestInit).headers).get("Authorization");

beforeEach(async () => {
  vi.resetModules();
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
  client = await import("./client");
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("api", () => {
  test("token bo'lmasa avval refresh qiladi va tokenni yuboradi", async () => {
    route((url) => (url.endsWith("/auth/refresh") ? json({ access_token: "t1" }) : json([{ id: 1 }])));

    await expect(client.api("/users")).resolves.toEqual([{ id: 1 }]);

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual(["/api/v1/auth/refresh", "/api/v1/users"]);
    expect(authHeader(calls("/users")[0])).toBe("Bearer t1");
    expect(calls("/auth/refresh")[0][1]).toMatchObject({ method: "POST", credentials: "include" });
  });

  test("parallel so'rovlarda refresh bitta bo'ladi", async () => {
    route((url) => (url.endsWith("/auth/refresh") ? json({ access_token: "t1" }) : json([])));

    await Promise.all([client.api("/users"), client.api("/users"), client.api("/users/me")]);

    expect(calls("/auth/refresh")).toHaveLength(1);
    expect(calls("/users")).toHaveLength(2);
  });

  test("401 kelsa refresh qilib so'rovni qayta yuboradi", async () => {
    let token = "old";
    route((url, init) => {
      if (url.endsWith("/auth/refresh")) {
        const res = json({ access_token: token });
        token = "new";
        return res;
      }
      const auth = new Headers(init.headers).get("Authorization");
      return auth === "Bearer new" ? json({ ok: true }) : json({ error: "expired" }, 401);
    });

    await expect(client.api("/users/me")).resolves.toEqual({ ok: true });
    expect(calls("/auth/refresh")).toHaveLength(2);
    expect(calls("/users/me").map(authHeader)).toEqual(["Bearer old", "Bearer new"]);
  });

  test("refresh ishlamasa 401 ApiError bo'lib qaytadi", async () => {
    route((url) => (url.endsWith("/auth/refresh") ? json({ error: "no session" }, 401) : json({ error: "unauthorized" }, 401)));

    const err = await client.api("/users").catch((e) => e);
    expect(err).toBeInstanceOf(client.ApiError);
    expect(err).toMatchObject({ status: 401, message: "unauthorized" });
    expect(authHeader(calls("/users")[0])).toBeNull();
  });

  test("xato javobdagi error matni ApiError ga o'tadi", async () => {
    route((url) => (url.endsWith("/auth/refresh") ? json({ access_token: "t1" }) : json({ error: "forbidden" }, 403)));

    await expect(client.api("/users", { method: "POST", body: "{}" })).rejects.toMatchObject({
      status: 403,
      message: "forbidden",
    });
  });

  test("javob JSON bo'lmasa statusText ishlatiladi", async () => {
    route((url) =>
      url.endsWith("/auth/refresh")
        ? json({ access_token: "t1" })
        : new Response("<html>", { status: 502, statusText: "Bad Gateway" }),
    );

    await expect(client.api("/users")).rejects.toMatchObject({ status: 502, message: "Bad Gateway" });
  });

  test("204 da undefined qaytaradi", async () => {
    route((url) => (url.endsWith("/auth/refresh") ? json({ access_token: "t1" }) : new Response(null, { status: 204 })));

    await expect(client.api("/sessions/1", { method: "DELETE" })).resolves.toBeUndefined();
  });

  test("body bo'lsa Content-Type qo'yiladi, bo'lmasa yo'q", async () => {
    route((url) => (url.endsWith("/auth/refresh") ? json({ access_token: "t1" }) : json({})));

    await client.api("/users", { method: "POST", body: "{}" });
    await client.api("/users");

    const [post, get] = calls("/users").map((c) => new Headers((c[1] as RequestInit).headers));
    expect(post.get("Content-Type")).toBe("application/json");
    expect(get.get("Content-Type")).toBeNull();
  });
});

describe("logout", () => {
  test("tokenni o'chiradi: keyingi so'rov yana refresh qiladi", async () => {
    route((url) => (url.endsWith("/auth/refresh") ? json({ access_token: "t1" }) : new Response(null, { status: 204 })));

    await client.api("/users/me");
    await client.logout();
    await client.api("/users/me");

    expect(calls("/auth/logout")).toHaveLength(1);
    expect(calls("/auth/refresh")).toHaveLength(2);
  });

  test("server xato qaytarsa ham tashlamaydi", async () => {
    route((url) => (url.endsWith("/auth/refresh") ? json({ access_token: "t1" }) : json({ error: "boom" }, 500)));

    await expect(client.logout()).resolves.toBeUndefined();
  });
});
