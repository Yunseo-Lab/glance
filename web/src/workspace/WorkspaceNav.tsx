import { Database, Inbox, LayoutDashboard, Settings } from "lucide-react";
import type { Board } from "../types";
import type { WorkspacePage, WorkspaceRoute } from "./navigation";
export default function WorkspaceNav({
  route,
  navigate,
  boards,
  mobile = false,
}: {
  route: WorkspaceRoute;
  navigate: (page: WorkspacePage, boardID?: string) => void;
  boards: Board[];
  mobile?: boolean;
}) {
  return (
    <nav
      className={mobile ? "workspace-mobile-nav" : "workspace-navigation"}
      aria-label={mobile ? "모바일 작업 공간" : "작업 공간"}
    >
      <label className="workspace-board-picker">
        <span className="sr-only">
          {mobile ? "모바일 보드 선택" : "보드 선택"}
        </span>
        <select
          aria-label={mobile ? "모바일 보드 선택" : "보드 선택"}
          value={route.boardID}
          onChange={(event) => navigate("board", event.target.value)}
        >
          {boards.length ? (
            boards.map((board) => (
              <option key={board.id} value={board.id}>
                {board.title}
              </option>
            ))
          ) : (
            <option value={route.boardID}>나의 보드</option>
          )}
        </select>
      </label>
      {(
        [
          { page: "board", title: "대시보드", Icon: LayoutDashboard },
          { page: "sources", title: "데이터 소스", Icon: Database },
          { page: "inbox", title: "모아 읽기", Icon: Inbox },
          { page: "settings", title: "설정", Icon: Settings },
        ] as const
      ).map(({ page, title, Icon }) => (
        <button
          key={page}
          className={`nav-item ${route.page === page ? "active" : ""}`}
          aria-label={title}
          aria-current={route.page === page ? "page" : undefined}
          onClick={() => navigate(page)}
        >
          <Icon size={17} />
          {!mobile && <span>{title}</span>}
        </button>
      ))}
    </nav>
  );
}
