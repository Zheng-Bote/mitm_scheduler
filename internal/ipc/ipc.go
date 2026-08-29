/**
 * SPDX-FileComment: IPC
 * SPDX-FileType: SOURCE
 * SPDX-FileContributor: ZHENG Robert
 * SPDX-FileCopyrightText: 2026 ZHENG Robert
 * SPDX-License-Identifier: Apache-2.0
 *
 * @file ipc.go
 * @brief Unix Domain Socket IPC for job status event communication
 * @version 1.0.0
 * @date 2026-06-02
 *
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @copyright Copyright (c) 2026 ZHENG Robert
 * @LICENSE Apache-2.0
 */

// Package ipc implements inter-process communication over Unix Domain Sockets.
// Jobs send JSON-formatted status and audit events to the scheduler, which
// persists them to the database.
package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

// StatusEvent represents the JSON payload sent by jobs
type StatusEvent struct {
	RunID     int    `json:"run_id"`
	Type      string `json:"type"` // "status" (default) or "audit"
	Component string `json:"component"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Progress  int    `json:"progress"`
}

// Server listens for StatusEvents on a Unix Domain Socket
type Server struct {
	SocketPath string
	OnEvent    func(event StatusEvent)
}

// Start runs the Unix Domain Socket server
func (s *Server) Start() error {
	if _, err := os.Stat(s.SocketPath); err == nil {
		_ = os.Remove(s.SocketPath)
	}

	l, err := net.Listen("unix", s.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on unix socket: %w", err)
	}

	// Set permissions for the socket
	_ = os.Chmod(s.SocketPath, 0600)
	
	sem := make(chan struct{}, 100) // connection limit

	go func() {
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			sem <- struct{}{}
			go func(c net.Conn) {
				defer func() { <-sem }()
				s.handleConnection(c)
			}(conn)
		}
	}()

	return nil
}

// handleConnection reads JSON-Lines from a connected Unix socket client and
// dispatches each parsed StatusEvent to the registered OnEvent callback.
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	
	lr := io.LimitReader(conn, 64*1024) // 64KB limit
	scanner := bufio.NewScanner(lr)
	for scanner.Scan() {
		var event StatusEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err == nil {
			if s.OnEvent != nil {
				s.OnEvent(event)
			}
		}
	}
}

// Client is used by jobs to send events to the scheduler
type Client struct {
	SocketPath string
}

// SendEvent sends a StatusEvent to the scheduler socket
func (c *Client) SendEvent(event StatusEvent) error {
	conn, err := net.Dial("unix", c.SocketPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = conn.Write(append(data, '\n'))
	return err
}
