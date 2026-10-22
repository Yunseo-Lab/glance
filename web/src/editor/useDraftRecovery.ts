import { useEffect, useState } from "react";
import type { Board } from "../types";
import {
  decodeDraft,
  draftPreference,
  draftPrefix,
  encodeDraft,
  purgeDeviceDrafts,
} from "./draftRecovery";
export function useDraftRecovery(
  saved: Board | undefined,
  draft: Board | undefined,
  dirty: boolean,
  authRequired: boolean,
) {
  const [enabled, setEnabled] = useState(() => {
      try {
        return localStorage.getItem(draftPreference) === "true";
      } catch {
        return false;
      }
    }),
    [candidate, setCandidate] = useState<Board>(),
    [ready, setReady] = useState(""),
    [error, setError] = useState("");
  useEffect(() => {
    const clear = () => {
      purgeDeviceDrafts();
      setEnabled(false);
      setCandidate(undefined);
      setReady("");
    };
    window.addEventListener("glance:logout", clear);
    return () => window.removeEventListener("glance:logout", clear);
  }, []);
  useEffect(() => {
    if (!saved || authRequired) return;
    try {
      const key = draftPrefix + saved.id,
        text = enabled ? localStorage.getItem(key) : null;
      const recovered = text ? decodeDraft(text, saved) : undefined;
      setCandidate(recovered);
      if (text && !recovered) localStorage.removeItem(key);
      setReady(saved.id);
    } catch {
      setError("이 브라우저에서는 초안을 보관할 수 없습니다.");
    }
  }, [saved?.id, enabled, authRequired]);
  useEffect(() => {
    if (
      !enabled ||
      !saved ||
      !draft ||
      authRequired ||
      ready !== saved.id ||
      candidate
    )
      return;
    try {
      const key = draftPrefix + saved.id;
      if (!dirty) {
        localStorage.removeItem(key);
        setError("");
        return;
      }
      const encoded = encodeDraft(saved, draft);
      if (!encoded) {
        localStorage.removeItem(key);
        setError(
          "새 소스 설정은 기기에 보관하지 않습니다. 대시보드 저장으로 보관하세요.",
        );
        return;
      }
      localStorage.setItem(key, encoded);
      setError("");
    } catch {
      setError(
        "초안을 기기에 보관하지 못했습니다. 대시보드 저장으로 보관하세요.",
      );
    }
  }, [saved, draft, dirty, enabled, ready, candidate, authRequired]);
  function toggle(value: boolean) {
    setEnabled(value);
    try {
      if (value) localStorage.setItem(draftPreference, "true");
      else purgeDeviceDrafts();
    } catch {
      setError("이 브라우저에서는 초안을 보관할 수 없습니다.");
    }
  }
  function dismiss() {
    if (saved)
      try {
        localStorage.removeItem(draftPrefix + saved.id);
      } catch {}
    setCandidate(undefined);
  }
  return { enabled, toggle, candidate, dismiss, error };
}
