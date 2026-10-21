import type { components } from "./generated/api";
export type Board = components["schemas"]["Board"];
export type Widget = components["schemas"]["Widget"];
export type WidgetType = Widget["type"];
export type Source = components["schemas"]["Source"];
export type WidgetConfig = components["schemas"]["WidgetConfig"];
export type SourceConfig = components["schemas"]["SourceConfig"];
export type Link = components["schemas"]["Link"];
export type FeedData = components["schemas"]["FeedData"];
export type WeatherData = components["schemas"]["WeatherData"];
export type Snapshot = components["schemas"]["Snapshot"];

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly retryAfter?: number,
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
      Number(response.headers.get("Retry-After")) || undefined,
    );
  return body as T;
}
