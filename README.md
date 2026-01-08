```text
  ____             ____                        _           _
 |  _ \  _____   _/ ___| _ __   __ _ _ __  ___| |__   ___ | |_
 | | | |/ _ \ \ / \___ \| '_ \ / _` | '_ \/ __| '_ \ / _ \| __|
 | |_| |  __/\ V /  ___) | | | | (_| | |_) \__ \ | | | (_) | |_
 |____/ \___| \_/  |____/|_| |_|\__,_| .__/|___/_| |_|\___/ \__|
                                     |_|
```

# DevSnapshot 📸

**The "Polaroid" of Development Environments.**

> _Share a fully reproducible development sandbox for a repo, issue, or code review in a single file._

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Status](https://img.shields.io/badge/Status-Beta-orange.svg)]()
[![SafarNow Innovation](https://img.shields.io/badge/SafarNow-Innovation-blue.svg)](https://safarnow.in/)

DevSnapshot is a tiny, portable, **open-source** tool that makes onboarding as simple as "download + run". Instead of telling contributors to "install dependencies" and hope they match, you share a single `.devsnap` file.

**No heavy VMs. No account setup. Just code.**

---

## 🚀 Why DevSnapshot?

- **Zero-Config Onboarding**: `devsnap start my-repo.devsnap` and you're coding in seconds.
- **Sherlock Detection**: Automatically detects project type (Node, Python, Go) even if manifests like `package.json` are missing.
- **Reproducible Bug Reports**: Attach a snapshot to a GitHub issue. Maintainers see exactly what you see.
- **Lightweight & Portable**: Archives only your code and a "brain" (metadata). No massive Docker images.

---

## 📦 Installation

DevSnapshot is a single `.exe` file. Run it from anywhere.

### 1. Build from Source

```powershell
go build -o devsnap.exe
```

### 2. Add to PATH (Global Access)

Run this in PowerShell to access `devsnap` from any folder:

```powershell
[System.Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\path\to\devsnap_folder", [System.EnvironmentVariableTarget]::User)
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

## 🌍 Supported Environments

| Language               | Manifest           | Status      | Notes                                                    |
| :--------------------- | :----------------- | :---------- | :------------------------------------------------------- |
| **Node.js**            | `package.json`     | ✅ Stable   | Works for all frameworks (React, Vue, Next, etc.)        |
| **Angular**            | `angular.json`     | ✅ Stable   | Auto-detects Angular & resolves core version             |
| **Go**                 | `go.mod`           | ✅ Stable   | Parses `go.mod` or scans imports + `go list` restoration |
| **Python**             | `requirements.txt` | ✅ Stable   | Standard pip install & run                               |
| **Sherlock (Generic)** | _Missing_          | 🚀 **Live** | Smart detection for Node & Go projects without manifests |

> **Note**: Sherlock Mode is currently in a **Testing Phase**. While it often works like magic, always verify the generated `.devpack` for complex projects.

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

## 📄 License

MIT
