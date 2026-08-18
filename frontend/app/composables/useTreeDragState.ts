import type { SortableEvent } from 'sortablejs'

export interface TreeDragItemRef {
  id: string
  type: 'folder' | 'request'
}

const treeDragState = reactive({
  active: false,
  persisting: false,
  item: null as TreeDragItemRef | null,
  hoverFolderId: null as string | null,
})

let hoverRaf: number | null = null
let pendingMove: SortableEvent | null = null
let stuckTimer: ReturnType<typeof setTimeout> | null = null

function isLocked() {
  return treeDragState.active || treeDragState.persisting
}

function clearHoverFolder() {
  treeDragState.hoverFolderId = null
}

function setHoverFolder(folderId: string | null) {
  treeDragState.hoverFolderId = folderId
}

function applyHoverFromMove(evt: SortableEvent) {
  const related = evt.related as HTMLElement | null
  if (!related) {
    setHoverFolder(null)
    return
  }
  const folderHeader = related.closest('[data-drop-folder-id]') as HTMLElement | null
  if (folderHeader) {
    setHoverFolder(folderHeader.getAttribute('data-drop-folder-id'))
    return
  }
  const container = related.closest('[data-collection-id]') as HTMLElement | null
  if (container) {
    setHoverFolder(container.getAttribute('data-collection-id') || null)
    return
  }
  setHoverFolder(null)
}

function cancelHoverRaf() {
  pendingMove = null
  if (hoverRaf != null && import.meta.client) {
    cancelAnimationFrame(hoverRaf)
    hoverRaf = null
  }
}

/* If a drag never reaches @end (cancelled drag, thrown error), the lock would
   freeze the tree forever. Watch for the native drag ending and force-reset. */
function onNativeDragFinished() {
  if (stuckTimer) clearTimeout(stuckTimer)
  stuckTimer = setTimeout(() => {
    stuckTimer = null
    if (treeDragState.active && !treeDragState.persisting) {
      resetDragState()
    }
  }, 600)
}

function attachFailSafe() {
  if (!import.meta.client) return
  document.addEventListener('dragend', onNativeDragFinished)
  document.addEventListener('drop', onNativeDragFinished)
  document.addEventListener('pointerup', onNativeDragFinished)
}

function detachFailSafe() {
  if (!import.meta.client) return
  document.removeEventListener('dragend', onNativeDragFinished)
  document.removeEventListener('drop', onNativeDragFinished)
  document.removeEventListener('pointerup', onNativeDragFinished)
  if (stuckTimer) {
    clearTimeout(stuckTimer)
    stuckTimer = null
  }
}

function resetDragState() {
  treeDragState.active = false
  treeDragState.persisting = false
  treeDragState.item = null
  clearHoverFolder()
  cancelHoverRaf()
  detachFailSafe()
  if (import.meta.client) {
    document.body.classList.remove('tree-dragging')
  }
}

export function useTreeDragState() {
  function startDrag(item: TreeDragItemRef) {
    if (treeDragState.active) return
    treeDragState.active = true
    treeDragState.item = item
    if (import.meta.client) {
      document.body.classList.add('tree-dragging')
    }
    attachFailSafe()
  }

  /** Lock the tree while the drop is being saved; hover state is no longer needed. */
  function beginPersist() {
    treeDragState.persisting = true
    treeDragState.active = false
    treeDragState.item = null
    clearHoverFolder()
    cancelHoverRaf()
    detachFailSafe()
    if (import.meta.client) {
      document.body.classList.remove('tree-dragging')
    }
  }

  function endDrag() {
    resetDragState()
  }

  /** Throttled to one DOM read per animation frame. */
  function detectHoverFromMove(evt: SortableEvent) {
    if (!treeDragState.active || !import.meta.client) return
    pendingMove = evt
    if (hoverRaf != null) return
    hoverRaf = requestAnimationFrame(() => {
      hoverRaf = null
      if (!pendingMove || !treeDragState.active) return
      applyHoverFromMove(pendingMove)
      pendingMove = null
    })
  }

  return {
    state: treeDragState,
    isLocked,
    startDrag,
    beginPersist,
    endDrag,
    clearHoverFolder,
    setHoverFolder,
    detectHoverFromMove,
  }
}
