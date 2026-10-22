import { createHash } from "node:crypto";
import type { Plugin } from "vite";
// Only build outputs are precached. API responses and owner data never enter CacheStorage.
export function appShell(): Plugin {
  return {
    name: "glance-public-shell",
    apply: "build",
    generateBundle(_, bundle) {
      const files = [
        "/",
        "/manifest.webmanifest",
        "/icon.svg",
        "/icon-192.png",
        "/icon-512.png",
        ...Object.keys(bundle)
          .filter((name) => name.startsWith("assets/"))
          .map((name) => "/" + name),
      ];
      const hash = createHash("sha256");
      for (const [name, output] of Object.entries(bundle).sort(([a], [b]) =>
        a.localeCompare(b),
      )) {
        hash.update(name);
        hash.update(output.type === "chunk" ? output.code : output.source);
      }
      const version = hash.digest("hex").slice(0, 16);
      this.emitFile({
        type: "asset",
        fileName: "sw.js",
        source: String.raw`
const CACHE = 'glance-shell-${version}';
const FILES = ${JSON.stringify(files)};
self.addEventListener('install', event => event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(FILES))));
self.addEventListener('activate', event => event.waitUntil((async () => {
  const older = (await caches.keys()).filter(key => key.startsWith('glance-shell-') && key !== CACHE);
  for (const key of older.slice(0, -1)) await caches.delete(key);
  await self.clients.claim();
})()));
self.addEventListener('message', event => { if (event.data === 'ACTIVATE_UPDATE') self.skipWaiting(); });
self.addEventListener('fetch', event => {
  const url = new URL(event.request.url);
  if (event.request.method !== 'GET' || url.origin !== self.location.origin || url.pathname.startsWith('/api/')) return;
  if (event.request.mode === 'navigate' && url.pathname === '/') {
    event.respondWith(fetch(event.request).catch(() => caches.open(CACHE).then(cache => cache.match('/')))); return;
  }
  if (FILES.includes(url.pathname) || /^\/assets\/[a-zA-Z0-9_.-]+$/.test(url.pathname)) event.respondWith((async () => {
    for (const key of [CACHE, ...(await caches.keys()).filter(key => key.startsWith('glance-shell-') && key !== CACHE)]) { const hit = await (await caches.open(key)).match(url.pathname); if (hit) return hit; }
    return fetch(event.request);
  })());
});
`,
      });
    },
  };
}
