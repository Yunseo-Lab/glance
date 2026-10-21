export function safeExternalURL(value: string): string | undefined {
  try {
    const url = new URL(value);
    return url.protocol === "https:" &&
      !url.username &&
      !url.password &&
      (!url.port || url.port === "443") &&
      new TextEncoder().encode(value).length <= 2048
      ? url.href
      : undefined;
  } catch {
    return undefined;
  }
}
