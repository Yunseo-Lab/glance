import { useEffect, useRef, useState } from "react";
export type WorkspacePage =
  | "board"
  | "sources"
  | "inbox"
  | "settings"
  | "not-found";
export interface WorkspaceRoute {
  page: WorkspacePage;
  boardID: string;
}
function parseRoute(hash: string, boardID = "home"): WorkspaceRoute {
  if (!hash || hash === "#main" || hash === "#/")
    return { page: "board", boardID: "home" };
  const board = /^#\/boards\/([a-zA-Z0-9_-]{1,64})$/.exec(hash);
  if (board) return { page: "board", boardID: board[1] };
  const [page, query] = hash.split("?");
  if (["#/sources", "#/inbox", "#/settings"].includes(page)) {
    const context = new URLSearchParams(query).get("board");
    if (context && /^[a-zA-Z0-9_-]{1,64}$/.test(context)) boardID = context;
    return { page: page.slice(2) as WorkspacePage, boardID };
  }
  return { page: "not-found", boardID };
}
function routeHash(route: WorkspaceRoute) {
  return route.page === "board"
    ? `#/boards/${route.boardID}`
    : `#/${route.page}${route.boardID !== "home" ? `?board=${route.boardID}` : ""}`;
}
export function useWorkspaceNavigation(canChangeBoard: () => boolean) {
  const [route, setRoute] = useState(() => parseRoute(window.location.hash));
  const current = useRef(route),
    guard = useRef(canChangeBoard);
  guard.current = canChangeBoard;
  useEffect(() => {
    const change = () => {
      if (window.location.hash === "#main") return;
      const next = parseRoute(window.location.hash, current.current.boardID);
      if (next.boardID !== current.current.boardID && !guard.current()) {
        history.replaceState(null, "", routeHash(current.current));
        return;
      }
      current.current = next;
      setRoute(next);
      requestAnimationFrame(() =>
        document.getElementById("main")?.focus({ preventScroll: true }),
      );
    };
    window.addEventListener("hashchange", change);
    return () => window.removeEventListener("hashchange", change);
  }, []);
  const navigate = (page: WorkspacePage, boardID = current.current.boardID) => {
    window.location.hash = routeHash({ page, boardID });
  };
  return { route, navigate };
}
