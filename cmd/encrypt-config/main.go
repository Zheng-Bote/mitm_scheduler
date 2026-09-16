/**
 * SPDX-FileComment: Encrypt Config
 * SPDX-FileType: SOURCE
 * SPDX-FileContributor: ZHENG Robert
 * SPDX-FileCopyrightText: 2026 ZHENG Robert
 * SPDX-License-Identifier: Apache-2.0
 *
 * @file main.go
 * @brief CLI utility to encrypt and decrypt JSON configuration files
 * @version 1.1.0
 * @date 2026-09-16
 *
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @copyright Copyright (c) 2026 ZHENG Robert
 * @LICENSE Apache-2.0
 */

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"

	"go-scheduler/internal/config"
	"go-scheduler/internal/crypto"

	"golang.org/x/term"
)

func main() {
	decryptMode := flag.Bool("d", false, "Decrypt mode")
	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Println("Usage: encrypt-config [-d] <input_file> <output_file>")
		fmt.Println("  -d    Decrypt the input file instead of encrypting it")
		os.Exit(1)
	}

	inputPath := flag.Arg(0)
	outputPath := flag.Arg(1)

	// Read input file
	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		log.Fatalf("Failed to read input file: %v", err)
	}

	if *decryptMode {
		// --- DECRYPT MODE ---
		fmt.Print("Enter password to decrypt: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("\nFailed to read password: %v", err)
		}
		fmt.Println()

		plaintext, err := crypto.Decrypt(inputData, bytePassword)
		if err != nil {
			log.Fatalf("Decryption failed (wrong password or corrupted file): %v", err)
		}

		// Optionally format the JSON output nicely if it is valid JSON
		var out bytes.Buffer
		if err := json.Indent(&out, plaintext, "", "  "); err == nil {
			plaintext = out.Bytes()
		}

		if err := os.WriteFile(outputPath, plaintext, 0600); err != nil {
			log.Fatalf("Failed to write decrypted file: %v", err)
		}

		fmt.Printf("Successfully decrypted %s to %s\n", inputPath, outputPath)
	} else {
		// --- ENCRYPT MODE ---
		// Validate JSON
		var dbConfig config.DBConfig
		if err := json.Unmarshal(inputData, &dbConfig); err != nil {
			log.Fatalf("Invalid JSON format in input file: %v", err)
		}

		// Ask for password twice
		fmt.Print("Enter password to encrypt: ")
		bytePassword1, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("\nFailed to read password: %v", err)
		}
		fmt.Println()

		fmt.Print("Confirm password: ")
		bytePassword2, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("\nFailed to read password confirmation: %v", err)
		}
		fmt.Println()

		if string(bytePassword1) != string(bytePassword2) {
			log.Fatalf("Passwords do not match. Aborting.")
		}

		// Encrypt
		ciphertext, err := crypto.Encrypt(inputData, bytePassword1)
		if err != nil {
			log.Fatalf("Encryption failed: %v", err)
		}

		// Save
		if err := os.WriteFile(outputPath, ciphertext, 0600); err != nil {
			log.Fatalf("Failed to write encrypted file: %v", err)
		}

		fmt.Printf("Successfully encrypted %s to %s\n", inputPath, outputPath)
	}
}
