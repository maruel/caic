// Reusable prompt input with image support: paste, drag & drop, attach button, and preview strip.
import { createSignal, For, Show, type JSX } from "solid-js";
import type { ImageData as APIImageData } from "@sdk/types.gen";
import { fileToImageData, imagesFromClipboard } from "./images";
import AutoResizeTextarea from "./AutoResizeTextarea";
import Button from "./Button";
import AttachIcon from "@material-symbols/svg-400/outlined/attach_file.svg?solid";
import styles from "./PromptInput.module.css";

interface Props {
  value: string;
  onInput: (value: string) => void;
  onSubmit?: () => void;
  placeholder?: string;
  disabled?: boolean;
  class?: string;
  tabIndex?: number;
  ref?: (el: HTMLTextAreaElement) => void;
  "data-testid"?: string;
  // Image support
  supportsImages?: boolean;
  images: APIImageData[];
  onImagesChange: (imgs: APIImageData[]) => void;
  children?: JSX.Element;
}

export default function PromptInput(props: Props) {
  const [dragging, setDragging] = createSignal(false);

  function handlePaste(e: ClipboardEvent) {
    if (!props.supportsImages) return;
    // eslint-disable-next-line solid/reactivity -- event handler registered via addEventListener
    imagesFromClipboard(e).then((imgs) => {
      if (imgs.length > 0) props.onImagesChange([...props.images, ...imgs]);
    });
  }

  function handleDragOver(e: DragEvent) {
    if (!props.supportsImages) return;
    e.preventDefault();
    setDragging(true);
  }

  function handleDragLeave(e: DragEvent) {
    // Only clear when leaving the wrapper, not child elements.
    const wrapper = e.currentTarget as HTMLElement;
    if (wrapper.contains(e.relatedTarget as Node)) return;
    setDragging(false);
  }

  async function handleDrop(e: DragEvent) {
    e.preventDefault();
    setDragging(false);
    if (!props.supportsImages || !e.dataTransfer?.files.length) return;
    const imgs = await Promise.all(Array.from(e.dataTransfer.files).map(fileToImageData));
    const valid = imgs.filter((i): i is APIImageData => i !== null);
    if (valid.length > 0) props.onImagesChange([...props.images, ...valid]);
  }

  let fileInputRef!: HTMLInputElement;

  async function handleFileChange() {
    const files = fileInputRef.files;
    if (!files?.length) return;
    // Snapshot and reset synchronously to prevent double-processing when
    // browsers fire both "change" and "input" for the same selection.
    const snapshot = Array.from(files);
    fileInputRef.value = "";
    const imgs = await Promise.all(snapshot.map(fileToImageData));
    const valid = imgs.filter((i): i is APIImageData => i !== null);
    if (valid.length > 0) props.onImagesChange([...props.images, ...valid]);
  }

  function removeImage(idx: number) {
    props.onImagesChange(props.images.filter((_, i) => i !== idx));
  }

  return (
    <div
      class={`${styles.container}${dragging() ? ` ${styles.dragOver}` : ""}`}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <div class={styles.row}>
        <AutoResizeTextarea
          ref={(el) => {
            el.addEventListener("paste", handlePaste);
            props.ref?.(el);
          }}
          value={props.value}
          onInput={props.onInput}
          onSubmit={props.onSubmit}
          placeholder={props.placeholder}
          disabled={props.disabled}
          class={props.class}
          tabIndex={props.tabIndex}
          data-testid={props["data-testid"]}
        />
        <Show when={props.supportsImages}>
          <input
            ref={(el) => {
              fileInputRef = el;
              // Chrome Android may not fire "change" on file inputs; listen
              // to both "change" and "input" via addEventListener so at least
              // one fires. The guard in handleFileChange (files?.length + value
              // reset) prevents double-processing on browsers that fire both.
              el.addEventListener("change", handleFileChange);
              el.addEventListener("input", handleFileChange);
            }}
            type="file"
            multiple
            accept="image/png,image/jpeg,image/gif,image/webp"
            class={styles.hiddenFileInput}
          />
          <Button type="button" variant="gray" disabled={props.disabled} title="Attach images" onClick={() => fileInputRef.click()}>
            <AttachIcon width="1.2em" height="1.2em" />
          </Button>
        </Show>
        {props.children}
      </div>
      <Show when={props.images.length > 0}>
        <div class={styles.imagePreviewRow}>
          <For each={props.images}>
            {(img, idx) => (
              <div class={styles.imageThumb}>
                <img src={`data:${img.mediaType};base64,${img.data}`} alt="attached" />
                <button class={styles.imageRemove} onClick={() => removeImage(idx())} title="Remove">&times;</button>
              </div>
            )}
          </For>
        </div>
      </Show>
    </div>
  );
}
