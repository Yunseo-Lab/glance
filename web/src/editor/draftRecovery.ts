import { widgetCatalog } from "../catalog";
import type { Board } from "../types";
export const draftPrefix = "glance:draft:";
export const draftPreference = "glance:draft-opt-in";
export const draftTTL = 24 * 60 * 60 * 1000;
type Envelope = {
  version: 1;
  expiresAt: number;
  revision: number;
  sourceIds: string[];
  board: Board;
};
export function encodeDraft(
  saved: Board,
  draft: Board,
  now = Date.now(),
): string | undefined {
  // Never copy provider configuration (URLs/credentials) to device storage.
  // Pending source edits must first be saved on the server; local text/layout
  // edits can recover using source IDs already present in the saved revision.
  if (
    saved.id !== draft.id ||
    draft.revision !== saved.revision ||
    draft.sources.some(
      (source) =>
        !saved.sources.some(
          (base) =>
            base.id === source.id &&
            JSON.stringify(base) === JSON.stringify(source),
        ),
    )
  )
    return;
  const allowed = new Set<string>(widgetCatalog.map((widget) => widget.type));
  if (draft.widgets.some((widget) => !allowed.has(widget.type))) return;
  const board = structuredClone(draft);
  board.sources = [];
  const text = JSON.stringify({
    version: 1,
    expiresAt: now + draftTTL,
    revision: saved.revision,
    sourceIds: draft.sources.map((source) => source.id),
    board,
  } satisfies Envelope);
  return new TextEncoder().encode(text).length <= 1_048_576 ? text : undefined;
}
export function decodeDraft(
  text: string,
  saved: Board,
  now = Date.now(),
): Board | undefined {
  try {
    if (text.length > 1_048_576) return;
    const data: Envelope = JSON.parse(text);
    if (
      data.version !== 1 ||
      !Number.isFinite(data.expiresAt) ||
      data.expiresAt <= now ||
      data.expiresAt > now + draftTTL ||
      data.revision !== saved.revision ||
      data.board?.id !== saved.id ||
      data.board.revision !== saved.revision ||
      !Array.isArray(data.sourceIds) ||
      data.sourceIds.some(
        (id) =>
          typeof id !== "string" ||
          !saved.sources.some((source) => source.id === id),
      )
    )
      return;
    return {
      ...data.board,
      sources: saved.sources.filter((source) =>
        data.sourceIds.includes(source.id),
      ),
    };
  } catch {
    return;
  }
}
export function purgeDeviceDrafts() {
  try {
    for (const key of Object.keys(localStorage))
      if (key.startsWith(draftPrefix) || key === draftPreference)
        localStorage.removeItem(key);
  } catch {}
}
