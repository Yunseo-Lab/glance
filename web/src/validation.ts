import type { Source, Widget } from "./types";

const bytes = (value: string) => new TextEncoder().encode(value).length;

function validPublicURL(value: string): boolean {
  try {
    const url = new URL(value);
    return (
      value.length <= 2048 &&
      url.protocol === "https:" &&
      !url.username &&
      !url.password &&
      (!url.port || url.port === "443")
    );
  } catch {
    return false;
  }
}

// Fast feedback for the editor; the server remains authoritative on save/import.
export function validateWidget(
  widget: Widget,
  source?: Source,
): string | undefined {
  if (!widget.title.trim()) return "위젯 제목을 입력하세요.";
  if (bytes(widget.title) > 100) return "위젯 제목이 너무 깁니다.";
  switch (widget.type) {
    case "clock":
      try {
        if (!widget.config.timezone || widget.config.timezone === "Local")
          throw new Error();
        new Intl.DateTimeFormat("ko-KR", { timeZone: widget.config.timezone });
      } catch {
        return "시간대를 확인하세요. 예: Asia/Seoul";
      }
      break;
    case "note":
      if (bytes(widget.config.body || "") > 12000) return "메모가 너무 깁니다.";
      break;
    case "bookmarks":
      if ((widget.config.links?.length || 0) > 20)
        return "즐겨찾기는 최대 20개입니다.";
      for (const link of widget.config.links || []) {
        if (!link.title.trim() || bytes(link.title) > 100)
          return "즐겨찾기 이름을 확인하세요.";
        if (!validPublicURL(link.url))
          return "인증 정보가 없는 HTTPS 주소를 입력하세요 (포트 443).";
      }
      break;
    case "rss":
      if (
        !Number.isInteger(widget.config.limit) ||
        widget.config.limit! < 1 ||
        widget.config.limit! > 20
      )
        return "표시 개수는 1~20 사이의 정수로 입력하세요.";
      if (!source?.config.url || !validPublicURL(source.config.url))
        return "인증 정보가 없는 HTTPS 피드 주소를 입력하세요 (포트 443).";
      break;
    case "weather": {
      const config = source?.config;
      if (!config?.location?.trim() || bytes(config.location) > 100)
        return "지역 이름을 입력하세요.";
      if (
        !Number.isFinite(config.latitude) ||
        config.latitude! < -90 ||
        config.latitude! > 90 ||
        !Number.isFinite(config.longitude) ||
        config.longitude! < -180 ||
        config.longitude! > 180
      )
        return "위도와 경도를 확인하세요.";
      break;
    }
  }
}
