package wallpaper

import (
	"net"
	"net/http"
	"os"
	"time"
)

// Serve only this video and our fixed player on an ephemeral loopback port.
// ServeContent supports byte ranges and seeks without buffering the whole MP4.
// This uses WebView2's normal HTTP media pipeline and exposes no directory tree.
func serveVideo(path string) (*http.Server, string, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	server := &http.Server{Handler: videoHandler(path), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() { _ = server.Serve(listener) }()
	return server, "http://" + listener.Addr().String() + "/", nil
}

func videoHandler(path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(videoHTML()))
		case "/video":
			file, err := os.Open(path)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer file.Close()
			info, err := file.Stat()
			if err != nil || !info.Mode().IsRegular() {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "video/mp4")
			http.ServeContent(w, r, "wallpaper.mp4", info.ModTime(), file)
		default:
			http.NotFound(w, r)
		}
	})
}

func videoHTML() string {
	return `<!doctype html><html><head><meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; media-src 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'">
<style>html,body{margin:0;width:100%;height:100%;overflow:hidden;background:#080d19}video{width:100%;height:100%;object-fit:cover;display:block}</style>
</head><body><video id="wallpaper" muted loop playsinline preload="auto" disablepictureinpicture disableremoteplayback></video>
<script>
const video = document.getElementById('wallpaper');
const report = message => window.chrome.webview.postMessage(message);
video.muted = true;
video.addEventListener('loadeddata', () => report('video-ready'), {once:true});
video.addEventListener('playing', () => report('video-playing'));
video.addEventListener('error', () => report('video-error'));
window.wallpaperPlayback = playing => {
  if (playing) video.play().catch(error => { if (error.name !== 'AbortError') report('video-error'); });
  else video.pause();
};
document.addEventListener('visibilitychange', () => { if (document.hidden) video.pause(); });
video.src = '/video';
</script></body></html>`
}
