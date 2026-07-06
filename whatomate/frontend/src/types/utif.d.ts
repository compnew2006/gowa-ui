declare module "utif" {
  export interface TiffIFD {
    width?: number;
    height?: number;
    t256?: number;
    t257?: number;
    [key: string]: unknown;
  }

  export function decode(buffer: ArrayBuffer): TiffIFD[];
  export function decodeImage(buffer: ArrayBuffer, ifd: TiffIFD): void;
  export function toRGBA8(ifd: TiffIFD): Uint8Array;
}
