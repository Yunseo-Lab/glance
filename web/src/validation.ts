import type { Source, Widget } from "./types";
const bytes = (value: string) => new TextEncoder().encode(value).length;
function validPublicURL(value: string): boolean {
  try {
    const url = new URL(value);
    return (
      bytes(value) <= 2048 &&
      url.protocol === "https:" &&
      !url.username &&
      !url.password &&
      (!url.port || url.port === "443")
    );
  } catch {
    return false;
  }
}
export interface FieldFailure {
  field: string;
  message: string;
}
export function validateWidgetDetail(
  widget: Widget,
  source?: Source,
): FieldFailure | undefined {
  const fail = (field: string, message: string) => ({ field, message });
  if (!widget.title.trim()) return fail("title", "위젯 제목을 입력하세요.");
  if (bytes(widget.title) > 100)
    return fail("title", "위젯 제목이 너무 깁니다.");
  switch (widget.type) {
    case "clock":
      try {
        if (!widget.config.timezone || widget.config.timezone === "Local")
          throw new Error();
        new Intl.DateTimeFormat("ko-KR", { timeZone: widget.config.timezone });
      } catch {
        return fail("timezone", "시간대를 확인하세요. 예: Asia/Seoul");
      }
      break;
    case "note":
      if (bytes(widget.config.body || "") > 12000)
        return fail("body", "메모가 너무 깁니다.");
      break;
    case "bookmarks":
      if ((widget.config.links?.length || 0) > 20)
        return fail("links", "즐겨찾기는 최대 20개입니다.");
      for (const [index, link] of (widget.config.links || []).entries()) {
        if (!link.title.trim() || bytes(link.title) > 100)
          return fail(`links.${index}.title`, "즐겨찾기 이름을 확인하세요.");
        if (!validPublicURL(link.url))
          return fail(
            `links.${index}.url`,
            "인증 정보가 없는 HTTPS 주소를 입력하세요 (포트 443).",
          );
      }
      break;
    case "github":
      if (
        !source?.config.repository ||
        !/^([a-zA-Z0-9][a-zA-Z0-9-]{0,38})\/([a-zA-Z0-9_.-]{1,100})$/.test(
          source.config.repository,
        ) ||
        [".", ".."].includes(source.config.repository.split("/")[1])
      )
        return fail(
          "repository",
          "owner/repository 형식의 저장소를 입력하세요.",
        );
      if (
        !Number.isInteger(widget.config.limit) ||
        widget.config.limit! < 1 ||
        widget.config.limit! > 20
      )
        return fail("limit", "표시 개수는 1~20 사이의 정수로 입력하세요.");
      break;
    case "rss":
      if (
        !Number.isInteger(widget.config.limit) ||
        widget.config.limit! < 1 ||
        widget.config.limit! > 20
      )
        return fail("limit", "표시 개수는 1~20 사이의 정수로 입력하세요.");
      if (!source?.config.url || !validPublicURL(source.config.url))
        return fail(
          "url",
          "인증 정보가 없는 HTTPS 피드 주소를 입력하세요 (포트 443).",
        );
      break;
    case "weather": {
      const config = source?.config;
      if (!config?.location?.trim() || bytes(config.location) > 100)
        return fail("location", "지역 이름을 입력하세요.");
      if (
        !Number.isFinite(config.latitude) ||
        config.latitude! < -90 ||
        config.latitude! > 90
      )
        return fail("latitude", "위도와 경도를 확인하세요.");
      if (
        !Number.isFinite(config.longitude) ||
        config.longitude! < -180 ||
        config.longitude! > 180
      )
        return fail("longitude", "위도와 경도를 확인하세요.");
      if (config.units && !["celsius", "fahrenheit"].includes(config.units))
        return fail("units", "온도 단위를 확인하세요.");
      if (config.timezone)
        try {
          new Intl.DateTimeFormat("ko-KR", {
            timeZone: config.timezone,
          }).format();
        } catch {
          return fail("timezone", "올바른 시간대를 입력하세요.");
        }
      break;
    }
  }
}
// Immediate editor feedback; final save/import validation remains on the server.
export function validateWidget(
  widget: Widget,
  source?: Source,
): string | undefined {
  return validateWidgetDetail(widget, source)?.message;
}
