const DEMO_HOSTS = new Set(['checkout.local', 'localhost', '127.0.0.1'])

function demoCheckoutPath(ticketId: string): string {
  return `/checkout?ticket=${encodeURIComponent(ticketId)}`
}

/**
 * Куда вести после activate.
 * Мок Авито отдаёт https://checkout.local/... — это не наш SPA, мапим на /checkout.
 * Относительный путь → navigate; настоящий внешний URL → полный переход.
 */
export function resolveCheckoutNavigation(
  ticketId: string,
  checkoutUrl?: string | null,
): { kind: 'external'; url: string } | { kind: 'internal'; path: string } {
  const fallback = demoCheckoutPath(ticketId)
  const url = checkoutUrl?.trim()
  if (!url) {
    return { kind: 'internal', path: fallback }
  }

  if (/^https?:\/\//i.test(url)) {
    try {
      const parsed = new URL(url)
      const host = parsed.hostname.toLowerCase()
      const sameOrigin =
        typeof window !== 'undefined' &&
        host === window.location.hostname.toLowerCase()

      if (DEMO_HOSTS.has(host) || sameOrigin) {
        if (sameOrigin && parsed.pathname.startsWith('/checkout')) {
          const path = `${parsed.pathname}${parsed.search}`
          return {
            kind: 'internal',
            path: parsed.searchParams.has('ticket') ? path : fallback,
          }
        }
        return { kind: 'internal', path: fallback }
      }
    } catch {
      return { kind: 'internal', path: fallback }
    }
    return { kind: 'external', url }
  }

  const path = url.startsWith('/') ? url : `/${url}`
  if (path.startsWith('/checkout')) {
    try {
      const parsed = new URL(path, 'http://local.invalid')
      if (!parsed.searchParams.get('ticket')) {
        return { kind: 'internal', path: fallback }
      }
    } catch {
      return { kind: 'internal', path: fallback }
    }
  }
  return { kind: 'internal', path }
}
