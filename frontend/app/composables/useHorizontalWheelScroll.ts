export function useHorizontalWheelScroll(elRef: Ref<HTMLElement | null>) {
  function onWheel(e: WheelEvent) {
    const el = elRef.value
    if (!el) return
    if (Math.abs(e.deltaY) <= Math.abs(e.deltaX)) return
    if (el.scrollWidth <= el.clientWidth) return

    e.preventDefault()
    el.scrollLeft += e.deltaY
  }

  watch(
    elRef,
    (el, _, onCleanup) => {
      if (!el) return
      el.addEventListener('wheel', onWheel, { passive: false })
      onCleanup(() => el.removeEventListener('wheel', onWheel))
    },
    { immediate: true },
  )
}
