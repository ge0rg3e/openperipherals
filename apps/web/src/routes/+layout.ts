// SEO: the landing page is statically prerendered so search engines and link
// previews get real HTML. Only the interactive /app route stays client-only
// (see src/routes/app/+layout.ts) because it talks to hardware via WebHID.
export const ssr = true;
export const prerender = true;
export const csr = true;
