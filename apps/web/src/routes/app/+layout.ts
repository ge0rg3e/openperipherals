// Client-only rendering: every interaction (WebHID, device state, control
// signalling) requires the browser. Turning off SSR prevents any module from
// running in a Node context and lets all scripts load on the client.
export const ssr = false;
export const prerender = false;
export const csr = true;
