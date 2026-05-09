import {
  ref,
  computed,
  watch,
  onUnmounted,
  nextTick,
  type Ref,
  type ComputedRef,
} from "vue";

export interface VirtualItem<T = any> {
  data: T;
  index: number;
}

interface Options<T extends { id: string }> {
  items: Ref<T[]> | ComputedRef<T[]>;
  getViewport: () => HTMLElement | null;
  estimatedHeight?: number;
  buffer?: number;
}

export function useVirtualMessageList<T extends { id: string }>({
  items,
  getViewport,
  estimatedHeight = 80,
  buffer = 15,
}: Options<T>) {
  const heights = new Map<string, number>();
  let scrollPos = 0;
  let viewH = 0;
  let listening = false;
  let raf = 0;
  const forceIdx = ref(-1);
  const ver = ref(0);

  const getH = (id: string) => heights.get(id) ?? estimatedHeight;

  function syncViewport() {
    const vp = getViewport();
    if (vp) {
      scrollPos = vp.scrollTop;
      viewH = vp.clientHeight;
      ver.value++;
    }
  }

  function onScroll() {
    if (raf) return;
    raf = requestAnimationFrame(() => {
      syncViewport();
      raf = 0;
    });
  }

  function computeOffsets() {
    const arr = items.value;
    const n = arr.length;
    const off = new Float64Array(n);
    let acc = 0;
    for (let i = 0; i < n; i++) {
      off[i] = acc;
      acc += getH(arr[i].id);
    }
    return off;
  }

  const visibleRange = computed(() => {
    ver.value;
    const arr = items.value;
    const n = arr.length;
    if (n === 0) return [0, -1] as const;

    const off = computeOffsets();
    const top = scrollPos;
    const bot = scrollPos + viewH;

    let lo = 0;
    let hi = n - 1;
    while (lo < hi) {
      const mid = (lo + hi) >>> 1;
      if (off[mid] + getH(arr[mid].id) <= top) lo = mid + 1;
      else hi = mid;
    }
    let start = Math.max(0, lo - buffer);

    lo = start;
    hi = n - 1;
    while (lo < hi) {
      const mid = (lo + hi) >>> 1;
      if (off[mid] < bot) lo = mid + 1;
      else hi = mid;
    }
    let end = Math.min(n - 1, lo + buffer);

    const fi = forceIdx.value;
    if (fi >= 0 && fi < n) {
      start = Math.min(start, fi);
      end = Math.max(end, fi);
    }

    return [start, end] as const;
  });

  const virtualItems = computed(() => {
    const [s, e] = visibleRange.value;
    const result: VirtualItem<T>[] = [];
    for (let i = s; i <= e; i++) {
      result.push({ data: items.value[i], index: i });
    }
    return result;
  });

  const topSpacer = computed(() => {
    const [s] = visibleRange.value;
    if (s <= 0) return 0;
    ver.value;
    const off = computeOffsets();
    return off[s];
  });

  const bottomSpacer = computed(() => {
    const [, e] = visibleRange.value;
    const n = items.value.length;
    if (e < 0 || e >= n - 1) return 0;
    ver.value;
    const off = computeOffsets();
    const total = off[n - 1] + getH(items.value[n - 1].id);
    return Math.max(0, total - off[e] - getH(items.value[e].id));
  });

  const totalHeight = computed(() => {
    ver.value;
    const arr = items.value;
    if (!arr.length) return 0;
    const off = computeOffsets();
    return off[arr.length - 1] + getH(arr[arr.length - 1].id);
  });

  function reportHeight(id: string, height: number) {
    if (height > 0) heights.set(id, height);
  }

  function setup() {
    const vp = getViewport();
    if (!vp || listening) return;
    vp.addEventListener("scroll", onScroll, { passive: true });
    listening = true;
    syncViewport();
  }

  function cleanup() {
    const vp = getViewport();
    if (vp && listening) {
      vp.removeEventListener("scroll", onScroll);
      listening = false;
    }
    if (raf) {
      cancelAnimationFrame(raf);
      raf = 0;
    }
  }

  function scrollToIndex(idx: number, behavior: ScrollBehavior = "smooth") {
    const vp = getViewport();
    if (!vp || idx < 0 || idx >= items.value.length) return;
    const off = computeOffsets();
    vp.scrollTo({ top: off[idx], behavior });
  }

  async function ensureVisible(index: number): Promise<void> {
    forceIdx.value = index;
    ver.value++;
    await nextTick();
    forceIdx.value = -1;
    ver.value++;
  }

  watch(items, (val) => {
    if (val.length === 0) heights.clear();
  });

  onUnmounted(cleanup);

  return {
    virtualItems,
    topSpacer,
    bottomSpacer,
    totalHeight,
    reportHeight,
    setup,
    cleanup,
    scrollToIndex,
    ensureVisible,
  };
}
