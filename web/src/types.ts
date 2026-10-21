export type WidgetType =
  | "clock"
  | "calendar"
  | "rss"
  | "weather"
  | "bookmarks"
  | "note";
export interface Link {
  title: string;
  url: string;
}
export interface WidgetConfig {
  timezone?: string;
  body?: string;
  links?: Link[];
  limit?: number;
}
export interface SourceConfig {
  url?: string;
  latitude?: number;
  longitude?: number;
  location?: string;
}
export interface Source {
  id: string;
  type: "rss" | "weather";
  config: SourceConfig;
}
export interface Widget {
  id: string;
  type: WidgetType;
  title: string;
  size: "small" | "wide";
  sourceId?: string;
  config: WidgetConfig;
}
export interface Board {
  id: string;
  title: string;
  revision: number;
  theme: "light" | "dark";
  widgets: Widget[];
  sources: Source[];
}
export interface FeedData {
  title: string;
  items: { title: string; url: string; publishedAt?: string }[];
}
export interface WeatherData {
  location: string;
  temperature: number;
  feelsLike: number;
  humidity: number;
  code: number;
}
export interface Snapshot {
  state: "fresh" | "loading" | "stale" | "error";
  data?: FeedData | WeatherData;
  updatedAt?: string;
  expiresAt?: string;
  retryAt?: string;
  error?: string;
}

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export async function request<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const response = await fetch(path, options);
  if (
    response.status === 401 &&
    path !== "/api/v1/auth/login" &&
    typeof window !== "undefined"
  )
    window.dispatchEvent(new Event("glance:unauthorized"));
  let body: any;
  try {
    body = await response.json();
  } catch {
    throw new ApiError(
      `서버 응답을 읽을 수 없습니다 (${response.status}).`,
      response.status,
    );
  }
  if (!response.ok)
    throw new ApiError(
      typeof body?.error === "string"
        ? body.error
        : `요청에 실패했습니다 (${response.status})`,
      response.status,
    );
  return body as T;
}
