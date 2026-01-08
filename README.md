<div align="center">
  <img src="./assets/logo.png" alt="DevSnapshot Mascot" width="180">
  <br>

```text
  ____             ____                        _           _
 |  _ \  _____   _/ ___| _ __   __ _ _ __  ___| |__   ___ | |_
 | | | |/ _ \ \ / \___ \| '_ \ / _` | '_ \/ __| '_ \ / _ \| __|
 | |_| |  __/\ V /  ___) | | | | (_| | |_) \__ \ | | | (_) | |_
 |____/ \___| \_/  |____/|_| |_|\__,_| .__/|___/_| |_|\___/ \__|
                                     |_|
```

</div>

# DevSnapshot 📸

**The "Polaroid" of Development Environments.**

> _Share a fully reproducible development sandbox for a repo, issue, or code review in a single file._

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Status](https://img.shields.io/badge/Status-Beta-orange.svg)]()
[![SafarNow Innovation](https://img.shields.io/badge/SafarNow-Innovation-blue.svg)](https://safarnow.in/)

## 📖 Table of Contents

- [Why DevSnapshot?](#-why-devsnapshot)
- [Installation](#-installation)
- [Quick Start](#-quick-start-choose-your-os)
- [Usage (Create, Inspect, Start)](#-usage)
- [Multi-Language Support (Polyglot)](#-polyglot--wizard-mode)
- [Security (EnvGuard)](#-envguard-secrets-management)
- [Supported Environments](#-supported-environments)
- [How it Works](#-how-it-works)
- [Contributing](#-contributing)

---

DevSnapshot is a tiny, portable, **open-source** tool that works on **Windows, macOS, and Linux**. It makes onboarding as simple as "download + run". Instead of telling contributors to "install dependencies" and hope they match, you share a single `.devsnap` file.

**No heavy VMs. No account setup. Just code.**

---

## 🚀 Why DevSnapshot?

- **Cross-Platform**: Runs natively on Windows, Mac (Intel/M1), and Linux.
- **Zero-Config Onboarding**: `devsnap start my-repo.devsnap` and you're coding in seconds.
- **Sherlock Detection**: Automatically detects project type (Node, Python, Go) even if manifests like `package.json` are missing.
- **Reproducible Bug Reports**: Attach a snapshot to a GitHub issue. Maintainers see exactly what you see.
- **Lightweight & Portable**: Archives only your code and a "brain" (metadata). No massive Docker images.

---

## 📦 Installation

DevSnapshot is a single binary file. Run it from anywhere.

### 1. Build from Source

**Windows**:

```powershell
go build -o devsnap.exe
```

**Mac (Apple Silicon)**:

```bash
GOOS=darwin GOARCH=arm64 go build -o devsnap-mac-arm64
chmod +x devsnap-mac-arm64
```

**Linux**:

```bash
GOOS=linux GOARCH=amd64 go build -o devsnap-linux
chmod +x devsnap-linux
```

### 2. Add to PATH (Global Access)

Run this in PowerShell to access `devsnap` from any folder:

```powershell
[System.Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\path\to\devsnap_folder", [System.EnvironmentVariableTarget]::User)
```

---

---

## 🏁 Quick Start: Choose Your OS

Since this is a standalone binary (no installation needed), how you run it depends on your OS:

### 🪟 Windows

1. Open **PowerShell** in the folder with `devsnap.exe`.
2. Run with `.\`:

```powershell
.\devsnap.exe create
```

### 🍎 Mac (macOS)

1. Open **Terminal**.
2. Make it executable first: `chmod +x devsnap-mac-arm64`
3. Run with `./`:

```bash
./devsnap-mac-arm64 create
```

### 🐧 Linux

1. Open **Terminal**.
2. Make it executable: `chmod +x devsnap-linux`
3. Run with `./`:

```bash
./devsnap-linux create
```

---

## 🛠️ Usage

### 1. Create a Snapshot (`create`)

Scan your current project and package it into a `.devsnap` file.

```powershell
cd my-project
devsnap create
# 📸 Snapping my-project...
# ✅ Snapshot ready: my-project.devsnap
```

**🕵️ Sherlock Mode (Advanced Detection)**
DevSnapshot features an intelligent "Sherlock" engine that works even when `package.json` or `go.mod` is missing:

1.  **Recursively Scans** for code files (`.js`, `.ts`, `.go`, etc.).
2.  **Identifies Dependencies** by reading `import` and `require` statements.
3.  **Resolves Versions** using a **Hybrid Strategy**:
    - **Source Truth**: Checks `node_modules` or `go.mod` for exact versions.
    - **CLI Check**: If missing, runs `npm list` or `go list`.
    - **Fallback**: Defaults to `latest`.
4.  **Generates `.devpack`**: Creates a `dependencies.devpack` file to lock this environment.

### 2. Inspect a Snapshot (`inspect`)

See exactly what's inside before you unzip it.

```powershell
devsnap inspect my-project.devsnap
# 🔍 Snapshot Metadata
# Name:        taskassignly
# Environment: node >=18.0.0
# Setup Cmd:   [#DEVPACK_INSTALL]
# Run Cmd:     npx vite
```

### 3. Start the Sandbox (`start`)

Unpacks to a safe sandbox (`.devsnap_sandbox`) and launches the environment.

#### Auto Mode (Default)

automatically installs dependencies and starts the app.

```powershell
devsnap start my-project.devsnap
# 🚀 Starting sandbox...
# 📦 Installing imports from devpack...
# ▶️ Running: npx vite
```

#### Manual Control Mode (`--manual`)

Gives you full control over every step.

```powershell
devsnap start my-project.devsnap --manual
# [?] Install Node dependencies (15 packages)? (Y/n):
```

---

## 🧙‍♂️ Polyglot & Wizard Mode

DevSnapshot now supports **Multi-Language Projects** (e.g., a Python backend with a Node.js frontend).

### 1. Unified Detection

It scans your entire project and detects **ALL** supported environments simultaneously:

```text
Detected polyglot-project [node (>=18.0.0), python (3.10)]
```

### 2. Interactive Wizard 🪄

When running `devsnap start` (especially with `--manual`), the **Interactive Wizard** guides you through the setup for _each_ environment sequentially:

1.  **Runtime Check**: Verifies you have the necessary tools (e.g., checks for `python` and `node`).
2.  **Sequential Install**: Prompts to install Node dependencies, then Python dependencies.
3.  **Controlled Launch**: Allows you to start services one by one.

---

## 🔐 EnvGuard (Secrets Management)

Never accidentally leak API keys again. DevSnapshot automatically scans your code for environment variable usage (e.g., `process.env.API_KEY`, `os.getenv("SECRET")`).

1.  **Detection**: Finds all required keys during `create`.
2.  **Exclusion**: **Ignoring** local `.env` files to prevent leaks.
3.  **Restoration**:
    - Generates a template `.env` in the sandbox.
    - **Prompts** you to enter missing secrets securely at runtime.
    - Loads them into the process for that session only.

---

## 🌍 Supported Environments

| Language               | Manifest           | Status      | Notes                                                            |
| :--------------------- | :----------------- | :---------- | :--------------------------------------------------------------- |
| **Node.js**            | `package.json`     | ✅ Stable   | Works for all frameworks (React, Vue, Next, etc.)                |
| **Angular**            | `angular.json`     | ✅ Stable   | Auto-detects Angular & resolves core version                     |
| **Go**                 | `go.mod`           | ✅ Stable   | Parses `go.mod` or scans imports + `go list` restoration         |
| **Python**             | `requirements.txt` | ✅ Stable   | Standard pip install & run                                       |
| **Sherlock (Generic)** | _Missing_          | 🚀 **Live** | Smart detection for Node, Python & Go projects without manifests |
| **Polyglot**           | _Mixed_            | ✨ **New**  | Supports **Node + Python + Go** in the same repo                 |

> **Note**: Sherlock Mode is currently in a **Testing Phase**. While it often works like magic, always verify the generated `.devpack` for complex projects.

---

---

## 🧩 How it Works

DevSnapshot isn't magic—it's just smart archiving.

1.  **Analysis (The "Brain")**:
    - It scans your code to identify languages, frameworks, and dependencies.
    - It ignores heavy folders like `node_modules`, `.venv`, or `target` to keep the file small (kB/MBs, not GBs).
2.  **Snapshotting**:
    - It bundles your source code + a `snapshot.json` metadata file into a compressed `.devsnap` archive.
    - It generates a `dependencies.devpack` (a lockfile of lockfiles) to ensure identical versions.
3.  **Sandboxing**:
    - When you run `start`, it unpacks into a `.devsnap_sandbox` folder.
    - It _reconstructs_ the environment by extracting code and freshly installing dependencies using the native package manager (npm, pip, go, cargo).

This ensures **Zero Pollution** on your main machine and **100% Reproducibility**.

---

---

## 🛣️ Roadmap & Vision

We are building the future of **Portable Development Environments**. Our goal is to make "it works on my machine" a phrase of the past.

### 🌟 The "Why" (Philosophy)

Modern development is too fragmented. Docker is heavy. Nix is complex.
DevSnapshot bridges the gap: **Native performance with Container-like reproducibility.**

### 📍 Current Focus (v1.0)

- ✅ Core Polyglot support (Node, Go, Python, Rust, PHP, Java)
- ✅ Intelligent "Sherlock" detection
- ✅ Cross-platform binaries

### 🔭 Future Horizons (v2.0+)

- [ ] **Cloud Snapshots**: `devsnap push/pull` to S3 or GitHub Packages.
- [ ] **IDE Integration**: VS Code extension to auto-load snapshots.
- [ ] **Deep Containerization**: Optional lightweight isolation using OS-native features (verify/jail).
- [ ] **Plugin System**: Allow community to write detectors for new languages (Ruby, Haskell, Swift).
- [ ] **GUI Wizard**: A desktop app for visual snapshot management.

### 📊 Performance Goals

- **Startup Time**: < 500ms for cached snapshots.
- **Archive Size**: < 10MB for typical microservices (code-only).

---

## 🤝 Contributing

This is a completely **Open Source** project. We welcome contributions from the community!

- Fork the repo
- Submit Pull Requests
- Report Issues

## � Other Innovative Products from SafarNow

- **[BEMP (Browser Enabled Model Protocol)](https://github.com/asish231/BEMP-Browser-Enabled-Model-Protocol-)**: A local, open-source bridge that turns web-based AI interfaces (Gemini, ChatGPT, DeepSeek, Qwen, Kimi, Venice, Blackbox, and others) into an API-like protocol you can call from your own code.

## �👨‍💻 Author & Credits

**Author**: Asish Kumar Sharma
**Role**: Founder & CEO @ **[SafarNow](https://www.safarnow.in)**

> **A SafarNow Innovation Product**

**Contact**:
📧 [asishkksharma@gmail.com](mailto:asishkksharma@gmail.com)

## 📄 License & Usage Policy

**DevSnapshot** is proudly **Open Source** under the **[MIT License](./LICENSE)**.

### 🔓 Why MIT? (The "Openness" Philosophy)

We believe developer tools should be foundational infrastructure, accessible to everyone. By choosing MIT, we ensure:

1.  **Zero Friction**: You can use this tool in personal, academic, or **commercial** projects without asking for permission.
2.  **Freedom to Fork**: If you need a custom version for your enterprise, you are free to modify the source code.
3.  **Community Ownership**: No vendor lock-in. If we stop maintaining it, the community can take over.

### ✅ What You CAN Do

- Use this tool for **Commercial** purposes (e.g., inside your startup).
- Modify the source code to fit your needs.
- Distribute copies of the tool to your team.
- Charge for services that use this tool (e.g., "DevOps consulting using DevSnapshot").

### ❌ What You CANNOT Do

- Hold the authors liable for any damages (software is provided "as is").
- Remove the original copyright notice and license text from the source code.

> **In summary**: As long as you keep the copyright header, you can do whatever you want with this code. Build, break, and scale. 🚀

---

## ⚠️ Digital Signatures & Liability Disclaimer

Please note that the provided binaries are currently **NOT digitally signed**.

- **Why?** Code signing certificates are costly for early-stage open-source projects. We plan to implement them in future releases.
- **Implication**: Your OS (Windows SmartScreen, macOS Gatekeeper) might warn you that the "Publisher is unknown." using the binaries.
- **Liability**: **The author (Asish Kumar Sharma) and SafarNow are NOT responsible for any damages** caused by the use of this software. You use it entirely at your own risk.

> **Paranoid?** That's good! Since this is open source, you don't have to trust our binaries. You can consistently audit the code and **Build from Source** (see Installation).

---

## 💌 A Message from the Team

> _"We hope DevSnapshot becomes the most credible tool in your workflow. We believe that **the more problems we face, the better we get**."_

This tool is designed to be **super easy**. If you face any issues or have needs, please **contact the author**. We are building this for you.

All the best,
**The SafarNow Team**
