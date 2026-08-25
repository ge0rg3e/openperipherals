// Bridges window-control IPC to the renderer without exposing node/Electron.
const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('desktop', {
	// Desktop environment info is resolved in the main process; sandboxed
	// preloads cannot read process.env themselves.
	env: () => ipcRenderer.invoke('win:env'),
	minimize: () => ipcRenderer.send('win:minimize'),
	toggleMaximize: () => ipcRenderer.send('win:maximize'),
	close: () => ipcRenderer.send('win:close'),
	isMaximized: () => ipcRenderer.invoke('win:is-maximized'),
	onMaximized: (cb) => {
		ipcRenderer.on('win:maximized', (_event, value) => cb(value));
	}
});
