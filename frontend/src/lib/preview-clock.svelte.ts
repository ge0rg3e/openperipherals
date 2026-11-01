// One low-rate clock for all visible effect thumbnails.
export const previewClock = $state({time: 1.4});
let subscribers = 0;
let frame = 0;
let last = 0;
function tick(now: number) {
  if (now - last >= 50 && document.visibilityState === "visible") {
    previewClock.time = now / 1000;
    last = now;
  }
  frame = requestAnimationFrame(tick);
}
export function animatePreviews() {
  if (++subscribers === 1) frame = requestAnimationFrame(tick);
  return () => {
    if (--subscribers === 0) {
      cancelAnimationFrame(frame);
      frame = 0;
    }
  };
}
