package main

import (
	"archive/tar"
	"compress/gzip"
	"devsnap/pkg/create"
	"devsnap/pkg/metadata"
	"devsnap/pkg/start"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "create":
		handleCreate(os.Args[2:])
	case "start":
		handleStart(os.Args[2:])
	case "inspect":
		handleInspect(os.Args[2:])
	case "help":
		printHelp()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("DevSnapshot 📸 - The 'Polaroid' of Development Environments")
	fmt.Println("\nUsage:")
	fmt.Println("  devsnap <command> [arguments]")
	fmt.Println("\nCommands:")
	fmt.Println("  create   Scan current execution and create a .devsnap archive")
	fmt.Println("  start    Unpack and run a .devsnap snapshot")
	fmt.Println("  inspect  View metadata of a .devsnap snapshot")
	fmt.Println("  help     Show this help message")
}

func handleCreate(args []string) {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📸 Snapping %s...\n", wd)

	// 1. Scan Files
	fmt.Print("   • Scanning... ")
	files, err := create.ScanDirectory(wd)
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d files.\n", len(files))

	// 2. Detect Project Type
	fmt.Print("   • Detecting... ")
	env, cmds, name, reqVars := create.DetectProject(wd)
	fmt.Printf("Detected %s (%s).\n", name, env.Type)
	if len(reqVars) > 0 {
		fmt.Printf("   🔐 Detected %d required secrets (e.g. %s)\n", len(reqVars), reqVars[0])
	}

	// 3. Metadata
	meta := metadata.SnapshotMetadata{
		SchemaVersion: "1.0",
		Name:          name,
		CreatedAt:     time.Now().Format(time.RFC3339),
		Environment:   env,
		Commands:      cmds,
		RequiredVars:  reqVars,
	}

	// 4. Archive
	outputName := fmt.Sprintf("%s.devsnap", name)
	fmt.Printf("   • Packing... ")
	err = create.CreateArchive(wd, files, meta, outputName)
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Done!")

	fmt.Printf("\n✅ Snapshot ready: %s\n", outputName)
}

func handleStart(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: devsnap start <snapshot-file> [--manual|-m]")
		os.Exit(1)
	}

	var snapshotFile string
	manualMode := false

	for _, arg := range args {
		if arg == "--manual" || arg == "-m" {
			manualMode = true
		} else {
			snapshotFile = arg
		}
	}

	if snapshotFile == "" {
		fmt.Println("Usage: devsnap start <snapshot-file> [--manual|-m]")
		os.Exit(1)
	}

	// 1. Unpack
	// Use a local sandbox directory for visibility, as requested
	sandboxDir := ".devsnap_sandbox"
	fmt.Printf("📂 Opening snapshot %s to %s...\n", snapshotFile, sandboxDir)

	// Clean up previous sandbox if exists to ensure fresh start
	os.RemoveAll(sandboxDir)

	meta, err := start.Unpack(snapshotFile, sandboxDir)
	if err != nil {
		fmt.Printf("Error unpacking: %v\n", err)
		os.Exit(1)
	}

	// 2. Run
	err = start.Run(sandboxDir, meta, manualMode)
	if err != nil {
		fmt.Printf("Error running snapshot: %v\n", err)
		os.Exit(1)
	}
}

func handleInspect(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: devsnap inspect <snapshot-file>")
		os.Exit(1)
	}
	snapshotFile := args[0]

	file, err := os.Open(snapshotFile)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		fmt.Printf("Error reading gzip: %v\n", err)
		os.Exit(1)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error reading tar: %v\n", err)
			os.Exit(1)
		}

		if filepath.Base(header.Name) == "metadata.json" {
			var meta metadata.SnapshotMetadata
			bytes, _ := ioutil.ReadAll(tr)
			if err := json.Unmarshal(bytes, &meta); err != nil {
				fmt.Printf("Error parsing metadata: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("\n🔍 Snapshot Metadata")
			fmt.Println("--------------------")
			fmt.Printf("Name:        %s\n", meta.Name)
			fmt.Printf("Created:     %s\n", meta.CreatedAt)
			fmt.Printf("Environment: %s %s\n", meta.Environment.Type, meta.Environment.Version)
			fmt.Printf("Setup Cmd:   %v\n", meta.Commands.Setup)
			fmt.Printf("Run Cmd:     %s\n", meta.Commands.Run)
			return
		}
	}
	fmt.Println("Error: metadata.json not found in snapshot")
}
