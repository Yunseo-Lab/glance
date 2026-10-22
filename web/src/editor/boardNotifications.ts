// Cross-tab messages contain IDs/revisions only. The server's CAS check remains
// authoritative for other devices and browsers without BroadcastChannel.
export const boardNotifications =
  typeof BroadcastChannel === "undefined"
    ? undefined
    : new BroadcastChannel("glance:board-revisions");
export function announceBoard(id: string, revision: number) {
  boardNotifications?.postMessage({ id, revision });
}
