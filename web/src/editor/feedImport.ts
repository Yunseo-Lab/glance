import type { components } from "../generated/api";
import type { Board } from "../types";
export type FeedImportItem = components["schemas"]["FeedImportItem"];
function canonical(value: string) {
  const url = new URL(value);
  url.hash = "";
  url.searchParams.sort();
  return url.href;
}
export function appendFeeds(
  board: Board,
  feeds: FeedImportItem[],
  newID = () => crypto.randomUUID(),
): Board {
  const seen = new Set(
    board.sources
      .filter((source) => source.type === "rss" && source.config.url)
      .map((source) => canonical(source.config.url!)),
  );
  const pending: FeedImportItem[] = [];
  for (const feed of feeds) {
    const key = canonical(feed.url);
    if (seen.has(key)) continue;
    seen.add(key);
    pending.push(feed);
  }
  if (
    board.widgets.length + pending.length > 32 ||
    board.sources.length + pending.length > 32
  )
    throw new Error(
      "이 보드에는 위젯과 소스를 각각 32개까지 둘 수 있습니다. 가져올 피드를 줄여주세요.",
    );
  const next = structuredClone(board);
  for (const feed of pending) {
    const sourceID = newID();
    let title = "";
    for (const character of feed.title) {
      if (new TextEncoder().encode(title + character).length > 100) break;
      title += character;
    }
    next.sources.push({
      id: sourceID,
      type: "rss",
      configVersion: 1,
      config: { url: feed.url },
    });
    next.widgets.push({
      id: newID(),
      type: "rss",
      configVersion: 1,
      title: title || new URL(feed.url).hostname,
      size: "wide",
      sourceId: sourceID,
      config: { limit: 5 },
    });
  }
  return next;
}
