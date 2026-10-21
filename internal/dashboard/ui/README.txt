Run `npm ci && npm run build` in web/ before building a distributable Go binary.
Vite writes the production assets into this directory's ignored dist/ folder.
The Go binary embeds those assets; Node is only needed at build time.
