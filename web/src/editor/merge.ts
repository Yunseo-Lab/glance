import type { Board, Source, Widget } from "../types";
import { pruneBoard } from "./reducer";
const equal = (a: unknown, b: unknown) =>
  JSON.stringify(a) === JSON.stringify(b);
export function mergeBoards(base: Board, local: Board, remote: Board) {
  if (base.id !== local.id || base.id !== remote.id)
    throw new Error("같은 보드만 비교할 수 있습니다.");
  const conflicts: string[] = [];
  function choose<T>(path: string, b: T, l: T, r: T): T {
    if (equal(l, b)) return r;
    if (!equal(r, b) && !equal(l, r)) conflicts.push(path);
    return l;
  }
  function entities<T extends { id: string }>(
    path: string,
    b: T[],
    l: T[],
    r: T[],
  ): T[] {
    const result: T[] = [];
    for (const id of new Set([
      ...r.map((item) => item.id),
      ...l.map((item) => item.id),
      ...b.map((item) => item.id),
    ])) {
      const value = choose(
        `${path}.${id}`,
        b.find((item) => item.id === id),
        l.find((item) => item.id === id),
        r.find((item) => item.id === id),
      );
      if (value) result.push(structuredClone(value));
    }
    return result;
  }
  const widgets = entities<Widget>(
    "widgets",
    base.widgets,
    local.widgets,
    remote.widgets,
  );
  const sources = entities<Source>(
    "sources",
    base.sources,
    local.sources,
    remote.sources,
  );
  const baseOrder = base.widgets.map((widget) => widget.id),
    localOrder = local.widgets.map((widget) => widget.id),
    remoteOrder = remote.widgets.map((widget) => widget.id);
  const order = choose("widgets.order", baseOrder, localOrder, remoteOrder);
  const ranks = new Map(order.map((id, index) => [id, index]));
  widgets.sort(
    (a, b) =>
      (ranks.get(a.id) ?? Number.MAX_SAFE_INTEGER) -
      (ranks.get(b.id) ?? Number.MAX_SAFE_INTEGER),
  );
  const board = pruneBoard({
    ...remote,
    title: choose("title", base.title, local.title, remote.title),
    theme: choose("theme", base.theme, local.theme, remote.theme),
    widgets,
    sources,
  });
  return { board, conflicts };
}
export function boardChanges(base: Board, next: Board): string[] {
  const changes: string[] = [];
  if (base.title !== next.title)
    changes.push(`보드 이름: ${base.title} → ${next.title}`);
  if (base.theme !== next.theme) changes.push("테마 변경");
  for (const widget of next.widgets) {
    const old = base.widgets.find((item) => item.id === widget.id);
    if (!old) changes.push(`위젯 추가: ${widget.title}`);
    else if (!equal(old, widget))
      changes.push(`위젯 설정 변경: ${widget.title}`);
  }
  for (const widget of base.widgets)
    if (!next.widgets.some((item) => item.id === widget.id))
      changes.push(`위젯 삭제: ${widget.title}`);
  if (
    !equal(
      base.widgets.map((widget) => widget.id),
      next.widgets.map((widget) => widget.id),
    )
  )
    changes.push("위젯 순서 또는 구성 변경");
  if (!equal(base.sources, next.sources)) changes.push("데이터 소스 구성 변경");
  return changes;
}
