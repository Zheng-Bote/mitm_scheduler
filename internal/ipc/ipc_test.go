package ipc

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIPCServer_Start(t *testing.T) {
	sockPath := filepath.Join(os.TempDir(), fmt.Sprintf("ipc_test_%d.sock", time.Now().UnixNano()))
	defer os.Remove(sockPath)

	srv := &Server{
		SocketPath: sockPath,
		OnEvent: func(e StatusEvent) {
			// dummy handler
		},
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	client := &Client{SocketPath: sockPath}
	err := client.SendEvent(StatusEvent{
		RunID:   1,
		Status:  "success",
		Message: "test message",
	})
	if err != nil {
		t.Errorf("Failed to send event: %v", err)
	}

	// Give it a tiny bit of time for async handleConnection
	time.Sleep(50 * time.Millisecond)
}
