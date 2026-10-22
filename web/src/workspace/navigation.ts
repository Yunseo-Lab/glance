import { useEffect, useRef, useState } from "react";
export type WorkspacePage =
  | "board"
  | "sources"
  | "inbox"
  | "brief"
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
  if (["#/sources", "#/inbox", "#/settings", "#/brief"].includes(page)) {
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
    guard = useRef(canChangeBoard),
    index = useRef<number>(history.state?.glanceIndex ?? 0),
    restoring = useRef(false);
  guard.current = canChangeBoard;
  function apply(next: WorkspaceRoute) {
    current.current = next;
    setRoute(next);
    requestAnimationFrame(() =>
      document.getElementById("main")?.focus({ preventScroll: true }),
    );
  }
  useEffect(() => {
    history.replaceState(
      { ...history.state, glanceIndex: index.current },
      "",
      window.location.href,
    );
    const change = () => {
      if (window.location.hash === "#main") return;
      if (restoring.current) {
        restoring.current = false;
        return;
      }
      const next = parseRoute(window.location.hash, current.current.boardID),
        target = history.state?.glanceIndex;
      if (next.boardID !== current.current.boardID && !guard.current()) {
        if (typeof target === "number" && target !== index.current) {
          restoring.current = true;
          history.go(index.current - target);
        } else
          history.replaceState(
            { ...history.state, glanceIndex: index.current },
            "",
            routeHash(current.current),
          );
        return;
      }
      index.current = typeof target === "number" ? target : index.current + 1;
      history.replaceState(
        { ...history.state, glanceIndex: index.current },
        "",
        window.location.href,
      );
      apply(next);
    };
    window.addEventListener("hashchange", change);
    return () => window.removeEventListener("hashchange", change);
  }, []);
  const navigate = (page: WorkspacePage, boardID = current.current.boardID) => {
    const next = { page, boardID };
    if (next.boardID !== current.current.boardID && !guard.current()) return;
    if (routeHash(next) === routeHash(current.current)) return;
    index.current++;
    history.pushState(
      { ...history.state, glanceIndex: index.current },
      "",
      routeHash(next),
    );
    apply(next);
  };
  return { route, navigate };
}
