import { useEffect, useState } from "react";
import { request } from "../types";
export function useResource<T>(path: string, interval = 0) {
  const [data, setData] = useState<T>(),
    [error, setError] = useState(""),
    [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    let timer: number | undefined,
      loading = false;
    async function load() {
      if (loading || controller.signal.aborted) return;
      loading = true;
      clearTimeout(timer);
      try {
        const result = await request<T>(path, { signal: controller.signal });
        if (!controller.signal.aborted) {
          setData(result);
          setError("");
        }
      } catch (err) {
        if (!controller.signal.aborted) setError((err as Error).message);
      } finally {
        loading = false;
        if (interval && !controller.signal.aborted)
          timer = window.setTimeout(load, interval);
      }
    }
    const wake = () => {
      if (!document.hidden) void load();
    };
    void load();
    window.addEventListener("online", wake);
    document.addEventListener("visibilitychange", wake);
    return () => {
      controller.abort();
      clearTimeout(timer);
      window.removeEventListener("online", wake);
      document.removeEventListener("visibilitychange", wake);
    };
  }, [path, interval, attempt]);
  return { data, error, reload: () => setAttempt((value) => value + 1) };
}
