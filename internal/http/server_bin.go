/**
 * SPDX-FileComment: HTTP Server Binary Handlers
 * SPDX-FileType: SOURCE
 * SPDX-FileContributor: ZHENG Robert
 * SPDX-FileCopyrightText: 2026 ZHENG Robert
 * SPDX-License-Identifier: Apache-2.0
 *
 * @file server_bin.go
 * @brief FlatBuffer HTTP API endpoints for system logs, audit logs, DLQ, and transformation errors
 * @version 1.0.0
 * @date 2026-07-26
 *
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @copyright Copyright (c) 2026 ZHENG Robert
 * @LICENSE Apache-2.0
 */

package http

import (
	"fmt"
	"net/http"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"

	"go-scheduler/schematas"
)

// handleDLQBin handles GET requests for dead letter queue entries as FlatBuffers binary data.
func (s *Server) handleDLQBin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	entries, err := s.Repo.GetDLQEntries(r.Context(), 100)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Failed to fetch DLQ entries: %v", err)))
		return
	}

	builder := flatbuffers.NewBuilder(1024)
	var entryOffsets []flatbuffers.UOffsetT

	for _, e := range entries {
		idOff := builder.CreateString(e.ID)
		var pkgIDOff flatbuffers.UOffsetT
		if e.PackageID != nil {
			pkgIDOff = builder.CreateString(*e.PackageID)
		}
		payloadOff := builder.CreateString(e.Payload)
		var errCodeOff flatbuffers.UOffsetT
		if e.ErrorCode != nil {
			errCodeOff = builder.CreateString(*e.ErrorCode)
		}
		var errMsgOff flatbuffers.UOffsetT
		if e.ErrorMessage != nil {
			errMsgOff = builder.CreateString(*e.ErrorMessage)
		}
		failedAtOff := builder.CreateString(e.FailedAt.Format(time.RFC3339))
		failedAtUnix := e.FailedAt.UnixNano() / 1e6

		var resolvedAtOff flatbuffers.UOffsetT
		var resolvedAtUnix int64
		if e.ResolvedAt != nil {
			resolvedAtOff = builder.CreateString(e.ResolvedAt.Format(time.RFC3339))
			resolvedAtUnix = e.ResolvedAt.UnixNano() / 1e6
		}

		schematas.DLQEntryStart(builder)
		schematas.DLQEntryAddId(builder, idOff)
		if pkgIDOff > 0 {
			schematas.DLQEntryAddPackageId(builder, pkgIDOff)
		}
		schematas.DLQEntryAddPayload(builder, payloadOff)
		if errCodeOff > 0 {
			schematas.DLQEntryAddErrorCode(builder, errCodeOff)
		}
		if errMsgOff > 0 {
			schematas.DLQEntryAddErrorMessage(builder, errMsgOff)
		}
		schematas.DLQEntryAddFailedAt(builder, failedAtOff)
		schematas.DLQEntryAddFailedAtUnixMs(builder, failedAtUnix)
		schematas.DLQEntryAddResolved(builder, e.Resolved)
		if resolvedAtOff > 0 {
			schematas.DLQEntryAddResolvedAt(builder, resolvedAtOff)
			schematas.DLQEntryAddResolvedAtUnixMs(builder, resolvedAtUnix)
		}
		entryOff := schematas.DLQEntryEnd(builder)
		entryOffsets = append(entryOffsets, entryOff)
	}

	schematas.DLQEntryListStartEntriesVector(builder, len(entryOffsets))
	for i := len(entryOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(entryOffsets[i])
	}
	vecOff := builder.EndVector(len(entryOffsets))

	schematas.DLQEntryListStart(builder)
	schematas.DLQEntryListAddEntries(builder, vecOff)
	rootOff := schematas.DLQEntryListEnd(builder)
	builder.Finish(rootOff)

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(builder.FinishedBytes())
}

// handleDownloadSystemLogsBin retrieves system logs within an optional date range
// and streams them as a FlatBuffers binary file download.
func (s *Server) handleDownloadSystemLogsBin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	from, err := parseDateParam(r.URL.Query().Get("from"))
	if err != nil {
		http.Error(w, "Invalid 'from' parameter. Use RFC3339 or YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	to, err := parseDateParam(r.URL.Query().Get("to"))
	if err != nil {
		http.Error(w, "Invalid 'to' parameter. Use RFC3339 or YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	if to != nil && len(r.URL.Query().Get("to")) == 10 {
		*to = to.Add(24*time.Hour - time.Second)
	}

	logs, err := s.Repo.GetSystemLogs(r.Context(), from, to)
	if err != nil {
		s.Repo.LogAdminAction(r.Context(), username, "download_system_logs_bin_fail", err.Error())
		http.Error(w, "Failed to retrieve system logs", http.StatusInternalServerError)
		return
	}

	s.Repo.LogAdminAction(r.Context(), username, "download_system_logs_bin_success", map[string]interface{}{
		"from":  from,
		"to":    to,
		"count": len(logs),
	})

	builder := flatbuffers.NewBuilder(1024)
	var logOffsets []flatbuffers.UOffsetT

	for _, l := range logs {
		levelOff := builder.CreateString(l.Level)
		compOff := builder.CreateString(l.Component)
		msgOff := builder.CreateString(l.Message)
		tsOff := builder.CreateString(l.TS.Format(time.RFC3339))

		schematas.SystemLogStart(builder)
		schematas.SystemLogAddId(builder, int64(l.ID))
		schematas.SystemLogAddLevel(builder, levelOff)
		schematas.SystemLogAddComponent(builder, compOff)
		schematas.SystemLogAddMessage(builder, msgOff)
		schematas.SystemLogAddTs(builder, tsOff)
		schematas.SystemLogAddTsUnixMs(builder, l.TS.UnixNano()/1e6)
		logOff := schematas.SystemLogEnd(builder)
		logOffsets = append(logOffsets, logOff)
	}

	schematas.SystemLogListStartLogsVector(builder, len(logOffsets))
	for i := len(logOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(logOffsets[i])
	}
	vecOff := builder.EndVector(len(logOffsets))

	schematas.SystemLogListStart(builder)
	schematas.SystemLogListAddLogs(builder, vecOff)
	rootOff := schematas.SystemLogListEnd(builder)
	builder.Finish(rootOff)

	w.Header().Set("Content-Disposition", "attachment; filename=system_logs.bin")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(builder.FinishedBytes())
}

// handleDownloadJobAuditLogsBin retrieves job audit logs within an optional date range
// and streams them as a FlatBuffers binary file download.
func (s *Server) handleDownloadJobAuditLogsBin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	from, err := parseDateParam(r.URL.Query().Get("from"))
	if err != nil {
		http.Error(w, "Invalid 'from' parameter. Use RFC3339 or YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	to, err := parseDateParam(r.URL.Query().Get("to"))
	if err != nil {
		http.Error(w, "Invalid 'to' parameter. Use RFC3339 or YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	if to != nil && len(r.URL.Query().Get("to")) == 10 {
		*to = to.Add(24*time.Hour - time.Second)
	}

	logs, err := s.Repo.GetJobAuditLogs(r.Context(), from, to)
	if err != nil {
		s.Repo.LogAdminAction(r.Context(), username, "download_job_audit_logs_bin_fail", err.Error())
		http.Error(w, "Failed to retrieve job audit logs", http.StatusInternalServerError)
		return
	}

	s.Repo.LogAdminAction(r.Context(), username, "download_job_audit_logs_bin_success", map[string]interface{}{
		"from":  from,
		"to":    to,
		"count": len(logs),
	})

	builder := flatbuffers.NewBuilder(1024)
	var logOffsets []flatbuffers.UOffsetT

	for _, l := range logs {
		compOff := builder.CreateString(l.Component)
		msgOff := builder.CreateString(l.Message)
		tsOff := builder.CreateString(l.TS.Format(time.RFC3339))

		schematas.JobAuditLogStart(builder)
		schematas.JobAuditLogAddId(builder, int64(l.ID))
		schematas.JobAuditLogAddRunId(builder, int64(l.RunID))
		schematas.JobAuditLogAddComponent(builder, compOff)
		schematas.JobAuditLogAddMessage(builder, msgOff)
		schematas.JobAuditLogAddTs(builder, tsOff)
		schematas.JobAuditLogAddTsUnixMs(builder, l.TS.UnixNano()/1e6)
		logOff := schematas.JobAuditLogEnd(builder)
		logOffsets = append(logOffsets, logOff)
	}

	schematas.JobAuditLogListStartLogsVector(builder, len(logOffsets))
	for i := len(logOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(logOffsets[i])
	}
	vecOff := builder.EndVector(len(logOffsets))

	schematas.JobAuditLogListStart(builder)
	schematas.JobAuditLogListAddLogs(builder, vecOff)
	rootOff := schematas.JobAuditLogListEnd(builder)
	builder.Finish(rootOff)

	w.Header().Set("Content-Disposition", "attachment; filename=job_audit_logs.bin")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(builder.FinishedBytes())
}

// handleDownloadAdminAuditLogsBin retrieves administrative audit logs within an optional date range
// and streams them as a FlatBuffers binary file download.
func (s *Server) handleDownloadAdminAuditLogsBin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	from, err := parseDateParam(r.URL.Query().Get("from"))
	if err != nil {
		http.Error(w, "Invalid 'from' parameter. Use RFC3339 or YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	to, err := parseDateParam(r.URL.Query().Get("to"))
	if err != nil {
		http.Error(w, "Invalid 'to' parameter. Use RFC3339 or YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	if to != nil && len(r.URL.Query().Get("to")) == 10 {
		*to = to.Add(24*time.Hour - time.Second)
	}

	logs, err := s.Repo.GetAdminAuditLogs(r.Context(), from, to)
	if err != nil {
		s.Repo.LogAdminAction(r.Context(), username, "download_admin_audit_logs_bin_fail", err.Error())
		http.Error(w, "Failed to retrieve admin audit logs", http.StatusInternalServerError)
		return
	}

	s.Repo.LogAdminAction(r.Context(), username, "download_admin_audit_logs_bin_success", map[string]interface{}{
		"from":  from,
		"to":    to,
		"count": len(logs),
	})

	builder := flatbuffers.NewBuilder(1024)
	var logOffsets []flatbuffers.UOffsetT

	for _, l := range logs {
		userOff := builder.CreateString(l.Username)
		actOff := builder.CreateString(l.Action)
		detOff := builder.CreateString(string(l.Details))
		tsOff := builder.CreateString(l.TS.Format(time.RFC3339))

		schematas.AdminAuditLogStart(builder)
		schematas.AdminAuditLogAddId(builder, int64(l.ID))
		schematas.AdminAuditLogAddUsername(builder, userOff)
		schematas.AdminAuditLogAddAction(builder, actOff)
		schematas.AdminAuditLogAddDetails(builder, detOff)
		schematas.AdminAuditLogAddTs(builder, tsOff)
		schematas.AdminAuditLogAddTsUnixMs(builder, l.TS.UnixNano()/1e6)
		logOff := schematas.AdminAuditLogEnd(builder)
		logOffsets = append(logOffsets, logOff)
	}

	schematas.AdminAuditLogListStartLogsVector(builder, len(logOffsets))
	for i := len(logOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(logOffsets[i])
	}
	vecOff := builder.EndVector(len(logOffsets))

	schematas.AdminAuditLogListStart(builder)
	schematas.AdminAuditLogListAddLogs(builder, vecOff)
	rootOff := schematas.AdminAuditLogListEnd(builder)
	builder.Finish(rootOff)

	w.Header().Set("Content-Disposition", "attachment; filename=admin_audit_logs.bin")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(builder.FinishedBytes())
}

// handleTransformationErrorsBin handles GET requests for transformation errors as FlatBuffers binary data.
func (s *Server) handleTransformationErrorsBin(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	res, err := s.Repo.GetTransformationErrors(r.Context(), 500)
	if err != nil {
		http.Error(w, "Failed to fetch transformation errors", http.StatusInternalServerError)
		return
	}

	s.Repo.LogAdminAction(r.Context(), username, "get_transformation_errors_bin", map[string]interface{}{
		"count": len(res),
	})

	builder := flatbuffers.NewBuilder(1024)
	var errOffsets []flatbuffers.UOffsetT

	for _, e := range res {
		idOff := builder.CreateString(e.ID)
		corrOff := builder.CreateString(e.CorrelationID)
		topOff := builder.CreateString(e.Topic)
		fldOff := builder.CreateString(e.FailedField)
		ruleOff := builder.CreateString(e.RuleName)
		msgOff := builder.CreateString(e.ErrorMessage)
		crOff := builder.CreateString(e.CreatedAt.Format(time.RFC3339))

		schematas.TransformationErrorStart(builder)
		schematas.TransformationErrorAddId(builder, idOff)
		schematas.TransformationErrorAddCorrelationId(builder, corrOff)
		schematas.TransformationErrorAddTopic(builder, topOff)
		schematas.TransformationErrorAddFailedField(builder, fldOff)
		schematas.TransformationErrorAddRuleName(builder, ruleOff)
		schematas.TransformationErrorAddErrorMessage(builder, msgOff)
		schematas.TransformationErrorAddCreatedAt(builder, crOff)
		schematas.TransformationErrorAddCreatedAtUnixMs(builder, e.CreatedAt.UnixNano()/1e6)
		errOff := schematas.TransformationErrorEnd(builder)
		errOffsets = append(errOffsets, errOff)
	}

	schematas.TransformationErrorListStartErrorsVector(builder, len(errOffsets))
	for i := len(errOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(errOffsets[i])
	}
	vecOff := builder.EndVector(len(errOffsets))

	schematas.TransformationErrorListStart(builder)
	schematas.TransformationErrorListAddErrors(builder, vecOff)
	rootOff := schematas.TransformationErrorListEnd(builder)
	builder.Finish(rootOff)

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(builder.FinishedBytes())
}
