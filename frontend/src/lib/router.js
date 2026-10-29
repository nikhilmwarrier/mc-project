// Tiny history-API router: no dependency, SPA-friendly.
export function navigate(path, replace = false) {
  if (replace) history.replaceState({}, '', path);
  else history.pushState({}, '', path);
  window.dispatchEvent(new PopStateEvent('popstate'));
}

export const pathNow = () => window.location.pathname;
