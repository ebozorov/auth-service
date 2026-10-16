import type { Schemas } from "@/api/client";

type Device = Pick<Schemas["Device"], "name" | "os" | "client">;

// User-Agent Client Hints: hozircha faqat Chromium brauzerlarida bor, TS lib.dom'da tipi yo'q.
type Brand = { brand: string; version: string };
type UAData = {
  mobile: boolean;
  platform: string;
  getHighEntropyValues(hints: string[]): Promise<{
    platformVersion?: string;
    model?: string;
    fullVersionList?: Brand[];
  }>;
};

// Qurilma ma'lumoti sessionlar ro'yxatida ko'rinadi (docs/DECISION.md #6).
// User-Agent "muzlatilgan" (Windows 11 ham "Windows NT 10.0", macOS doim 10_15_7),
// shuning uchun imkon bo'lsa Client Hints'dan olinadi, bo'lmasa UA'dan.
export async function detectDevice(): Promise<Device> {
  const uaData = (navigator as Navigator & { userAgentData?: UAData }).userAgentData;
  if (uaData) {
    try {
      return await fromClientHints(uaData);
    } catch {}
  }
  return fromUserAgent(navigator.userAgent);
}

async function fromClientHints(uaData: UAData): Promise<Device> {
  const hints = await uaData.getHighEntropyValues(["platformVersion", "model", "fullVersionList"]);
  const os = osName(uaData.platform, hints.platformVersion ?? "");
  return {
    name: hints.model || deviceName(uaData.platform, uaData.mobile),
    os,
    client: browserFromBrands(hints.fullVersionList ?? []),
  };
}

function osName(platform: string, version: string) {
  const [major = "", minor = ""] = version.split(".");
  switch (platform) {
    case "Windows":
      // platformVersion 13+ Windows 11, 1..10 esa Windows 10.
      return Number(major) >= 13 ? "Windows 11" : Number(major) > 0 ? "Windows 10" : "Windows";
    case "macOS":
    case "Android":
      return major ? `${platform} ${minor !== "0" && minor ? `${major}.${minor}` : major}` : platform;
    case "Linux":
      return linuxDistro(navigator.userAgent) ?? platform;
    default:
      return platform;
  }
}

function deviceName(platform: string, mobile: boolean) {
  if (platform === "macOS") return "Mac";
  if (platform === "Android") return mobile ? "Android phone" : "Android tablet";
  return `${platform} PC`;
}

const brandNames: Record<string, string> = {
  "Google Chrome": "Chrome",
  "Microsoft Edge": "Edge",
};

// GREASE brendlari ("Not.A/Brand") va umumiy "Chromium" tashlanadi, aniq brauzer qoladi.
function browserFromBrands(brands: Brand[]) {
  const real = brands.filter((b) => !/not.a.brand/i.test(b.brand));
  const b = real.find((b) => b.brand !== "Chromium") ?? real[0];
  if (!b) return "";
  return `${brandNames[b.brand] ?? b.brand} ${b.version.split(".")[0]}`;
}

function fromUserAgent(ua: string): Device {
  return { name: uaDeviceName(ua), os: uaOS(ua), client: uaBrowser(ua) };
}

function uaDeviceName(ua: string) {
  if (/iPhone/.test(ua)) return "iPhone";
  // iPadOS Safari o'zini Mac deb ko'rsatadi, sensorli ekrandan ajratiladi.
  if (/iPad/.test(ua) || (/Macintosh/.test(ua) && navigator.maxTouchPoints > 1)) return "iPad";
  if (/Macintosh/.test(ua)) return "Mac";
  if (/Android/.test(ua)) return /Mobile/.test(ua) ? "Android phone" : "Android tablet";
  if (/Windows/.test(ua)) return "Windows PC";
  if (/Linux/.test(ua)) return "Linux PC";
  return "";
}

function uaOS(ua: string) {
  let m: RegExpMatchArray | null;
  if ((m = ua.match(/(?:iPhone|CPU) OS (\d+)_(\d+)/))) return `iOS ${m[1]}.${m[2]}`;
  if ((m = ua.match(/Android (\d+(?:\.\d+)?)/))) return `Android ${m[1]}`;
  // Windows va macOS versiyasi UA'da muzlatilgan, shuning uchun faqat nomi.
  if (/Windows/.test(ua)) return "Windows";
  if (/Macintosh/.test(ua)) return navigator.maxTouchPoints > 1 ? "iPadOS" : "macOS";
  if (/CrOS/.test(ua)) return "ChromeOS";
  if (/Linux/.test(ua)) return linuxDistro(ua) ?? "Linux";
  return "";
}

// Brauzer distributivni bermaydi; faqat ba'zi distributivlarning o'z Firefox/Chromium
// build'lari nomini UA'ga qo'shadi, masalan "X11; Ubuntu; Linux x86_64".
const distros: [RegExp, string][] = [
  [/Ubuntu/, "Ubuntu"],
  [/Fedora/, "Fedora"],
  [/Debian/, "Debian"],
  [/Linux Mint|\bMint\b/, "Linux Mint"],
  [/openSUSE/i, "openSUSE"],
  [/Manjaro/, "Manjaro"],
  [/Arch Linux/, "Arch Linux"],
  [/CentOS/, "CentOS"],
  [/Red Hat/, "Red Hat"],
];

function linuxDistro(ua: string) {
  return distros.find(([re]) => re.test(ua))?.[1];
}

function uaBrowser(ua: string) {
  const rules: [RegExp, string][] = [
    [/Edg(?:A|iOS)?\/(\d+)/, "Edge"],
    [/OPR\/(\d+)/, "Opera"],
    [/YaBrowser\/(\d+)/, "Yandex"],
    [/(?:Firefox|FxiOS)\/(\d+)/, "Firefox"],
    [/(?:Chrome|CriOS)\/(\d+)/, "Chrome"],
    [/Version\/(\d+(?:\.\d+)?).*Safari/, "Safari"],
  ];
  for (const [re, name] of rules) {
    const m = ua.match(re);
    if (m) return `${name} ${m[1]}`;
  }
  return "";
}
