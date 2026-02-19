// Helpers for converting image files to base64 ImageData payloads.
import type { ImageData as APIImageData } from "@sdk/types.gen";

const ALLOWED_TYPES = new Set(["image/png", "image/jpeg", "image/gif", "image/webp"]);

/** Convert a File to an APIImageData, or null if the type is unsupported. */
export async function fileToImageData(file: File): Promise<APIImageData | null> {
  if (!ALLOWED_TYPES.has(file.type)) return null;
  const buf = await file.arrayBuffer();
  const data = btoa(String.fromCharCode(...new Uint8Array(buf)));
  return { mediaType: file.type, data };
}

/** Extract image files from a paste event and convert them. */
export async function imagesFromClipboard(e: ClipboardEvent): Promise<APIImageData[]> {
  const items = e.clipboardData?.items;
  if (!items) return [];
  const files: File[] = [];
  for (const item of items) {
    if (item.kind === "file" && ALLOWED_TYPES.has(item.type)) {
      const f = item.getAsFile();
      if (f) files.push(f);
    }
  }
  const results = await Promise.all(files.map(fileToImageData));
  return results.filter((r): r is APIImageData => r !== null);
}
