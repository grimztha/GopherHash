import './style.css';

import {
    HashFile,
    HashAll,
    CompareFiles,
    VerifySingleHash,
    VerifyHashFile,
    VerifyChecksumText,
    OpenFile,
    SelectChecksumFile,
    ReadFileContent,
    WriteTempFile,
    GetVersion
} from '../wailsjs/go/main/App';
import {
    ClipboardSetText,
    OnFileDrop,
    WindowSetLightTheme,
    WindowSetDarkTheme
} from '../wailsjs/runtime';

// Initialize version badge from backend
if (typeof GetVersion === 'function') {
    GetVersion().then((ver) => {
        const verEl = document.getElementById('app-version');
        if (verEl && ver) {
            verEl.textContent = `v${ver}`;
        }
    }).catch(() => { });
}

// ============================================================================
// Theme Management (Dark / Light)
// ============================================================================
const themeToggleBtn = document.getElementById('theme-toggle');
const themeLabel = document.getElementById('theme-label');

function applyTheme(theme, save = true) {
    document.documentElement.setAttribute('data-theme', theme);
    if (themeLabel) {
        themeLabel.textContent = theme === 'light' ? 'Light' : 'Dark';
    }
    if (themeToggleBtn) {
        themeToggleBtn.setAttribute('title', `Switch to ${theme === 'light' ? 'Dark' : 'Light'} Mode`);
    }

    try {
        if (theme === 'light' && typeof WindowSetLightTheme === 'function') {
            WindowSetLightTheme();
        } else if (theme === 'dark' && typeof WindowSetDarkTheme === 'function') {
            WindowSetDarkTheme();
        }
    } catch (err) {
        console.debug("Native window theme not supported:", err);
    }

    if (save) {
        try {
            localStorage.setItem('gopherhash_theme', theme);
        } catch (e) { }
    }
}

let currentAppTheme = 'dark';
try {
    const saved = localStorage.getItem('gopherhash_theme');
    if (saved === 'light' || saved === 'dark') {
        currentAppTheme = saved;
    } else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
        currentAppTheme = 'light';
    }
} catch (e) { }

applyTheme(currentAppTheme, false);

if (themeToggleBtn) {
    themeToggleBtn.addEventListener('click', () => {
        currentAppTheme = currentAppTheme === 'dark' ? 'light' : 'dark';
        applyTheme(currentAppTheme, true);
    });
}

function setAppStatus(text, isBusy = false) {
    // Retained for compatibility with calculation status triggers
}

// ============================================================================
// Toast Notification Utility
// ============================================================================
function showToast(title, message = "", type = "info") {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;

    let iconSvg = '';
    if (type === 'success') {
        iconSvg = `<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"></polyline></svg>`;
    } else if (type === 'error') {
        iconSvg = `<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>`;
    } else {
        iconSvg = `<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>`;
    }

    toast.innerHTML = `
        <div class="toast-icon">${iconSvg}</div>
        <div class="toast-content">
            <div class="toast-title">${escapeHtml(title)}</div>
            ${message ? `<div class="toast-message">${escapeHtml(message)}</div>` : ''}
        </div>
    `;

    const removeToast = () => {
        if (toast.classList.contains('toast-hiding')) return;
        toast.classList.add('toast-hiding');
        setTimeout(() => {
            if (toast.parentNode) toast.parentNode.removeChild(toast);
        }, 260);
    };

    toast.addEventListener('click', removeToast);
    container.appendChild(toast);

    setTimeout(removeToast, 3800);
}

// ============================================================================
// Helpers
// ============================================================================
function escapeHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

function getBaseName(path) {
    if (!path) return '';
    return path.replace(/\\/g, '/').split('/').pop() || path;
}

function formatBytes(bytes) {
    if (!bytes || bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function formatSeconds(seconds) {
    if (typeof seconds !== 'number' || isNaN(seconds)) return '0.000s';
    if (seconds > 0 && seconds < 0.001) return '< 0.001s';
    return `${seconds.toFixed(3)}s`;
}

async function copyToClipboard(text, label = "Hash") {
    if (!text) return;
    try {
        await ClipboardSetText(text);
        showToast("Copied to Clipboard", `${label} copied successfully.`, "success");
    } catch (e) {
        if (navigator.clipboard && navigator.clipboard.writeText) {
            try {
                await navigator.clipboard.writeText(text);
                showToast("Copied to Clipboard", `${label} copied successfully.`, "success");
                return;
            } catch (err) { }
        }
        showToast("Copy Failed", "Clipboard operation not permitted.", "error");
    }
}

// ============================================================================
// Drag & Drop Setup
// ============================================================================
const registeredDropZones = [];

function setupDropZone(dropElement, onFileDropped) {
    if (!dropElement) return;

    let dragCounter = 0;

    dropElement.addEventListener('dragenter', (e) => {
        e.preventDefault();
        dragCounter++;
        dropElement.classList.add('dragover');
    });

    dropElement.addEventListener('dragover', (e) => {
        e.preventDefault();
        if (e.dataTransfer) {
            e.dataTransfer.dropEffect = 'copy';
        }
    });

    dropElement.addEventListener('dragleave', (e) => {
        e.preventDefault();
        dragCounter--;
        if (dragCounter <= 0) {
            dragCounter = 0;
            dropElement.classList.remove('dragover');
        }
    });

    dropElement.addEventListener('drop', (e) => {
        // Do not stop propagation; allow event to reach Wails's window drop handler
        dropElement.classList.remove('dragover');
        dragCounter = 0;
    });

    registeredDropZones.push({ element: dropElement, handler: onFileDropped });
}

// Track currently active navigation tab
let currentTab = 'hash-content';

// Initialize Wails native file drop listener
OnFileDrop(async (x, y, paths) => {
    if (!paths || paths.length === 0) return;
    const firstPath = paths[0];

    // Clean up all dragover and active target styles
    document.querySelectorAll('.dragover, .wails-drop-target-active').forEach(el => {
        el.classList.remove('dragover', 'wails-drop-target-active');
    });

    // 1. If two or more files are dropped on Compare tab, auto-assign slot A and slot B
    if (currentTab === 'compare-content' && paths.length >= 2) {
        if (typeof slotA !== 'undefined' && typeof slotB !== 'undefined') {
            slotA.setFile(paths[0]);
            slotB.setFile(paths[1]);
            return;
        }
    }

    // 2. Check if drop occurred within any registered drop zone
    const target = document.elementFromPoint(x, y);
    if (target) {
        for (const zone of registeredDropZones) {
            if (zone.element === target || zone.element.contains(target)) {
                await zone.handler(firstPath);
                return;
            }
        }
    }

    // 3. Fallback routing based on the active tab
    if (currentTab === 'compare-content') {
        if (typeof slotA !== 'undefined' && typeof slotB !== 'undefined') {
            if (!slotA.getFile()) {
                slotA.setFile(firstPath);
            } else {
                slotB.setFile(firstPath);
            }
        }
    } else if (currentTab === 'verify-content') {
        const verifySinglePanel = document.getElementById('verify-single-panel');
        const isSingleActive = verifySinglePanel && verifySinglePanel.classList.contains('active');
        if (isSingleActive) {
            const manualFileInput = document.getElementById('manual-file');
            if (manualFileInput) {
                manualFileInput.value = firstPath;
                showToast("File Selected", getBaseName(firstPath), "info");
            }
        } else {
            try {
                const content = await ReadFileContent(firstPath);
                const hashInput = document.getElementById('hash-input');
                if (hashInput) {
                    hashInput.value = content;
                    showToast("Checksum File Loaded", getBaseName(firstPath), "success");
                }
            } catch (err) {
                showToast("Error", "Could not read checksum file: " + err, "error");
            }
        }
    } else {
        setHashFile(firstPath);
    }
}, false);

// ============================================================================
// Navigation Tabs
// ============================================================================
const navTabs = document.querySelectorAll('.nav-tab');
const tabPanes = document.querySelectorAll('.tab-pane');

navTabs.forEach(tab => {
    tab.addEventListener('click', () => {
        const targetPaneId = tab.getAttribute('data-tab');
        currentTab = targetPaneId;
        navTabs.forEach(t => t.classList.remove('active'));
        tabPanes.forEach(p => p.classList.remove('active'));

        tab.classList.add('active');
        const targetPane = document.getElementById(targetPaneId);
        if (targetPane) targetPane.classList.add('active');
    });
});

// Verify Sub-Tabs
const verifyBatchTab = document.getElementById('verify-batch-tab');
const verifySingleTab = document.getElementById('verify-single-tab');
const verifyBatchPanel = document.getElementById('verify-batch-panel');
const verifySinglePanel = document.getElementById('verify-single-panel');

if (verifyBatchTab && verifySingleTab) {
    verifyBatchTab.addEventListener('click', () => {
        verifyBatchTab.classList.add('active');
        verifySingleTab.classList.remove('active');
        verifyBatchPanel.classList.add('active');
        verifySinglePanel.classList.remove('active');
    });

    verifySingleTab.addEventListener('click', () => {
        verifySingleTab.classList.add('active');
        verifyBatchTab.classList.remove('active');
        verifySinglePanel.classList.add('active');
        verifyBatchPanel.classList.remove('active');
    });
}

// ============================================================================
// TAB 1: Hash Generator
// ============================================================================
let currentFile = null;
let currentHashes = [];

const selectFileBtn = document.getElementById('select-file-btn');
const clearBtn = document.getElementById('clear-btn');
const hashDropZone = document.getElementById('hash-drop-zone');
const fileInfoBanner = document.getElementById('file-info-banner');
const fileNameDisplay = document.getElementById('file-name-display');
const filePathDisplay = document.getElementById('file-path');
const computeBtn = document.getElementById('compute-btn');
const computeBtnText = document.getElementById('compute-btn-text');
const copyAllBtn = document.getElementById('copy-all-btn');
const hashResultsList = document.getElementById('hash-results-list');
const resultsCountBadge = document.getElementById('results-count-badge');
const algorithmSelect = document.getElementById('algorithm-select');

function setHashFile(path) {
    if (!path) return;
    currentFile = path;
    fileNameDisplay.textContent = getBaseName(path);
    filePathDisplay.textContent = path;
    fileInfoBanner.classList.remove('hidden');
    hashDropZone.classList.add('hidden');
    showToast("File Selected", getBaseName(path), "info");
}

function clearHashFile() {
    currentFile = null;
    currentHashes = [];
    fileInfoBanner.classList.add('hidden');
    hashDropZone.classList.remove('hidden');
    copyAllBtn.classList.add('hidden');
    resultsCountBadge.textContent = "0 results";
    hashResultsList.innerHTML = `
        <div class="empty-state">
            <div class="empty-icon">
                <svg viewBox="0 0 24 24" width="36" height="36" fill="none" stroke="currentColor" stroke-width="1.5">
                    <line x1="4" y1="9" x2="20" y2="9"></line>
                    <line x1="4" y1="15" x2="20" y2="15"></line>
                    <line x1="10" y1="3" x2="8" y2="21"></line>
                    <line x1="16" y1="3" x2="14" y2="21"></line>
                </svg>
            </div>
            <div class="empty-title">No Hashes Generated Yet</div>
            <div class="empty-desc">Select a file and click "Generate Hash" to compute checksums.</div>
        </div>
    `;
}

selectFileBtn.addEventListener('click', async () => {
    try {
        const file = await OpenFile();
        if (file) setHashFile(file);
    } catch (e) {
        showToast("Error", "Could not open file dialog: " + e, "error");
    }
});

clearBtn.addEventListener('click', clearHashFile);
setupDropZone(hashDropZone, setHashFile);
setupDropZone(fileInfoBanner, setHashFile);

function renderHashCards(results) {
    currentHashes = results;
    hashResultsList.innerHTML = '';
    resultsCountBadge.textContent = `${results.length} ${results.length === 1 ? 'hash' : 'hashes'}`;

    if (results.length > 0) {
        copyAllBtn.classList.remove('hidden');
    } else {
        copyAllBtn.classList.add('hidden');
    }

    results.forEach(res => {
        const card = document.createElement('div');
        card.className = 'hash-card';

        if (res.Error) {
            card.innerHTML = `
                <div class="hash-meta">
                    <span class="hash-algo-pill" style="background:var(--danger-bg);color:var(--danger-color);">${escapeHtml(res.Algorithm)}</span>
                    <div class="hash-value" style="color:var(--danger-color);">Error: ${escapeHtml(res.Error)}</div>
                </div>
            `;
        } else {
            const timeStr = formatSeconds(res.Duration);
            const sizeStr = formatBytes(res.Size);

            card.innerHTML = `
                <div class="hash-meta">
                    <span class="hash-algo-pill">${escapeHtml(res.Algorithm)}</span>
                    <div class="hash-value">${escapeHtml(res.Hash)}</div>
                    <div class="hash-extra">${sizeStr} • Computed in ${timeStr}</div>
                </div>
                <div class="hash-actions">
                    <button class="btn btn-secondary btn-sm copy-single-btn" title="Copy ${escapeHtml(res.Algorithm)} hash">
                        <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2">
                            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                        </svg>
                        <span>Copy</span>
                    </button>
                </div>
            `;

            card.querySelector('.copy-single-btn').addEventListener('click', () => {
                copyToClipboard(res.Hash, res.Algorithm);
            });
        }

        hashResultsList.appendChild(card);
    });
}

computeBtn.addEventListener('click', async () => {
    if (!currentFile) {
        showToast("No File Selected", "Please choose a file to hash first.", "error");
        return;
    }

    const algo = algorithmSelect.value;
    computeBtn.disabled = true;
    computeBtnText.textContent = "Computing...";
    setAppStatus("Computing...", true);

    try {
        if (algo === 'all') {
            const results = await HashAll(currentFile);
            renderHashCards(results);
            showToast("Complete", `Computed all ${results.length} hashes in single pass.`, "success");
        } else {
            const result = await HashFile(currentFile, algo);
            renderHashCards([result]);
            showToast("Complete", `Computed ${algo.toUpperCase()} hash.`, "success");
        }
    } catch (e) {
        showToast("Error", "Failed to compute hash: " + e, "error");
    } finally {
        computeBtn.disabled = false;
        computeBtnText.textContent = "Generate Hash";
        setAppStatus("Ready", false);
    }
});

copyAllBtn.addEventListener('click', async () => {
    if (!currentHashes || currentHashes.length === 0) return;
    const lines = currentHashes
        .filter(r => !r.Error)
        .map(r => `${r.Algorithm}: ${r.Hash}`)
        .join('\n');
    await copyToClipboard(lines, "All Hashes");
});

// ============================================================================
// TAB 2: Compare Two Files
// ============================================================================
function createFilePickerSlot({ browseBtn, clearBtn, dropZone, nameEl, sizeEl, label }) {
    let currentPath = null;

    function setFile(path) {
        if (!path) return;
        currentPath = path;
        nameEl.textContent = getBaseName(path);
        sizeEl.textContent = path;
        showToast(`${label} Selected`, getBaseName(path), "info");
    }

    function clear() {
        currentPath = null;
        nameEl.textContent = "No file selected";
        sizeEl.textContent = "Drop file here or browse";
    }

    if (browseBtn) {
        browseBtn.addEventListener('click', async () => {
            try {
                const file = await OpenFile();
                if (file) setFile(file);
            } catch (e) {
                showToast("Error", `Could not browse ${label}: ` + e, "error");
            }
        });
    }

    if (clearBtn) {
        clearBtn.addEventListener('click', clear);
    }

    if (dropZone) {
        setupDropZone(dropZone, setFile);
    }

    return {
        getFile: () => currentPath,
        setFile,
        clear
    };
}

const slotA = createFilePickerSlot({
    browseBtn: document.getElementById('browse-compare-1-btn'),
    clearBtn: document.getElementById('clear-compare-1-btn'),
    dropZone: document.getElementById('compare-drop-1'),
    nameEl: document.getElementById('compare-name-1'),
    sizeEl: document.getElementById('compare-size-1'),
    label: "File A"
});

const slotB = createFilePickerSlot({
    browseBtn: document.getElementById('browse-compare-2-btn'),
    clearBtn: document.getElementById('clear-compare-2-btn'),
    dropZone: document.getElementById('compare-drop-2'),
    nameEl: document.getElementById('compare-name-2'),
    sizeEl: document.getElementById('compare-size-2'),
    label: "File B"
});

const compareExecBtn = document.getElementById('compare-exec-btn');
const compareAlgoSelect = document.getElementById('compare-algo-select');
const compareResultContainer = document.getElementById('compare-result-container');

compareExecBtn.addEventListener('click', async () => {
    const fileA = slotA.getFile();
    const fileB = slotB.getFile();

    if (!fileA || !fileB) {
        showToast("Missing Files", "Please select both File A and File B to compare.", "error");
        return;
    }

    const algo = compareAlgoSelect.value;
    compareExecBtn.disabled = true;
    setAppStatus("Comparing...", true);

    try {
        const res = await CompareFiles(fileA, fileB, algo);

        if (res.Error) {
            compareResultContainer.innerHTML = `
                <div class="compare-banner compare-banner-mismatch">
                    <div class="compare-banner-header">
                        <div class="compare-banner-icon">
                            <svg viewBox="0 0 24 24" width="26" height="26" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                        </div>
                        <div>
                            <div class="compare-banner-title">Comparison Error</div>
                            <div class="compare-banner-subtitle">${escapeHtml(res.Error)}</div>
                        </div>
                    </div>
                </div>
            `;
            showToast("Comparison Error", res.Error, "error");
            return;
        }

        if (res.Match) {
            compareResultContainer.innerHTML = `
                <div class="compare-banner compare-banner-match">
                    <div class="compare-banner-header">
                        <div class="compare-banner-icon">
                            <svg viewBox="0 0 24 24" width="28" height="28" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"></polyline></svg>
                        </div>
                        <div>
                            <div class="compare-banner-title">IDENTICAL FILES — Hashes Match 100%!</div>
                            <div class="compare-banner-subtitle">Both files produced the exact same ${escapeHtml(res.Algorithm)} cryptographic hash in ${formatSeconds(res.Duration)}.</div>
                        </div>
                    </div>
                    <div class="compare-details-box">
                        <div class="compare-detail-row">
                            <span class="compare-detail-label">Matching ${escapeHtml(res.Algorithm)} Hash:</span>
                            <div class="compare-hash-preview">
                                <span>${escapeHtml(res.Hash1)}</span>
                                <button class="btn btn-secondary btn-xs" id="copy-match-hash-btn">Copy Hash</button>
                            </div>
                        </div>
                        <div class="compare-detail-row">
                            <span class="compare-detail-label">File Details:</span>
                            <span style="font-size:12px;color:var(--text-secondary);">
                                File A (${formatBytes(res.File1Size)}) == File B (${formatBytes(res.File2Size)})
                            </span>
                        </div>
                    </div>
                </div>
            `;
            document.getElementById('copy-match-hash-btn').addEventListener('click', () => {
                copyToClipboard(res.Hash1, `${res.Algorithm} Hash`);
            });
            showToast("Hashes Match!", "Both files are completely identical.", "success");
        } else {
            const sizeDifference = res.File1Size === res.File2Size
                ? "Both files have identical byte length, but different contents!"
                : `File size differs: File A is ${formatBytes(res.File1Size)}, File B is ${formatBytes(res.File2Size)}.`;

            compareResultContainer.innerHTML = `
                <div class="compare-banner compare-banner-mismatch">
                    <div class="compare-banner-header">
                        <div class="compare-banner-icon">
                            <svg viewBox="0 0 24 24" width="26" height="26" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                        </div>
                        <div>
                            <div class="compare-banner-title">DIFFERENT FILES — Hash Mismatch!</div>
                            <div class="compare-banner-subtitle">${escapeHtml(sizeDifference)}</div>
                        </div>
                    </div>
                    <div class="compare-details-box">
                        <div class="compare-detail-row">
                            <span class="compare-detail-label">File A (${escapeHtml(res.File1Name)}) ${escapeHtml(res.Algorithm)}:</span>
                            <div class="compare-hash-preview">
                                <span>${escapeHtml(res.Hash1)}</span>
                                <button class="btn btn-secondary btn-xs" id="copy-hash1-btn">Copy</button>
                            </div>
                        </div>
                        <div class="compare-detail-row">
                            <span class="compare-detail-label">File B (${escapeHtml(res.File2Name)}) ${escapeHtml(res.Algorithm)}:</span>
                            <div class="compare-hash-preview">
                                <span>${escapeHtml(res.Hash2)}</span>
                                <button class="btn btn-secondary btn-xs" id="copy-hash2-btn">Copy</button>
                            </div>
                        </div>
                    </div>
                </div>
            `;
            document.getElementById('copy-hash1-btn').addEventListener('click', () => copyToClipboard(res.Hash1, "File A Hash"));
            document.getElementById('copy-hash2-btn').addEventListener('click', () => copyToClipboard(res.Hash2, "File B Hash"));
            showToast("Hashes Differ!", "The two files do not match.", "error");
        }
    } catch (e) {
        showToast("Error", "Comparison failed: " + e, "error");
    } finally {
        compareExecBtn.disabled = false;
        setAppStatus("Ready", false);
    }
});

// ============================================================================
// TAB 3: Verify Checksums (Batch Verification)
// ============================================================================
const loadChecksumFileBtn = document.getElementById('load-checksum-file-btn');
const clearChecksumInputBtn = document.getElementById('clear-checksum-input-btn');
const hashInput = document.getElementById('hash-input');
const verifyBtn = document.getElementById('verify-btn');
const verifyResultsContainer = document.getElementById('verify-results-container');
const badgeTotal = document.getElementById('badge-total');
const badgePass = document.getElementById('badge-pass');
const badgeFail = document.getElementById('badge-fail');
const badgeErr = document.getElementById('badge-err');

let currentChecksumFilePath = '';

loadChecksumFileBtn.addEventListener('click', async () => {
    try {
        const file = await SelectChecksumFile();
        if (file) {
            const content = await ReadFileContent(file);
            currentChecksumFilePath = file;
            hashInput.value = content;
            showToast("Checksum File Loaded", getBaseName(file), "success");
        }
    } catch (e) {
        showToast("Error", "Could not load checksum file: " + e, "error");
    }
});

clearChecksumInputBtn.addEventListener('click', () => {
    currentChecksumFilePath = '';
    hashInput.value = '';
    badgeTotal.textContent = "0 total";
    badgePass.classList.add('hidden');
    badgeFail.classList.add('hidden');
    badgeErr.classList.add('hidden');
    verifyResultsContainer.innerHTML = `
        <div class="empty-state">
            <div class="empty-icon">
                <svg viewBox="0 0 24 24" width="36" height="36" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                </svg>
            </div>
            <div class="empty-title">No Verification Performed Yet</div>
            <div class="empty-desc">Paste hashes or load a checksum file above and click Verify.</div>
        </div>
    `;
});

const batchDropCard = verifyBatchPanel ? verifyBatchPanel.querySelector('.glass-card') : null;
if (batchDropCard) {
    setupDropZone(batchDropCard, async (file) => {
        try {
            const content = await ReadFileContent(file);
            currentChecksumFilePath = file;
            hashInput.value = content;
            showToast("Checksum File Loaded", getBaseName(file), "success");
        } catch (e) {
            showToast("Error", "Could not load checksum file: " + e, "error");
        }
    });
}

verifyBtn.addEventListener('click', async () => {
    const inputVal = hashInput.value.trim();
    if (!inputVal) {
        showToast("Empty Input", "Please paste hashes or load a checksum file first.", "error");
        return;
    }

    verifyBtn.disabled = true;
    setAppStatus("Verifying...", true);

    try {
        let baseDir = '';
        if (currentChecksumFilePath) {
            const lastSep = Math.max(currentChecksumFilePath.lastIndexOf('/'), currentChecksumFilePath.lastIndexOf('\\'));
            if (lastSep !== -1) {
                baseDir = currentChecksumFilePath.substring(0, lastSep);
            }
        }

        const results = typeof VerifyChecksumText === 'function'
            ? await VerifyChecksumText(inputVal, baseDir)
            : await VerifyHashFile(await WriteTempFile(inputVal));

        let passCount = 0;
        let failCount = 0;
        let errCount = 0;

        verifyResultsContainer.innerHTML = '';

        results.forEach(r => {
            const item = document.createElement('div');
            item.className = 'verify-item';

            if (r.Error) {
                errCount++;
                item.classList.add('verify-error');
                item.innerHTML = `
                    <div class="verify-item-left">
                        <div class="verify-status-icon">
                            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>
                        </div>
                        <div class="verify-item-meta">
                            <div class="verify-item-name">${escapeHtml(getBaseName(r.FileName))}</div>
                            <div class="verify-item-sub" style="color:var(--warning-color);">Error: ${escapeHtml(r.Error)}</div>
                        </div>
                    </div>
                    <div class="verify-item-right">
                        <span class="badge badge-warning">ERROR</span>
                    </div>
                `;
            } else if (r.Valid) {
                passCount++;
                item.classList.add('verify-valid');
                item.innerHTML = `
                    <div class="verify-item-left">
                        <div class="verify-status-icon">
                            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"></polyline></svg>
                        </div>
                        <div class="verify-item-meta">
                            <div class="verify-item-name">${escapeHtml(getBaseName(r.FileName))}</div>
                            <div class="verify-item-sub">${escapeHtml(r.Algorithm)}: ${escapeHtml(r.Actual)}</div>
                        </div>
                    </div>
                    <div class="verify-item-right">
                        <span class="badge badge-success">✓ OK</span>
                    </div>
                `;
            } else {
                failCount++;
                item.classList.add('verify-invalid');
                item.innerHTML = `
                    <div class="verify-item-left">
                        <div class="verify-status-icon">
                            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                        </div>
                        <div class="verify-item-meta">
                            <div class="verify-item-name">${escapeHtml(getBaseName(r.FileName))}</div>
                            <div class="verify-item-sub">Expected: ${escapeHtml(r.Expected)} | Actual: ${escapeHtml(r.Actual)}</div>
                        </div>
                    </div>
                    <div class="verify-item-right">
                        <span class="badge badge-danger">✗ MISMATCH</span>
                    </div>
                `;
            }

            verifyResultsContainer.appendChild(item);
        });

        badgeTotal.textContent = `${results.length} total`;

        if (passCount > 0) {
            badgePass.textContent = `${passCount} passed`;
            badgePass.classList.remove('hidden');
        } else {
            badgePass.classList.add('hidden');
        }

        if (failCount > 0) {
            badgeFail.textContent = `${failCount} failed`;
            badgeFail.classList.remove('hidden');
        } else {
            badgeFail.classList.add('hidden');
        }

        if (errCount > 0) {
            badgeErr.textContent = `${errCount} errors`;
            badgeErr.classList.remove('hidden');
        } else {
            badgeErr.classList.add('hidden');
        }

        showToast("Verification Done", `${passCount} passed, ${failCount} failed, ${errCount} errors.`, failCount > 0 ? "error" : "success");
    } catch (e) {
        showToast("Verification Failed", "Error processing checksums: " + e, "error");
    } finally {
        verifyBtn.disabled = false;
        setAppStatus("Ready", false);
    }
});

// ============================================================================
// TAB 3: Verify Checksums (Single File Verification)
// ============================================================================
const manualHashInput = document.getElementById('manual-hash');
const manualAlgoSelect = document.getElementById('manual-algo');
const manualFileInput = document.getElementById('manual-file');
const browseBtn = document.getElementById('browse-btn');
const verifySingleBtn = document.getElementById('verify-single-btn');
const singleVerifyResultContainer = document.getElementById('single-verify-result-container');

browseBtn.addEventListener('click', async () => {
    try {
        const file = await OpenFile();
        if (file) manualFileInput.value = file;
    } catch (e) {
        showToast("Error", "Could not browse file: " + e, "error");
    }
});

const singleDropCard = verifySinglePanel ? verifySinglePanel.querySelector('.glass-card') : null;
if (singleDropCard) {
    setupDropZone(singleDropCard, (file) => {
        manualFileInput.value = file;
        showToast("File Selected", getBaseName(file), "info");
    });
}

verifySingleBtn.addEventListener('click', async () => {
    const hashVal = manualHashInput.value.trim();
    const filePath = manualFileInput.value.trim();
    const algo = manualAlgoSelect.value;

    if (!hashVal || !filePath) {
        showToast("Missing Inputs", "Please enter both the known hash and target file path.", "error");
        return;
    }

    verifySingleBtn.disabled = true;
    setAppStatus("Verifying...", true);

    try {
        const res = await VerifySingleHash(filePath, algo, hashVal);

        if (res.Error) {
            singleVerifyResultContainer.innerHTML = `
                <div class="compare-banner compare-banner-mismatch" style="margin-top:16px;">
                    <div class="compare-banner-header">
                        <div class="compare-banner-icon">
                            <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                        </div>
                        <div>
                            <div class="compare-banner-title">Error Verifying File</div>
                            <div class="compare-banner-subtitle">${escapeHtml(res.Error)}</div>
                        </div>
                    </div>
                </div>
            `;
            showToast("Error", res.Error, "error");
        } else if (res.Valid) {
            singleVerifyResultContainer.innerHTML = `
                <div class="compare-banner compare-banner-match" style="margin-top:16px;">
                    <div class="compare-banner-header">
                        <div class="compare-banner-icon">
                            <svg viewBox="0 0 24 24" width="26" height="26" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"></polyline></svg>
                        </div>
                        <div>
                            <div class="compare-banner-title">HASH VERIFIED — Integrity Confirmed!</div>
                            <div class="compare-banner-subtitle">The computed ${escapeHtml(res.Algorithm)} matches the known expected hash.</div>
                        </div>
                    </div>
                    <div class="compare-details-box">
                        <div class="compare-detail-row">
                            <span class="compare-detail-label">Verified Hash:</span>
                            <div class="compare-hash-preview">
                                <span>${escapeHtml(res.Actual)}</span>
                                <button class="btn btn-secondary btn-xs" id="copy-single-verified-btn">Copy</button>
                            </div>
                        </div>
                    </div>
                </div>
            `;
            document.getElementById('copy-single-verified-btn').addEventListener('click', () => {
                copyToClipboard(res.Actual, "Verified Hash");
            });
            showToast("Hash Verified", "File integrity confirmed!", "success");
        } else {
            singleVerifyResultContainer.innerHTML = `
                <div class="compare-banner compare-banner-mismatch" style="margin-top:16px;">
                    <div class="compare-banner-header">
                        <div class="compare-banner-icon">
                            <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                        </div>
                        <div>
                            <div class="compare-banner-title">HASH MISMATCH — File May Be Corrupted!</div>
                            <div class="compare-banner-subtitle">The computed hash differs from the expected hash.</div>
                        </div>
                    </div>
                    <div class="compare-details-box">
                        <div class="compare-detail-row">
                            <span class="compare-detail-label">Expected:</span>
                            <div class="compare-hash-preview">${escapeHtml(res.Expected)}</div>
                        </div>
                        <div class="compare-detail-row">
                            <span class="compare-detail-label">Actual:</span>
                            <div class="compare-hash-preview">${escapeHtml(res.Actual)}</div>
                        </div>
                    </div>
                </div>
            `;
            showToast("Hash Mismatch", "The file hash did not match the expected value.", "error");
        }
    } catch (e) {
        showToast("Error", "Single verification failed: " + e, "error");
    } finally {
        verifySingleBtn.disabled = false;
        setAppStatus("Ready", false);
    }
});
