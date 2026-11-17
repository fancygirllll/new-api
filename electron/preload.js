const { contextBridge } = require('electron');

// Resolve the data directory path (shown to the user).
// Prefer the real path set by the main process; fall back to manual joining otherwise.
function getDataDirPath() {
  // If the main process has set the real path, use it directly
  if (process.env.ELECTRON_DATA_DIR) {
    return process.env.ELECTRON_DATA_DIR;
  }
}

contextBridge.exposeInMainWorld('electron', {
  isElectron: true,
  version: process.versions.electron,
  platform: process.platform,
  versions: process.versions,
  dataDir: getDataDirPath()
});