export async function registerGeneric(host: string): Promise<void> {
  try {
    await chrome.scripting.registerContentScripts([
      { id: `wt-generic-${host}`, matches: [`*://${host}/*`], js: ["dist/content.js"], runAt: "document_idle" },
    ]);
  } catch {
    /* already registered */
  }
}

export async function unregisterGeneric(host: string): Promise<void> {
  try {
    await chrome.scripting.unregisterContentScripts({ ids: [`wt-generic-${host}`] });
  } catch {
    /* not registered */
  }
}
