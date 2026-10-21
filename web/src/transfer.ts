import type { Board } from "./types";

export interface DashboardFile {
  format: "glance-dashboard";
  version: 1;
  board: Board;
}

export function parseDashboardFile(text: string): Board {
  let value: unknown;
  try {
    value = JSON.parse(text);
  } catch {
    throw new Error("올바른 JSON 설정 파일을 선택하세요.");
  }
  if (
    !value ||
    typeof value !== "object" ||
    !("format" in value) ||
    value.format !== "glance-dashboard" ||
    !("version" in value) ||
    value.version !== 1 ||
    !("board" in value) ||
    !value.board ||
    typeof value.board !== "object" ||
    Array.isArray(value.board)
  ) {
    throw new Error("지원하지 않는 설정 파일 형식 또는 버전입니다.");
  }
  // This value is untrusted until the server validates it; do not render it yet.
  return value.board as Board;
}

export function downloadBoard(board: Board) {
  const file: DashboardFile = { format: "glance-dashboard", version: 1, board };
  const blob = new Blob([JSON.stringify(file, null, 2)], {
    type: "application/json",
  });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = "glance-dashboard.json";
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  // Leave time for browsers to consume the download before releasing its URL.
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}
