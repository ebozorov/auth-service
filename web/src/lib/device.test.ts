import { afterEach, describe, expect, test, vi } from "vitest";
import { detectDevice } from "./device";

type Brand = { brand: string; version: string };

function stubNavigator(nav: {
  userAgent?: string;
  maxTouchPoints?: number;
  userAgentData?: {
    mobile: boolean;
    platform: string;
    hints?: { platformVersion?: string; model?: string; fullVersionList?: Brand[] };
    fail?: boolean;
  };
}) {
  const uaData = nav.userAgentData && {
    mobile: nav.userAgentData.mobile,
    platform: nav.userAgentData.platform,
    getHighEntropyValues: () =>
      nav.userAgentData!.fail ? Promise.reject(new Error("denied")) : Promise.resolve(nav.userAgentData!.hints ?? {}),
  };
  vi.stubGlobal("navigator", { userAgent: nav.userAgent ?? "", maxTouchPoints: nav.maxTouchPoints ?? 0, userAgentData: uaData });
}

afterEach(() => {
  vi.unstubAllGlobals();
});

const chromeBrands = (version: string): Brand[] => [
  { brand: "Not)A;Brand", version: "8.0.0.0" },
  { brand: "Chromium", version },
  { brand: "Google Chrome", version },
];

describe("Client Hints", () => {
  test.each([
    {
      name: "Windows 11, Chrome",
      platform: "Windows",
      mobile: false,
      hints: { platformVersion: "15.0.0", fullVersionList: chromeBrands("130.0.6723.92") },
      want: { name: "Windows PC", os: "Windows 11", client: "Chrome 130" },
    },
    {
      name: "Windows 10, Edge",
      platform: "Windows",
      mobile: false,
      hints: {
        platformVersion: "10.0.0",
        fullVersionList: [
          { brand: "Microsoft Edge", version: "129.0.2792.79" },
          { brand: "Not=A?Brand", version: "8.0.0.0" },
          { brand: "Chromium", version: "129.0.6668.90" },
        ],
      },
      want: { name: "Windows PC", os: "Windows 10", client: "Edge 129" },
    },
    {
      name: "Windows, versiya yo'q",
      platform: "Windows",
      mobile: false,
      hints: { platformVersion: "0.0.0" },
      want: { name: "Windows PC", os: "Windows", client: "" },
    },
    {
      name: "macOS 14.5",
      platform: "macOS",
      mobile: false,
      hints: { platformVersion: "14.5.0", fullVersionList: chromeBrands("130.0.0.0") },
      want: { name: "Mac", os: "macOS 14.5", client: "Chrome 130" },
    },
    {
      name: "macOS 15.0: minor 0 ko'rsatilmaydi",
      platform: "macOS",
      mobile: false,
      hints: { platformVersion: "15.0.0" },
      want: { name: "Mac", os: "macOS 15", client: "" },
    },
    {
      name: "Android telefon, model bor",
      platform: "Android",
      mobile: true,
      hints: { platformVersion: "14.0.0", model: "Pixel 8", fullVersionList: chromeBrands("130.0.0.0") },
      want: { name: "Pixel 8", os: "Android 14", client: "Chrome 130" },
    },
    {
      name: "Android planshet, model yo'q",
      platform: "Android",
      mobile: false,
      hints: { platformVersion: "13.0.0", model: "" },
      want: { name: "Android tablet", os: "Android 13", client: "" },
    },
    {
      name: "faqat Chromium brendi",
      platform: "Linux",
      mobile: false,
      hints: { fullVersionList: [{ brand: "Chromium", version: "128.0.0.0" }] },
      want: { name: "Linux PC", os: "Linux", client: "Chromium 128" },
    },
  ])("$name", async ({ platform, mobile, hints, want }) => {
    stubNavigator({ userAgentData: { platform, mobile, hints } });
    await expect(detectDevice()).resolves.toEqual(want);
  });

  test("Linux distributivi UA'dan olinadi", async () => {
    stubNavigator({
      userAgent: "Mozilla/5.0 (X11; Ubuntu; Linux x86_64) AppleWebKit/537.36 Chrome/130.0.0.0 Safari/537.36",
      userAgentData: { platform: "Linux", mobile: false, hints: { fullVersionList: chromeBrands("130.0.0.0") } },
    });
    await expect(detectDevice()).resolves.toEqual({ name: "Linux PC", os: "Ubuntu", client: "Chrome 130" });
  });

  test("Client Hints xato bersa UA ishlatiladi", async () => {
    stubNavigator({
      userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
      userAgentData: { platform: "Windows", mobile: false, fail: true },
    });
    await expect(detectDevice()).resolves.toEqual({ name: "Windows PC", os: "Windows", client: "Chrome 130" });
  });
});

describe("User-Agent", () => {
  test.each([
    {
      name: "Windows, Firefox",
      ua: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:131.0) Gecko/20100101 Firefox/131.0",
      want: { name: "Windows PC", os: "Windows", client: "Firefox 131" },
    },
    {
      name: "macOS, Safari",
      ua: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15",
      want: { name: "Mac", os: "macOS", client: "Safari 18.0" },
    },
    {
      name: "iPadOS: Mac deb ko'rinadi, sensorli ekrandan ajratiladi",
      ua: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",
      touch: 5,
      want: { name: "iPad", os: "iPadOS", client: "Safari 17.5" },
    },
    {
      name: "iPhone, Safari",
      ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Mobile/15E148 Safari/604.1",
      want: { name: "iPhone", os: "iOS 17.6", client: "Safari 17.6" },
    },
    {
      name: "iPhone, Chrome (CriOS)",
      ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/130.0.6723.90 Mobile/15E148 Safari/604.1",
      want: { name: "iPhone", os: "iOS 18.0", client: "Chrome 130" },
    },
    {
      name: "Android telefon, Samsung Internet emas Chrome",
      ua: "Mozilla/5.0 (Linux; Android 14; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Mobile Safari/537.36",
      want: { name: "Android phone", os: "Android 14", client: "Chrome 130" },
    },
    {
      name: "Android planshet",
      ua: "Mozilla/5.0 (Linux; Android 12.1; SM-X200) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
      want: { name: "Android tablet", os: "Android 12.1", client: "Chrome 129" },
    },
    {
      name: "Edge Chrome'dan oldin tekshiriladi",
      ua: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.2849.56",
      want: { name: "Windows PC", os: "Windows", client: "Edge 130" },
    },
    {
      name: "Opera",
      ua: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36 OPR/114.0.0.0",
      want: { name: "Windows PC", os: "Windows", client: "Opera 114" },
    },
    {
      name: "Yandex",
      ua: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 YaBrowser/24.10.0.0 Safari/537.36",
      want: { name: "Windows PC", os: "Windows", client: "Yandex 24" },
    },
    {
      name: "Fedora, Firefox",
      ua: "Mozilla/5.0 (X11; Fedora; Linux x86_64; rv:131.0) Gecko/20100101 Firefox/131.0",
      want: { name: "Linux PC", os: "Fedora", client: "Firefox 131" },
    },
    {
      name: "Linux, distributiv noma'lum",
      ua: "Mozilla/5.0 (X11; Linux x86_64; rv:131.0) Gecko/20100101 Firefox/131.0",
      want: { name: "Linux PC", os: "Linux", client: "Firefox 131" },
    },
    {
      // UA'da "Linux" yo'q, qurilma nomi aniqlanmaydi (UI "Noma'lum qurilma" ko'rsatadi).
      name: "ChromeOS",
      ua: "Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
      want: { name: "", os: "ChromeOS", client: "Chrome 130" },
    },
    {
      name: "noma'lum",
      ua: "curl/8.5.0",
      want: { name: "", os: "", client: "" },
    },
  ])("$name", async ({ ua, touch, want }) => {
    stubNavigator({ userAgent: ua, maxTouchPoints: touch });
    await expect(detectDevice()).resolves.toEqual(want);
  });
});
