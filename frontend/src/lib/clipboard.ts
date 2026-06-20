// Copy text to the clipboard in a way that also works over plain HTTP.
//
// navigator.clipboard only exists in a "secure context" (HTTPS or localhost).
// When the panel is served over plain HTTP on a remote IP (e.g.
// http://103.26.176.213:5173) navigator.clipboard is undefined and calling it
// throws "navigator.clipboard is undefined". Fall back to the legacy
// execCommand("copy") path via a hidden textarea, which works in insecure
// contexts. Returns true on success so callers can show a "copied" state or a
// manual-copy hint on failure.
export async function copyToClipboard(text: string): Promise<boolean> {
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // fall through to the legacy path
    }
  }

  try {
    const textarea = document.createElement("textarea")
    textarea.value = text
    // Keep it out of view and out of the layout/scroll flow.
    textarea.style.position = "fixed"
    textarea.style.top = "-9999px"
    textarea.style.left = "-9999px"
    textarea.setAttribute("readonly", "")
    document.body.appendChild(textarea)
    textarea.select()
    const ok = document.execCommand("copy")
    document.body.removeChild(textarea)
    return ok
  } catch {
    return false
  }
}
