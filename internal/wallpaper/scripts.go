package wallpaper

import "fmt"

// Installed before page scripts, in every frame. One native rAF services the
// callback queue; using one timer per callback would increase wakeups and break
// cancellation. CSS/video/worker animation is intentionally not advertised as capped.
func animationScript(fps int) string {
	return fmt.Sprintf(`(() => {
  const request = window.requestAnimationFrame.bind(window);
  const cancel = window.cancelAnimationFrame.bind(window);
  const step = 1000 / %d;
  let callbacks = new Map(), active = null, token = 0, next = 0, last = -Infinity;
  function frame(now) {
    token = 0;
    if (now - last >= step - .5) {
      last = now;
      const batch = callbacks;
      active = batch;
      callbacks = new Map();
      for (const [id, fn] of batch) {
        if (!batch.has(id)) continue;
        try { fn(now); } catch (error) { setTimeout(() => { throw error; }); }
      }
      active = null;
    }
    if (callbacks.size && !token) token = request(frame);
  }
  window.requestAnimationFrame = fn => {
    if (typeof fn !== 'function') throw new TypeError('Callback must be a function');
    const id = ++next;
    callbacks.set(id, fn);
    if (!token) token = request(frame);
    return id;
  };
  window.cancelAnimationFrame = id => {
    callbacks.delete(id);
    active?.delete(id);
    if (!callbacks.size && token) { cancel(token); token = 0; }
  };
  // No page-to-host bridge, popup windows, dialogs or audible media.
  window.open = () => null;
  document.addEventListener('DOMContentLoaded', () => {
    document.documentElement.style.overflow = 'hidden';
    document.querySelectorAll('audio,video').forEach(media => media.muted = true);
  }, {once:true});
})();`, fps)
}

const builtinHTML = `<!doctype html><html><head><meta charset="utf-8"><style>
html,body{margin:0;width:100%;height:100%;overflow:hidden;background:#080d19}canvas{width:100%;height:100%;display:block}
</style></head><body><canvas id="scene"></canvas><script>
const canvas=document.getElementById('scene'),ctx=canvas.getContext('2d',{alpha:false});
function resize(){canvas.width=Math.min(innerWidth,1600);canvas.height=Math.round(canvas.width*innerHeight/innerWidth)}
addEventListener('resize',resize);resize();
function draw(now){const w=canvas.width,h=canvas.height,t=now/1000;ctx.fillStyle='#080d19';ctx.fillRect(0,0,w,h);
for(let i=0;i<3;i++){const x=w*(.2+i*.3)+Math.sin(t*.08+i*2)*w*.14,y=h*(.42+Math.cos(t*.06+i)*.22),r=w*.52;
const g=ctx.createRadialGradient(x,y,0,x,y,r);g.addColorStop(0,['#154c5760','#203e7060','#20544360'][i]);g.addColorStop(1,'#080d1900');ctx.fillStyle=g;ctx.fillRect(0,0,w,h)}
requestAnimationFrame(draw)}requestAnimationFrame(draw);
</script></body></html>`
