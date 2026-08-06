/**
 * Куда вести после activate.
 * Абсолютный checkout_url → полный переход; относительный → SPA navigate;
 * пусто → наш демо-checkout.
 */
export function resolveCheckoutNavigation(
  ticketId: string,
  checkoutUrl?: string | null,
): { kind: 'external'; url: string } | { kind: 'internal'; path: string } {
  const url = checkoutUrl?.trim()
  if (!url) {
    return { kind: 'internal', path: `/checkout?ticket=${ticketId}` }
  }
  if (/^https?:\/\//i.test(url)) {
    return { kind: 'external', url }
  }
  return { kind: 'internal', path: url.startsWith('/') ? url : `/${url}` }
}
