const { app, BrowserWindow, dialog, Tray, Menu, shell } = require('electron');
const { spawn } = require('child_process');
const path = require('path');
const http = require('http');
const fs = require('fs');

let mainWindow;
let serverProcess;
let tray = null;
let serverErrorLogs = [];
const PORT = 3000;
const DEV_FRONTEND_PORT = 5173; // Vite dev server port

// Save the error log to a file and open it
function saveAndOpenErrorLog() {
  try {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const logFileName = `new-api-crash-${timestamp}.log`;
    const logDir = app.getPath('logs');
    const logFilePath = path.join(logDir, logFileName);
    
    // Ensure the log directory exists
    if (!fs.existsSync(logDir)) {
      fs.mkdirSync(logDir, { recursive: true });
    }
    
    // Write the log file
    const logContent = `TrustHarbor crash log
Generated at: ${new Date().toLocaleString('en-US')}
Platform: ${process.platform}
Architecture: ${process.arch}
App version: ${app.getVersion()}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Full error log:

${serverErrorLogs.join('\n')}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Log file location: ${logFilePath}
`;
    
    fs.writeFileSync(logFilePath, logContent, 'utf8');
    
    // Open the log file
    shell.openPath(logFilePath).then((error) => {
      if (error) {
        console.error('Failed to open log file:', error);
        // If opening the file fails, at least reveal its location
        shell.showItemInFolder(logFilePath);
      }
    });
    
    return logFilePath;
  } catch (err) {
    console.error('Failed to save error log:', err);
    return null;
  }
}

// Analyze the error log, identify common failures, and provide solutions
function analyzeError(errorLogs) {
  const allLogs = errorLogs.join('\n');
  
  // Detect port-in-use errors
  if (allLogs.includes('failed to start HTTP server') || 
      allLogs.includes('bind: address already in use') ||
      allLogs.includes('listen tcp') && allLogs.includes('bind: address already in use')) {
    return {
      type: 'Port in use',
      title: 'Port ' + PORT + ' is already in use',
      message: 'Unable to start the server because the port is occupied by another program',
      solution: `Possible solutions:\n\n1. Close the other program using port ${PORT}\n2. Check whether another TrustHarbor instance is already running\n3. Find the process using the port with:\n   Mac/Linux: lsof -i :${PORT}\n   Windows: netstat -ano | findstr :${PORT}\n4. Restart your computer to release the port`
    };
  }
  
  // Detect database errors
  if (allLogs.includes('database is locked') || 
      allLogs.includes('unable to open database')) {
    return {
      type: 'Data file in use',
      title: 'Unable to access the data file',
      message: 'The application data file is being used by another program',
      solution: 'Possible solutions:\n\n1. Check whether another TrustHarbor window is already open\n   - Look for another TrustHarbor icon in the taskbar/Dock\n   - Look for a TrustHarbor icon in the system tray (Windows) or menu bar (Mac)\n\n2. If you just closed the app, wait 10 seconds and try again\n\n3. Restart your computer to release the locked file\n\n4. If the problem persists, try:\n   - Quit all TrustHarbor instances\n   - Delete the temporary files (.db-shm and .db-wal) in the data directory\n   - Restart the app'
    };
  }
  
  // Detect permission errors
  if (allLogs.includes('permission denied') || 
      allLogs.includes('access denied')) {
    return {
      type: 'Permission error',
      title: 'Insufficient permissions',
      message: 'The program does not have enough permissions to perform the operation',
      solution: 'Possible solutions:\n\n1. Run the program as administrator/root\n2. Check read/write permissions on the data directory\n3. Check the permissions of the executable\n4. On Mac, check the Security & Privacy settings'
    };
  }
  
  // Detect network errors
  if (allLogs.includes('network is unreachable') || 
      allLogs.includes('no such host') ||
      allLogs.includes('connection refused')) {
    return {
      type: 'Network error',
      title: 'Network connection failed',
      message: 'Unable to establish a network connection',
      solution: 'Possible solutions:\n\n1. Check whether the network connection is working\n2. Check the firewall settings\n3. Check the proxy configuration\n4. Confirm the target server address is correct'
    };
  }
  
  // Detect configuration errors
  if (allLogs.includes('invalid configuration') || 
      allLogs.includes('failed to parse config') ||
      allLogs.includes('yaml') || allLogs.includes('json') && allLogs.includes('parse')) {
    return {
      type: 'Configuration error',
      title: 'Configuration file error',
      message: 'The configuration file is malformed or contains invalid settings',
      solution: 'Possible solutions:\n\n1. Check whether the configuration file format is correct\n2. Restore the default configuration\n3. Delete the configuration file so the program regenerates it\n4. Consult the documentation for the correct configuration format'
    };
  }
  
  // Detect out-of-memory errors
  if (allLogs.includes('out of memory') || 
      allLogs.includes('cannot allocate memory')) {
    return {
      type: 'Out of memory',
      title: 'System memory exhausted',
      message: 'The program ran out of memory',
      solution: 'Possible solutions:\n\n1. Close other memory-intensive programs\n2. Increase available system memory\n3. Restart your computer to free memory\n4. Check for memory leaks'
    };
  }
  
  // Detect missing-file errors
  if (allLogs.includes('no such file or directory') || 
      allLogs.includes('cannot find the file')) {
    return {
      type: 'Missing file',
      title: 'Required file not found',
      message: 'A file required to run the program is missing',
      solution: 'Possible solutions:\n\n1. Reinstall the application\n2. Check whether the installation directory is complete\n3. Make sure all dependency files exist\n4. Check whether the file path is correct'
    };
  }
  
  return null;
}

function getBinaryPath() {
  const isDev = process.env.NODE_ENV === 'development';
  const platform = process.platform;

  if (isDev) {
    const binaryName = platform === 'win32' ? 'new-api.exe' : 'new-api';
    return path.join(__dirname, '..', binaryName);
  }

  let binaryName;
  switch (platform) {
    case 'win32':
      binaryName = 'new-api.exe';
      break;
    case 'darwin':
      binaryName = 'new-api';
      break;
    case 'linux':
      binaryName = 'new-api';
      break;
    default:
      binaryName = 'new-api';
  }

  return path.join(process.resourcesPath, 'bin', binaryName);
}

// Check if a server is available with retry logic
function checkServerAvailability(port, maxRetries = 30, retryDelay = 1000) {
  return new Promise((resolve, reject) => {
    let currentAttempt = 0;
    
    const tryConnect = () => {
      currentAttempt++;
      
      if (currentAttempt % 5 === 1 && currentAttempt > 1) {
        console.log(`Attempting to connect to port ${port}... (attempt ${currentAttempt}/${maxRetries})`);
      }
      
      const req = http.get({
        hostname: '127.0.0.1', // Use IPv4 explicitly instead of 'localhost' to avoid IPv6 issues
        port: port,
        timeout: 10000
      }, (res) => {
        // Server responded, connection successful
        req.destroy();
        console.log(`✓ Successfully connected to port ${port} (status: ${res.statusCode})`);
        resolve();
      });

      req.on('error', (err) => {
        if (currentAttempt >= maxRetries) {
          reject(new Error(`Failed to connect to port ${port} after ${maxRetries} attempts: ${err.message}`));
        } else {
          setTimeout(tryConnect, retryDelay);
        }
      });

      req.on('timeout', () => {
        req.destroy();
        if (currentAttempt >= maxRetries) {
          reject(new Error(`Connection timeout on port ${port} after ${maxRetries} attempts`));
        } else {
          setTimeout(tryConnect, retryDelay);
        }
      });
    };
    
    tryConnect();
  });
}

function startServer() {
  return new Promise((resolve, reject) => {
    const isDev = process.env.NODE_ENV === 'development';

    const userDataPath = app.getPath('userData');
    const dataDir = path.join(userDataPath, 'data');
    
    // Expose the data directory to preload.js through an environment variable
    process.env.ELECTRON_DATA_DIR = dataDir;
    
    if (isDev) {
      // Development mode: assume the developer started the Go backend and the
      // frontend dev server manually; only wait for the frontend dev server.
      console.log('Development mode: skipping server startup');
      console.log('Please make sure you have started:');
      console.log('  1. Go backend: go run main.go (port 3000)');
      console.log('  2. Frontend dev server: cd web && bun dev (port 5173)');
      console.log('');
      console.log('Checking if servers are running...');
      
      // First check if both servers are accessible
      checkServerAvailability(DEV_FRONTEND_PORT)
        .then(() => {
          console.log('✓ Frontend dev server is accessible on port 5173');
          resolve();
        })
        .catch((err) => {
          console.error(`✗ Cannot connect to frontend dev server on port ${DEV_FRONTEND_PORT}`);
          console.error('Please make sure the frontend dev server is running:');
          console.error('  cd web && bun dev');
          reject(err);
        });
      return;
    }

    // Production mode: start the bundled binary server
    const env = { ...process.env, PORT: PORT.toString() };

    if (!fs.existsSync(dataDir)) {
      fs.mkdirSync(dataDir, { recursive: true });
    }

    env.SQLITE_PATH = path.join(dataDir, 'new-api.db');
    
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📁 Your data is stored at:');
    console.log('   ' + dataDir);
    console.log('   💡 Backup tip: copy this directory to back up all data');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

    const binaryPath = getBinaryPath();
    const workingDir = process.resourcesPath;
    
    console.log('Starting server from:', binaryPath);

    serverProcess = spawn(binaryPath, [], {
      env,
      cwd: workingDir
    });

    serverProcess.stdout.on('data', (data) => {
      console.log(`Server: ${data}`);
    });

    serverProcess.stderr.on('data', (data) => {
      const errorMsg = data.toString();
      console.error(`Server Error: ${errorMsg}`);
      serverErrorLogs.push(errorMsg);
      // Keep only the most recent 100 error log entries
      if (serverErrorLogs.length > 100) {
        serverErrorLogs.shift();
      }
    });

    serverProcess.on('error', (err) => {
      console.error('Failed to start server:', err);
      reject(err);
    });

    serverProcess.on('close', (code) => {
      console.log(`Server process exited with code ${code}`);
      
      // A non-zero exit code means the server crashed
      if (code !== 0 && code !== null) {
        const errorDetails = serverErrorLogs.length > 0 
          ? serverErrorLogs.slice(-20).join('\n') 
          : 'No error logs were captured';
        
        // Analyze the error type
        const knownError = analyzeError(serverErrorLogs);
        
        let dialogOptions;
        if (knownError) {
          // Known error: show a friendly message with solutions
          dialogOptions = {
            type: 'error',
            title: knownError.title,
            message: knownError.message,
            detail: `${knownError.solution}\n\n━━━━━━━━━━━━━━━━━━━━━━\n\nExit code: ${code}\n\nError type: ${knownError.type}\n\nRecent error logs:\n${errorDetails}`,
            buttons: ['Quit', 'View full log'],
            defaultId: 0,
            cancelId: 0
          };
        } else {
          // Unknown error: show a generic error message
          dialogOptions = {
            type: 'error',
            title: 'Server crashed',
            message: 'The server process exited unexpectedly',
            detail: `Exit code: ${code}\n\nRecent error output:\n${errorDetails}`,
            buttons: ['Quit', 'View full log'],
            defaultId: 0,
            cancelId: 0
          };
        }
        
        dialog.showMessageBox(dialogOptions).then((result) => {
          if (result.response === 1) {
            // The user chose to view details: save and open the log file
            const logPath = saveAndOpenErrorLog();
            
            // Show a confirmation dialog
            const confirmMessage = logPath 
              ? `Log saved to:\n${logPath}\n\nThe log file has been opened in your default text editor.\n\nClick "Quit" to close the application.`
              : 'Failed to save the log, but it has been printed to the console.\n\nClick "Quit" to close the application.';
            
            dialog.showMessageBox({
              type: 'info',
              title: 'Log saved',
              message: confirmMessage,
              buttons: ['Quit'],
              defaultId: 0
            }).then(() => {
              app.isQuitting = true;
              app.quit();
            });
            
            // Also print to the console
            console.log('=== Full error log ===');
            console.log(serverErrorLogs.join('\n'));
          } else {
            // The user chose to quit immediately
            app.isQuitting = true;
            app.quit();
          }
        });
      } else {
        // Normal exit (code is 0 or null): close the window directly
        if (mainWindow && !mainWindow.isDestroyed()) {
          mainWindow.close();
        }
      }
    });

    checkServerAvailability(PORT)
      .then(() => {
        console.log('✓ Backend server is accessible on port 3000');
        resolve();
      })
      .catch((err) => {
        console.error('✗ Failed to connect to backend server');
        reject(err);
      });
  });
}

function createWindow() {
  const isDev = process.env.NODE_ENV === 'development';
  const loadPort = isDev ? DEV_FRONTEND_PORT : PORT;
  
  mainWindow = new BrowserWindow({
    width: 1080,
    height: 720,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: false,
      contextIsolation: true
    },
    title: 'New API',
    icon: path.join(__dirname, 'icon.png')
  });

  mainWindow.loadURL(`http://127.0.0.1:${loadPort}`);
  
  console.log(`Loading from: http://127.0.0.1:${loadPort}`);

  if (isDev) {
    mainWindow.webContents.openDevTools();
  }

  // Close to tray instead of quitting
  mainWindow.on('close', (event) => {
    if (!app.isQuitting) {
      event.preventDefault();
      mainWindow.hide();
      if (process.platform === 'darwin') {
        app.dock.hide();
      }
    }
  });

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
}

function createTray() {
  // Use template icon for macOS (black with transparency, auto-adapts to theme)
  // Use colored icon for Windows
  const trayIconPath = process.platform === 'darwin'
    ? path.join(__dirname, 'tray-iconTemplate.png')
    : path.join(__dirname, 'tray-icon-windows.png');

  tray = new Tray(trayIconPath);

  const contextMenu = Menu.buildFromTemplate([
    {
      label: 'Show New API',
      click: () => {
        if (mainWindow === null) {
          createWindow();
        } else {
          mainWindow.show();
          if (process.platform === 'darwin') {
            app.dock.show();
          }
        }
      }
    },
    { type: 'separator' },
    {
      label: 'Quit',
      click: () => {
        app.isQuitting = true;
        app.quit();
      }
    }
  ]);

  tray.setToolTip('New API');
  tray.setContextMenu(contextMenu);

  // On macOS, clicking the tray icon shows the window
  tray.on('click', () => {
    if (mainWindow === null) {
      createWindow();
    } else {
      mainWindow.isVisible() ? mainWindow.hide() : mainWindow.show();
      if (mainWindow.isVisible() && process.platform === 'darwin') {
        app.dock.show();
      }
    }
  });
}

app.whenReady().then(async () => {
  try {
    await startServer();
    createTray();
    createWindow();
  } catch (err) {
    console.error('Failed to start application:', err);
    
    // Analyze the startup failure
    const knownError = analyzeError(serverErrorLogs);
    
    if (knownError) {
      dialog.showMessageBox({
        type: 'error',
        title: knownError.title,
        message: `Startup failed: ${knownError.message}`,
        detail: `${knownError.solution}\n\n━━━━━━━━━━━━━━━━━━━━━━\n\nError: ${err.message}\n\nError type: ${knownError.type}`,
        buttons: ['Quit', 'View full log'],
        defaultId: 0,
        cancelId: 0
      }).then((result) => {
        if (result.response === 1) {
          // The user chose to view the log
          const logPath = saveAndOpenErrorLog();
          
          const confirmMessage = logPath 
            ? `Log saved to:\n${logPath}\n\nThe log file has been opened in your default text editor.\n\nClick "Quit" to close the application.`
            : 'Failed to save the log, but it has been printed to the console.\n\nClick "Quit" to close the application.';
          
          dialog.showMessageBox({
            type: 'info',
            title: 'Log saved',
            message: confirmMessage,
            buttons: ['Quit'],
            defaultId: 0
          }).then(() => {
            app.quit();
          });
          
          console.log('=== Full error log ===');
          console.log(serverErrorLogs.join('\n'));
        } else {
          app.quit();
        }
      });
    } else {
      dialog.showMessageBox({
        type: 'error',
        title: 'Startup failed',
        message: 'Unable to start the server',
        detail: `Error: ${err.message}\n\nPlease check the logs for more information.`,
        buttons: ['Quit', 'View full log'],
        defaultId: 0,
        cancelId: 0
      }).then((result) => {
        if (result.response === 1) {
          // The user chose to view the log
          const logPath = saveAndOpenErrorLog();
          
          const confirmMessage = logPath 
            ? `Log saved to:\n${logPath}\n\nThe log file has been opened in your default text editor.\n\nClick "Quit" to close the application.`
            : 'Failed to save the log, but it has been printed to the console.\n\nClick "Quit" to close the application.';
          
          dialog.showMessageBox({
            type: 'info',
            title: 'Log saved',
            message: confirmMessage,
            buttons: ['Quit'],
            defaultId: 0
          }).then(() => {
            app.quit();
          });
          
          console.log('=== Full error log ===');
          console.log(serverErrorLogs.join('\n'));
        } else {
          app.quit();
        }
      });
    }
  }
});

app.on('window-all-closed', () => {
  // Don't quit when window is closed, keep running in tray
  // Only quit when explicitly choosing Quit from tray menu
});

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) {
    createWindow();
  }
});

app.on('before-quit', (event) => {
  if (serverProcess) {
    event.preventDefault();

    console.log('Shutting down server...');
    serverProcess.kill('SIGTERM');

    setTimeout(() => {
      if (serverProcess) {
        serverProcess.kill('SIGKILL');
      }
      app.exit();
    }, 5000);

    serverProcess.on('close', () => {
      serverProcess = null;
      app.exit();
    });
  }
});