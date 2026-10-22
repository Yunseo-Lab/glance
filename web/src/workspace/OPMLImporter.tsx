import {useModalDialog} from "./useModalDialog";
import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { request } from "../types";
import type { components } from "../generated/api";
import type { FeedImportItem } from "../editor/feedImport";
export default function OPMLImporter({
  onClose,
  onApply,
}: {
  onClose: () => void;
  onApply: (feeds: FeedImportItem[]) => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [preview, setPreview] =
      useState<components["schemas"]["FeedImportPreview"]>(),
    [selected, setSelected] = useState<Set<string>>(new Set()),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  useModalDialog(dialog,'input');
  async function load(file: File) {
    setError("");
    setPreview(undefined);
    setBusy(true);
    try {
      if (file.size > 524288)
        throw new Error("OPML 파일은 512KiB 이하여야 합니다.");
      const result = await request<components["schemas"]["FeedImportPreview"]>(
        "/api/v1/imports/opml/preview",
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ document: await file.text() }),
        },
      );
      setPreview(result);
      setSelected(new Set(result.feeds.map((feed) => feed.url)));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <dialog
      ref={dialog}
      aria-labelledby="opml-title"
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
    >
      <div className="dialog-header">
        <h2 id="opml-title">피드 묶음 가져오기</h2>
        <button
          className="icon-button"
          disabled={busy}
          aria-label="피드 가져오기 닫기"
          onClick={onClose}
        >
          <X size={18} />
        </button>
      </div>
      <label>
        OPML 파일
        <input
          aria-label="OPML 파일"
          type="file"
          accept=".opml,.xml,text/xml,application/xml"
          disabled={busy}
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (file) void load(file);
          }}
        />
      </label>
      <p className="form-help">
        공개 HTTPS RSS·RSSHub·RSS-Bridge 피드 주소를 가져옵니다. 개인 토큰이 든
        주소는 포함하지 마세요. 미리보기에서는 정보를 가져오거나 저장하지
        않습니다.
      </p>
      {busy && <p role="status">피드 목록을 확인하고 있어요.</p>}
      {error && (
        <p role="alert" className="error-text">
          {error}
        </p>
      )}
      {preview && (
        <>
          <p role="status">
            피드 {preview.feeds.length}개 · 중복 {preview.duplicateCount}개 제외
          </p>
          <div className="opml-feeds">
            {preview.feeds.map((feed) => (
              <label key={feed.url}>
                <input
                  type="checkbox"
                  checked={selected.has(feed.url)}
                  onChange={(event) =>
                    setSelected((current) => {
                      const next = new Set(current);
                      if (event.target.checked) next.add(feed.url);
                      else next.delete(feed.url);
                      return next;
                    })
                  }
                />
                <span>
                  <strong>{feed.title}</strong>
                  <small>
                    {feed.folder && `${feed.folder} · `}
                    {new URL(feed.url).hostname}
                  </small>
                </span>
              </label>
            ))}
          </div>
          {preview.feeds.length === 0 && <p>가져올 피드가 없습니다.</p>}
          <div className="dialog-footer">
            <button className="button secondary" onClick={onClose}>
              취소
            </button>
            <button
              className="button primary"
              disabled={busy || selected.size === 0}
              onClick={() => {
                try {
                  onApply(
                    preview.feeds.filter((feed) => selected.has(feed.url)),
                  );
                  onClose();
                } catch (err) {
                  setError((err as Error).message);
                }
              }}
            >
              선택한 {selected.size}개 초안에 추가
            </button>
          </div>
        </>
      )}
    </dialog>
  );
}
