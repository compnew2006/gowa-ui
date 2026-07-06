import { openPrintDialogForMedia } from "@/lib/media-actions";

const PDF_MIME_TYPE = "application/pdf";
const MERGED_PRINT_FILENAME = "merged-print.pdf";
const PRINT_URL_REVOKE_DELAY_MS = 120_000;
const A4_PAGE_WIDTH_POINTS = 595.28;
const A4_PAGE_HEIGHT_POINTS = 841.89;
const A4_PAGE_MARGIN_POINTS = 24;
const TIFF_MIME_TYPES = new Set([
  "image/tiff",
  "image/tif",
  "application/tiff",
  "application/x-tiff",
]);

export function isMergePrintableMimeType(mimeType: string): boolean {
  const normalized = String(mimeType || "").trim().toLowerCase();
  if (!normalized) return false;
  return normalized === PDF_MIME_TYPE || normalized.startsWith("image/");
}

export function isMergePrintableFile(file: Pick<File, "type">): boolean {
  return isMergePrintableMimeType(file.type);
}

async function toArrayBuffer(file: File): Promise<ArrayBuffer> {
  return file.arrayBuffer();
}

function normalize(value?: string | null): string {
  return String(value || "").trim().toLowerCase();
}

function isTiffLikeFilename(filename: string): boolean {
  const normalized = normalize(filename);
  return normalized.endsWith(".tif") || normalized.endsWith(".tiff");
}

function isTiffLikeFile(file: Pick<File, "type" | "name">): boolean {
  return TIFF_MIME_TYPES.has(normalize(file.type)) || isTiffLikeFilename(file.name);
}

function ensureDomDocumentAvailable(): void {
  if (typeof document === "undefined") {
    throw new Error("Document is not available in this runtime");
  }
}

async function canvasToPngBytes(
  canvas: HTMLCanvasElement,
  filename: string,
): Promise<Uint8Array> {
  const pngBlob = await new Promise<Blob | null>((resolve) =>
    canvas.toBlob(resolve, "image/png"),
  );
  if (!pngBlob) {
    throw new Error(`Failed to rasterize image: ${filename}`);
  }
  return new Uint8Array(await pngBlob.arrayBuffer());
}

async function decodeImageElementToPngBytes(file: File): Promise<Uint8Array> {
  ensureDomDocumentAvailable();

  const objectUrl = URL.createObjectURL(file);
  try {
    const image = await new Promise<HTMLImageElement>((resolve, reject) => {
      const img = new Image();
      img.onload = () => resolve(img);
      img.onerror = () => reject(new Error(`Failed to decode image: ${file.name}`));
      img.src = objectUrl;
    });

    const canvas = document.createElement("canvas");
    canvas.width = Math.max(1, image.naturalWidth || image.width);
    canvas.height = Math.max(1, image.naturalHeight || image.height);

    const ctx = canvas.getContext("2d");
    if (!ctx) {
      throw new Error("Unable to create a canvas 2D context");
    }

    // Draw decoded image into a canvas to flatten orientation/frames to a single raster.
    ctx.drawImage(image, 0, 0, canvas.width, canvas.height);

    return canvasToPngBytes(canvas, file.name);
  } finally {
    URL.revokeObjectURL(objectUrl);
  }
}

async function decodeTiffToPngBytes(file: File): Promise<Uint8Array> {
  ensureDomDocumentAvailable();

  const utifModule = await import("utif");
  const utif = (
    "default" in utifModule && utifModule.default
      ? utifModule.default
      : utifModule
  ) as typeof import("utif");
  const sourceBuffer = await toArrayBuffer(file);
  const ifds = utif.decode(sourceBuffer);
  if (!ifds || ifds.length === 0) {
    throw new Error(`Failed to decode TIFF: ${file.name}`);
  }

  const firstFrame = ifds[0] as Record<string, unknown>;
  utif.decodeImage(sourceBuffer, firstFrame);
  const rgba = utif.toRGBA8(firstFrame);

  const widthRaw = Number(firstFrame.width ?? firstFrame.t256 ?? 0);
  const heightRaw = Number(firstFrame.height ?? firstFrame.t257 ?? 0);
  const width = Math.max(1, Number.isFinite(widthRaw) ? widthRaw : 0);
  const height = Math.max(1, Number.isFinite(heightRaw) ? heightRaw : 0);
  if (!rgba || rgba.length !== width * height * 4) {
    throw new Error(`Invalid TIFF pixel data: ${file.name}`);
  }

  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;

  const ctx = canvas.getContext("2d");
  if (!ctx) {
    throw new Error("Unable to create a canvas 2D context");
  }

  const imageData = new ImageData(new Uint8ClampedArray(rgba), width, height);
  ctx.putImageData(imageData, 0, 0);

  return canvasToPngBytes(canvas, file.name);
}

async function imageFileToPngBytes(file: File): Promise<Uint8Array> {
  try {
    return await decodeImageElementToPngBytes(file);
  } catch (error) {
    if (!isTiffLikeFile(file)) {
      throw error;
    }
    return decodeTiffToPngBytes(file);
  }
}

function fitContentIntoA4Page(contentWidth: number, contentHeight: number): {
  x: number;
  y: number;
  width: number;
  height: number;
} {
  const sourceWidth = Math.max(1, contentWidth);
  const sourceHeight = Math.max(1, contentHeight);

  const availableWidth = A4_PAGE_WIDTH_POINTS - A4_PAGE_MARGIN_POINTS * 2;
  const availableHeight = A4_PAGE_HEIGHT_POINTS - A4_PAGE_MARGIN_POINTS * 2;
  const scale = Math.min(
    availableWidth / sourceWidth,
    availableHeight / sourceHeight,
  );

  const width = sourceWidth * scale;
  const height = sourceHeight * scale;
  const x = (A4_PAGE_WIDTH_POINTS - width) / 2;
  const y = (A4_PAGE_HEIGHT_POINTS - height) / 2;

  return { x, y, width, height };
}

async function appendPdfFile(
  mergedPdf: import("pdf-lib").PDFDocument,
  file: File,
): Promise<number> {
  const sourceBytes = await toArrayBuffer(file);
  const embeddedPages = await mergedPdf.embedPdf(sourceBytes);
  for (const embeddedPage of embeddedPages) {
    const page = mergedPdf.addPage([A4_PAGE_WIDTH_POINTS, A4_PAGE_HEIGHT_POINTS]);
    page.drawPage(
      embeddedPage,
      fitContentIntoA4Page(embeddedPage.width, embeddedPage.height),
    );
  }
  return embeddedPages.length;
}

async function appendImageFile(
  mergedPdf: import("pdf-lib").PDFDocument,
  file: File,
): Promise<void> {
  const pngBytes = await imageFileToPngBytes(file);
  const embeddedImage = await mergedPdf.embedPng(pngBytes);
  const page = mergedPdf.addPage([A4_PAGE_WIDTH_POINTS, A4_PAGE_HEIGHT_POINTS]);
  page.drawImage(
    embeddedImage,
    fitContentIntoA4Page(embeddedImage.width, embeddedImage.height),
  );
}

export async function mergePhotosAndPdfsToFlattenedPdf(
  files: File[],
): Promise<Uint8Array> {
  const { PDFDocument } = await import("pdf-lib");
  const mergedPdf = await PDFDocument.create();

  for (const file of files) {
    if (!isMergePrintableFile(file)) {
      throw new Error(`Unsupported file type: ${file.type || "unknown"}`);
    }
    if (String(file.type || "").toLowerCase() === PDF_MIME_TYPE) {
      const pageCount = await appendPdfFile(mergedPdf, file);
      if (pageCount === 0) {
        throw new Error(`PDF has no pages: ${file.name}`);
      }
      continue;
    }
    await appendImageFile(mergedPdf, file);
  }

  return mergedPdf.save();
}

export async function mergePhotosAndPdfsAndOpenPrintDialog(
  files: File[],
): Promise<boolean> {
  const mergedBytes = await mergePhotosAndPdfsToFlattenedPdf(files);
  const mergedBlob = new Blob([mergedBytes], { type: PDF_MIME_TYPE });
  const mergedUrl = URL.createObjectURL(mergedBlob);
  const opened = openPrintDialogForMedia(mergedUrl);
  if (!opened) {
    URL.revokeObjectURL(mergedUrl);
    return false;
  }
  window.setTimeout(() => URL.revokeObjectURL(mergedUrl), PRINT_URL_REVOKE_DELAY_MS);
  return true;
}

export const batchPrintOutputFilename = MERGED_PRINT_FILENAME;
