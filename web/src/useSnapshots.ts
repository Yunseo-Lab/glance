import { useQuery } from "@tanstack/react-query";
import type { Snapshot } from "./types";
import { queryClient, queryOptions } from "./workspace/queries";
export function useSnapshots(revision?: number, boardID = "home") {
  const enabled = revision !== undefined;
  const result = useQuery(
    {
      ...queryOptions<Record<string, Snapshot>>(
        boardID === "home"
          ? "/api/v1/snapshots"
          : `/api/v1/boards/${boardID}/snapshots`,
      ),
      enabled,
      refetchInterval: (query) =>
        Object.values(query.state.data || {}).some(
          (snapshot) =>
            snapshot.state === "loading" ||
            (snapshot.state === "stale" && !snapshot.error),
        )
          ? 2000
          : 30_000,
      refetchIntervalInBackground: false,
    },
    queryClient,
  );
  return {
    snapshots: enabled ? result.data || {} : {},
    error: result.error
      ? "소식을 불러올 수 없습니다. 연결되면 자동으로 다시 시도합니다."
      : "",
    retry: () => {
      void result.refetch();
    },
  };
}
