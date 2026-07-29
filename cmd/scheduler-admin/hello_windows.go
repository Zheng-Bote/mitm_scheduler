//go:build windows

/**
 * SPDX-FileComment: Scheduler Admin
 * SPDX-FileType: SOURCE
 * SPDX-FileContributor: ZHENG Robert
 * SPDX-FileCopyrightText: 2026 ZHENG Robert
 * SPDX-License-Identifier: Apache-2.0
 *
 * @file hello_windows.go
 * @brief Windows Hello biometric verification for admin tool
 * @version 1.0.0
 * @date 2026-06-02
 *
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @copyright Copyright (c) 2026 ZHENG Robert
 * @LICENSE Apache-2.0
 */

package main

import (
	"fmt"
	"time"
	"unsafe"

	"go-scheduler/windows/security/credentials/ui"

	"fyne.io/fyne/v2"
	"github.com/go-ole/go-ole"
	"github.com/saltosystems/winrt-go/windows/foundation"
)

// waitAsync blocks until the given WinRT IAsyncOperation completes, polling
// every 50 ms. It returns the operation's result pointer on success, or an
// error describing cancellation or the failed HRESULT.
func waitAsync(asyncOp *foundation.IAsyncOperation) (unsafe.Pointer, error) {
	disp, err := asyncOp.QueryInterface(ole.NewGUID(foundation.GUIDIAsyncInfo))
	if err != nil {
		return nil, err
	}
	asyncInfo := (*foundation.IAsyncInfo)(unsafe.Pointer(disp))
	defer asyncInfo.Release()

	for {
		status, err := asyncInfo.GetStatus()
		if err != nil {
			return nil, err
		}
		if status != 0 { // 0 is AsyncStatusStarted
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	status, err := asyncInfo.GetStatus()
	if err != nil {
		return nil, err
	}

	if status == 1 { // 1 is AsyncStatusCompleted
		return asyncOp.GetResults()
	} else if status == 2 {
		return nil, fmt.Errorf("canceled")
	} else {
		hres, err := asyncInfo.GetErrorCode()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed with HRESULT %08x", hres.Value)
	}
}

// verifyWindowsHello performs Windows Hello biometric verification using the
// WinRT UserConsentVerifier API. It checks availability first, then prompts
// the user for fingerprint / face verification. Returns true only when the
// user successfully verifies.
func verifyWindowsHello(w fyne.Window) (bool, error) {
	// Initialize OLE (COM) for the thread
	_ = ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
	defer ole.CoUninitialize()

	// 1. Check availability
	asyncOp, err := ui.UserConsentVerifierCheckAvailabilityAsync()
	if err != nil {
		return false, fmt.Errorf("failed to check availability: %w", err)
	}
	defer asyncOp.Release()

	resPtr, err := waitAsync(asyncOp)
	if err != nil {
		return false, err
	}
	availability := ui.UserConsentVerifierAvailability(uintptr(resPtr))
	if availability != ui.UserConsentVerifierAvailabilityAvailable {
		return false, fmt.Errorf("not available (status: %d)", availability)
	}

	// 2. Request verification
	asyncOpVerify, err := ui.UserConsentVerifierRequestVerificationAsync("Verifizieren Sie sich, um auf den Scheduler zuzugreifen.")
	if err != nil {
		return false, fmt.Errorf("failed to request verification: %w", err)
	}
	defer asyncOpVerify.Release()

	resPtrVerify, err := waitAsync(asyncOpVerify)
	if err != nil {
		return false, err
	}
	result := ui.UserConsentVerificationResult(uintptr(resPtrVerify))

	if result == ui.UserConsentVerificationResultVerified {
		return true, nil
	}
	if result == ui.UserConsentVerificationResultCanceled {
		return false, fmt.Errorf("verification canceled by user")
	}
	return false, fmt.Errorf("failed (result: %d)", result)
}
