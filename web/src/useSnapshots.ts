import { useEffect, useRef, useState } from "react";
import { request, type Snapshot } from "./types";

export function useSnapshots(revision?: number) {
  const [snapshots, setSnapshots] = useState<Record<string, Snapshot>>({});
  const [error, setError] = useState("");
  const retry = useRef<() => void>(() => {});

  useEffect(() => {
    if (revision === undefined) return;
    let stopped = false,
      fetching = false;
    let timer: number | undefined;
    const controller = new AbortController();
    async function poll() {
      clearTimeout(timer);
      if (stopped || document.hidden || fetching) return;
      fetching = true;
      let interval = 30_000;
      try {
        const result = await request<Record<string, Snapshot>>(
          "/api/v1/snapshots",
          { signal: controller.signal },
        );
        if (!stopped) {
          setSnapshots(result);
          setError("");
          if (
            Object.values(result).some(
              (s) => s.state === "loading" || (s.state === "stale" && !s.error),
            )
          )
            interval = 2000;
        }
      } catch {
        if (!stopped)
          setError(
            "소식을 불러올 수 없습니다. 연결되면 자동으로 다시 시도합니다.",
          );
        interval = 5000;
      } finally {
        fetching = false;
        if (!stopped && !document.hidden)
          timer = window.setTimeout(poll, interval);
      }
    }
    const wake = () => {
      if (document.hidden) clearTimeout(timer);
      else void poll();
    };
    retry.current = () => void poll();
    window.addEventListener("online", wake);
    window.addEventListener("glance:refresh", wake);
    document.addEventListener("visibilitychange", wake);
    void poll();
    return () => {
      stopped = true;
      controller.abort();
      clearTimeout(timer);
      retry.current = () => {};
      window.removeEventListener("online", wake);
      window.removeEventListener("glance:refresh", wake);
      document.removeEventListener("visibilitychange", wake);
    };
  }, [revision]);
  return { snapshots, error, retry: () => retry.current() };
}
