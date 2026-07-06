interface MessageMediaDescriptor {
  id: string;
  message_type?: string;
  media_filename?: string;
  media_mime_type?: string;
}

const MIME_TO_EXTENSION: Record<string, string> = {
  "application/pdf": "pdf",
  "application/msword": "doc",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
    "docx",
  "application/vnd.ms-excel": "xls",
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": "xlsx",
  "application/vnd.ms-powerpoint": "ppt",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation":
    "pptx",
  "application/rtf": "rtf",
  "application/vnd.oasis.opendocument.text": "odt",
  "application/vnd.oasis.opendocument.spreadsheet": "ods",
  "application/vnd.oasis.opendocument.presentation": "odp",
  "text/plain": "txt",
  "text/csv": "csv",
  "image/jpeg": "jpg",
  "image/png": "png",
  "image/webp": "webp",
  "image/gif": "gif",
  "image/tiff": "tiff",
};

const PRINT_FALLBACK_DELAY_MS = 900;
const PRINT_FRAME_CLEANUP_DELAY_MS = 1800;
const PRINT_AFTER_LOAD_DELAY_MS = 250;

function sanitizeFilename(filename: string): string {
  return filename.replace(/[\\/:*?"<>|]/g, "_").trim();
}

function hasExtension(filename: string): boolean {
  return /\.[A-Za-z0-9]{1,8}$/u.test(filename);
}

function extensionFromMime(mimeType?: string): string {
  if (!mimeType) return "";
  return MIME_TO_EXTENSION[mimeType] || "";
}

function buildFallbackBasename(message: MessageMediaDescriptor): string {
  const messageType = (message.message_type || "").trim().toLowerCase();
  if (messageType === "image") return "photo";
  if (messageType === "document") return "document";
  if (messageType) return messageType;
  return "file";
}

export function resolveMediaFilename(message: MessageMediaDescriptor): string {
  const explicitFilename = sanitizeFilename(message.media_filename || "");
  const mimeExtension = extensionFromMime(message.media_mime_type);

  if (explicitFilename) {
    if (hasExtension(explicitFilename) || !mimeExtension) {
      return explicitFilename;
    }
    return `${explicitFilename}.${mimeExtension}`;
  }

  const fallbackBase = `${buildFallbackBasename(message)}_${message.id}`;
  if (!mimeExtension) {
    return fallbackBase;
  }
  return `${fallbackBase}.${mimeExtension}`;
}

export function downloadMediaFromUrl(mediaUrl: string, filename: string): void {
  const anchor = document.createElement("a");
  anchor.href = mediaUrl;
  anchor.download = filename;
  anchor.style.display = "none";
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
}

export function downloadMessageMedia(
  mediaUrl: string,
  message: MessageMediaDescriptor,
): void {
  downloadMediaFromUrl(mediaUrl, resolveMediaFilename(message));
}

function openPrintDialogInNewTab(mediaUrl: string): boolean {
  const printWindow = window.open("", "_blank");
  if (!printWindow) {
    return false;
  }

  const printInWindow = () => {
    try {
      printWindow.focus();
      printWindow.print();
    } catch {
      // Ignore print invocation failures.
    }
  };

  const fallbackTimer = window.setTimeout(
    printInWindow,
    PRINT_FALLBACK_DELAY_MS,
  );

  try {
    printWindow.addEventListener(
      "load",
      () => {
        window.clearTimeout(fallbackTimer);
        printInWindow();
      },
      { once: true },
    );
    printWindow.location.href = mediaUrl;
  } catch {
    window.clearTimeout(fallbackTimer);
    printWindow.close();
    return false;
  }

  return true;
}

function openPrintDialogInCurrentTab(mediaUrl: string): boolean {
  if (!document.body) {
    return openPrintDialogInNewTab(mediaUrl);
  }

  const iframe = document.createElement("iframe");
  iframe.style.position = "fixed";
  iframe.style.left = "-99999px";
  iframe.style.top = "0";
  iframe.style.width = "100vw";
  iframe.style.height = "100vh";
  iframe.style.opacity = "0.001";
  iframe.style.pointerEvents = "none";
  iframe.style.border = "0";
  iframe.setAttribute("aria-hidden", "true");
  iframe.tabIndex = -1;

  let handled = false;
  let fallbackTimer = 0;

  const cleanup = () => {
    iframe.onload = null;
    iframe.onerror = null;
    if (iframe.parentNode) {
      iframe.parentNode.removeChild(iframe);
    }
  };

  const triggerPrint = () => {
    const frameWindow = iframe.contentWindow;
    if (!frameWindow) {
      return false;
    }
    try {
      frameWindow.focus();
      frameWindow.print();
      return true;
    } catch {
      return false;
    }
  };

  const handleReadyToPrint = () => {
    if (handled) return;
    handled = true;
    window.clearTimeout(fallbackTimer);
    window.setTimeout(() => {
      if (!triggerPrint()) {
        openPrintDialogInNewTab(mediaUrl);
      }
      window.setTimeout(cleanup, PRINT_FRAME_CLEANUP_DELAY_MS);
    }, PRINT_AFTER_LOAD_DELAY_MS);
  };

  iframe.onload = () => {
    handleReadyToPrint();
  };

  iframe.onerror = () => {
    if (handled) return;
    handled = true;
    window.clearTimeout(fallbackTimer);
    openPrintDialogInNewTab(mediaUrl);
    cleanup();
  };

  document.body.appendChild(iframe);
  fallbackTimer = window.setTimeout(handleReadyToPrint, PRINT_FALLBACK_DELAY_MS);
  iframe.src = mediaUrl;
  return true;
}

export function openPrintDialogForMedia(mediaUrl: string): boolean {
  if (!mediaUrl) return false;
  return openPrintDialogInCurrentTab(mediaUrl);
}
