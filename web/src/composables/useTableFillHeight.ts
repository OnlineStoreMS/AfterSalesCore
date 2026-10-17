import { nextTick, onMounted, onUnmounted, ref, type Ref } from 'vue'

/**
 * 让 el-table 占满页面剩余高度，横/竖滚动条落在表格可视区内，
 * 避免整页拉到最底下才出现横向滚动条，也避免 fixed 列把表体高度撑出空白。
 */
export function useTableFillHeight(
  pageRef: Ref<HTMLElement | null>,
  reserveRefs: Ref<HTMLElement | null>[],
  options?: { min?: number; gap?: number },
) {
  const tableHeight = ref(Math.max(options?.min ?? 320, window.innerHeight - 280))
  const min = options?.min ?? 320
  const gap = options?.gap ?? 16

  function update() {
    const page = pageRef.value
    if (!page) return
    const pageH = page.clientHeight
    let reserved = 0
    for (const r of reserveRefs) {
      reserved += r.value?.offsetHeight ?? 0
    }
    tableHeight.value = Math.max(min, pageH - reserved - gap)
  }

  let ro: ResizeObserver | null = null

  onMounted(async () => {
    await nextTick()
    update()
    window.addEventListener('resize', update)
    if (typeof ResizeObserver !== 'undefined') {
      ro = new ResizeObserver(() => update())
      if (pageRef.value) ro.observe(pageRef.value)
      for (const r of reserveRefs) {
        if (r.value) ro.observe(r.value)
      }
    }
  })

  onUnmounted(() => {
    window.removeEventListener('resize', update)
    ro?.disconnect()
  })

  return { tableHeight, updateTableHeight: update }
}
