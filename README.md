# GopherHash - Modern Cryptographic File Hashing & Verification

A high-performance desktop application for computing, comparing, and verifying file checksums using MD5, SHA-1, SHA-256, SHA-512, BLAKE3, and CRC32 algorithms. Built with [Wails v2](https://wails.io/) using Go and modern web technologies.

---

## Features

- **High-Performance Streaming Engine**:
  - **Single-Pass Multi-Hashing**: Computes all 6 algorithms simultaneously via `io.MultiWriter` in a single disk read pass, eliminating disk thrashing and reducing I/O bottleneck by up to 6x.
  - Native streaming support for large files (ISO, archives, media files) with minimal memory footprint.
- **Hash Generator**:
  - Drag-and-drop file support with real-time metadata display (file size, full path).
  - Compute single hashes or all 6 algorithms (SHA-256, BLAKE3, CRC32, MD5, SHA-1, SHA-512) simultaneously.
  - 1-click clipboard copying for individual or all hashes with toast notifications.
- **Compare Two Files**:
  - Select or drag-and-drop File A and File B to compare their contents.
  - Choose algorithm (SHA-256, BLAKE3, CRC32, MD5, SHA-1, SHA-512).
  - High-visibility visual match banner (green for 100% identical match, red for difference).
  - Side-by-side hash comparison and size difference detection.
- **Verify Checksums**:
  - **Batch Verification**: Directly load `.sha256`, `.md5`, `.b3`, `.blake3`, `.crc32`, `.sfv`, `.txt`, `.checksum` files or paste checksums in standard `sha256sum`/`b3sum`/`md5sum`/SFV or BSD/OpenSSL format.
  - **Single File Verification**: Validate any local file against an expected known hash string.
  - Formatted status cards with summary metrics (total, passed, failed, errors).
- **Modern UI & Aesthetic**:
  - Dark glassmorphic aesthetic with custom animations and dynamic status lighting.
  - Header status indicator pulsing during compute, compare, and verify operations.
  - Native toast notification system (no browser alert popups).
  - Clean monospace formatting for hash representations.
  - Cross-platform desktop support (Windows, macOS, Linux).

---

## Supported Algorithms

| Algorithm | Type | Digest Size | Primary Use Case |
|-----------|------|-------------|------------------|
| **SHA-256** | Cryptographic | 256 bits (64 hex) | Industry standard, secure verification |
| **BLAKE3** | Cryptographic | 256 bits (64 hex) | Ultra-fast hashing with hardware acceleration |
| **CRC32** | Checksum | 32 bits (8 hex) | High-speed data integrity & SFV files |
| **SHA-512** | Cryptographic | 512 bits (128 hex) | Ultra-high bit collision resistance |
| **SHA-1** | Cryptographic (Legacy) | 160 bits (40 hex) | Legacy checksum matching & Git commits |
| **MD5** | Cryptographic (Legacy) | 128 bits (32 hex) | Legacy download verification |

---

## Building

### Prerequisites

- Go 1.25+
- Node.js and npm
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Build Steps

1. Install dependencies:
   ```bash
   go mod download
   cd frontend && npm install && cd ..
   ```

2. Build the desktop executable:
   ```bash
   wails build
   ```

3. The compiled binary will be in `build/bin/gopherhash.exe` (Windows) or binary/app bundle on other platforms.

### Running in Development

```bash
wails dev
```

---

## Tech Stack

- **Backend**: Go (standard library crypto & `hash/crc32`; `lukechampine.com/blake3` for pure Go BLAKE3)
- **Frontend**: HTML5, Vanilla CSS Design System with Go-cyan accents, JavaScript (ES Modules / Vite)
- **Framework**: Wails v2
- **Webview**: WebView2 (Windows), WebKit (macOS / Linux)
