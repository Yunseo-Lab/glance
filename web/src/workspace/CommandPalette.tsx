import {useModalDialog} from "./useModalDialog";
import { useEffect, useRef, useState } from "react";
import { Command, Search, X } from "lucide-react";
import type { Board } from "../types";
import type { WorkspacePage } from "./navigation";
export default function CommandPalette({
  boards,
  onNavigate,
  onAdd,
  onClose,
}: {
  boards: Board[];
  onNavigate: (page: WorkspacePage, boardID?: string) => void;
  onAdd: () => void;
  onClose: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [query, setQuery] = useState(""),
    [active, setActive] = useState(0);
  const commands = [
    { id: "add", title: "위젯 추가", hint: "이 보드에 추가", run: onAdd },
    ...(
      [
        { page: "board", title: "대시보드로 이동" },
        { page: "sources", title: "데이터 소스 관리" },
        { page: "inbox", title: "모아 읽기" },
        { page: "brief", title: "하루 브리핑" },
        { page: "settings", title: "설정 열기" },
      ] as const
    ).map((item) => ({
      id: item.page,
      title: item.title,
      hint: "화면 이동",
      run: () => onNavigate(item.page),
    })),
    ...boards.map((board) => ({
      id: board.id,
      title: `보드: ${board.title}`,
      hint: "보드 이동",
      run: () => onNavigate("board", board.id),
    })),
  ].filter((command) =>
    [command.title, command.hint]
      .join(" ")
      .toLocaleLowerCase()
      .includes(query.toLocaleLowerCase()),
  );
  const choose = (index: number) => {
    const command = commands[index];
    if (!command) return;
    onClose();
    requestAnimationFrame(command.run);
  };
  useModalDialog(dialog,'input');
  useEffect(() => {
    document
      .getElementById(`command-option-${active}`)
      ?.scrollIntoView({ block: "nearest" });
  }, [active]);
  return (
    <dialog
      className="command-palette"
      ref={dialog}
      aria-labelledby="command-title"
      onCancel={onClose}
    >
      <div className="dialog-header">
        <h2 id="command-title">
          <Command size={18} /> 무엇을 할까요?
        </h2>
        <button
          className="icon-button"
          aria-label="명령 팔레트 닫기"
          onClick={onClose}
        >
          <X size={18} />
        </button>
      </div>
      <label className="command-search">
        <Search size={18} />
        <span className="sr-only">명령 검색</span>
        <input
          autoFocus
          role="combobox"
          aria-expanded="true"
          aria-controls="command-results"
          aria-activedescendant={
            commands.length ? `command-option-${active}` : undefined
          }
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            setActive(0);
          }}
          onKeyDown={(event) => {
            if (event.nativeEvent.isComposing) return;
            if (event.key === "ArrowDown") {
              event.preventDefault();
              setActive((value) => (value + 1) % Math.max(1, commands.length));
            }
            if (event.key === "ArrowUp") {
              event.preventDefault();
              setActive(
                (value) =>
                  (value - 1 + commands.length) % Math.max(1, commands.length),
              );
            }
            if (event.key === "Enter") {
              event.preventDefault();
              choose(active);
            }
          }}
          placeholder="화면, 보드, 위젯 추가를 검색하세요"
        />
      </label>
      <div
        role="listbox"
        id="command-results"
        aria-label="명령 결과"
        className="command-results"
      >
        {commands.map((command, index) => (
          <div
            id={`command-option-${index}`}
            key={command.id}
            role="option"
            aria-selected={active === index}
            onMouseDown={(event) => event.preventDefault()}
            onMouseMove={() => setActive(index)}
            onClick={() => choose(index)}
          >
            <span>{command.title}</span>
            <small>{command.hint}</small>
          </div>
        ))}
      </div>
      {!commands.length && (
        <p role="status" className="form-help">
          일치하는 명령이 없습니다.
        </p>
      )}
      <p className="command-help">
        ↑ ↓ 선택 · Enter 실행 · Esc 닫기 · Ctrl/⌘ K 열기
      </p>
    </dialog>
  );
}
