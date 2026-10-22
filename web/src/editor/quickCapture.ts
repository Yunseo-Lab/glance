import type { Board, Widget } from "../types";
import { validateWidgetDetail } from "../validation";
export type Capture = {
  kind: "note" | "bookmark";
  title: string;
  body: string;
  url: string;
};
export function captureToBoard(
  board: Board,
  input: Capture,
  newID = () => crypto.randomUUID(),
): Board {
  const next = structuredClone(board);
  let widget: Widget;
  if (input.kind === "note") {
    widget = {
      id: newID(),
      configVersion: 1,
      type: "note",
      title: input.title.trim() || "빠른 메모",
      size: "small",
      config: { body: input.body },
    };
    if (!input.body.trim()) throw new Error("메모 내용을 입력하세요.");
    const failure = validateWidgetDetail(widget);
    if (failure) throw new Error(failure.message);
    if (next.widgets.length >= 32)
      throw new Error("보드에 위젯이 32개 있어 새 메모를 둘 수 없습니다.");
    next.widgets.push(widget);
  } else {
    const title =
      input.title.trim() ||
      (() => {
        try {
          return new URL(input.url).hostname;
        } catch {
          return "바로가기";
        }
      })();
    const link = { title, url: input.url.trim() };
    const existing = next.widgets.find(
      (widget) =>
        widget.type === "bookmarks" && (widget.config.links?.length || 0) < 20,
    );
    widget = existing
      ? {
          ...existing,
          config: { links: [...(existing.config.links || []), link] },
        }
      : {
          id: newID(),
          configVersion: 1,
          type: "bookmarks",
          title: "빠른 즐겨찾기",
          size: "small",
          config: { links: [link] },
        };
    const failure = validateWidgetDetail(widget);
    if (failure) throw new Error(failure.message);
    if (
      next.widgets.some(
        (widget) =>
          widget.type === "bookmarks" &&
          widget.config.links?.some((item) => {
            try {
              return new URL(item.url).href === new URL(link.url).href;
            } catch {
              return false;
            }
          }),
      )
    )
      throw new Error("이 주소는 이미 즐겨찾기에 있습니다.");
    if (existing)
      next.widgets[next.widgets.findIndex((item) => item.id === existing.id)] =
        widget;
    else {
      if (next.widgets.length >= 32)
        throw new Error("보드에 위젯이 32개 있어 즐겨찾기를 둘 수 없습니다.");
      next.widgets.push(widget);
    }
  }
  return next;
}
