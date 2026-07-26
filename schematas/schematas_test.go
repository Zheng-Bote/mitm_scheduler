/**
 * SPDX-FileComment: FlatBuffers Schemas Tests
 * SPDX-FileType: SOURCE
 * SPDX-FileContributor: ZHENG Robert
 * SPDX-FileCopyrightText: 2026 ZHENG Robert
 * SPDX-License-Identifier: Apache-2.0
 *
 * @file schematas_test.go
 * @brief Unit tests for FlatBuffers schemas serialization and deserialization
 * @version 1.0.0
 * @date 2026-07-26
 *
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @copyright Copyright (c) 2026 ZHENG Robert
 * @LICENSE Apache-2.0
 */

package schematas

import (
	"testing"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
)

func TestDLQEntrySerialization(t *testing.T) {
	builder := flatbuffers.NewBuilder(1024)

	idOff := builder.CreateString("dlq-123")
	pkgOff := builder.CreateString("pkg-456")
	payloadOff := builder.CreateString("{\"key\": \"val\"}")
	errCodeOff := builder.CreateString("ERR_001")
	errMsgOff := builder.CreateString("something went wrong")
	nowStr := builder.CreateString("2026-07-26T12:00:00Z")

	DLQEntryStart(builder)
	DLQEntryAddId(builder, idOff)
	DLQEntryAddPackageId(builder, pkgOff)
	DLQEntryAddPayload(builder, payloadOff)
	DLQEntryAddErrorCode(builder, errCodeOff)
	DLQEntryAddErrorMessage(builder, errMsgOff)
	DLQEntryAddFailedAt(builder, nowStr)
	DLQEntryAddFailedAtUnixMs(builder, 1785067200000)
	DLQEntryAddResolved(builder, true)
	entryOff := DLQEntryEnd(builder)

	DLQEntryListStartEntriesVector(builder, 1)
	builder.PrependUOffsetT(entryOff)
	vecOff := builder.EndVector(1)

	DLQEntryListStart(builder)
	DLQEntryListAddEntries(builder, vecOff)
	rootOff := DLQEntryListEnd(builder)
	builder.Finish(rootOff)

	data := builder.FinishedBytes()

	list := GetRootAsDLQEntryList(data, 0)
	if list.EntriesLength() != 1 {
		t.Fatalf("expected 1 entry, got %d", list.EntriesLength())
	}

	var entry DLQEntry
	if !list.Entries(&entry, 0) {
		t.Fatalf("failed to get entry 0")
	}

	if string(entry.Id()) != "dlq-123" {
		t.Errorf("expected id dlq-123, got %s", string(entry.Id()))
	}
	if string(entry.PackageId()) != "pkg-456" {
		t.Errorf("expected pkg id pkg-456, got %s", string(entry.PackageId()))
	}
	if string(entry.ErrorCode()) != "ERR_001" {
		t.Errorf("expected err code ERR_001, got %s", string(entry.ErrorCode()))
	}
	if !entry.Resolved() {
		t.Errorf("expected resolved true, got false")
	}
	if entry.FailedAtUnixMs() != 1785067200000 {
		t.Errorf("expected ts 1785067200000, got %d", entry.FailedAtUnixMs())
	}
}

func TestSystemLogSerialization(t *testing.T) {
	builder := flatbuffers.NewBuilder(1024)

	lvlOff := builder.CreateString("INFO")
	compOff := builder.CreateString("scheduler")
	msgOff := builder.CreateString("system started")
	tsOff := builder.CreateString(time.Now().Format(time.RFC3339))

	SystemLogStart(builder)
	SystemLogAddId(builder, 42)
	SystemLogAddLevel(builder, lvlOff)
	SystemLogAddComponent(builder, compOff)
	SystemLogAddMessage(builder, msgOff)
	SystemLogAddTs(builder, tsOff)
	SystemLogAddTsUnixMs(builder, 123456789)
	logOff := SystemLogEnd(builder)

	SystemLogListStartLogsVector(builder, 1)
	builder.PrependUOffsetT(logOff)
	vecOff := builder.EndVector(1)

	SystemLogListStart(builder)
	SystemLogListAddLogs(builder, vecOff)
	rootOff := SystemLogListEnd(builder)
	builder.Finish(rootOff)

	list := GetRootAsSystemLogList(builder.FinishedBytes(), 0)
	if list.LogsLength() != 1 {
		t.Fatalf("expected 1 log, got %d", list.LogsLength())
	}

	var l SystemLog
	if !list.Logs(&l, 0) {
		t.Fatalf("failed to get log 0")
	}
	if l.Id() != 42 {
		t.Errorf("expected id 42, got %d", l.Id())
	}
	if string(l.Level()) != "INFO" {
		t.Errorf("expected level INFO, got %s", string(l.Level()))
	}
	if l.TsUnixMs() != 123456789 {
		t.Errorf("expected ts 123456789, got %d", l.TsUnixMs())
	}
}

func TestJobAuditLogSerialization(t *testing.T) {
	builder := flatbuffers.NewBuilder(1024)

	compOff := builder.CreateString("job-runner")
	msgOff := builder.CreateString("job executed")
	tsOff := builder.CreateString("2026-07-26T12:00:00Z")

	JobAuditLogStart(builder)
	JobAuditLogAddId(builder, 1)
	JobAuditLogAddRunId(builder, 100)
	JobAuditLogAddComponent(builder, compOff)
	JobAuditLogAddMessage(builder, msgOff)
	JobAuditLogAddTs(builder, tsOff)
	JobAuditLogAddTsUnixMs(builder, 999)
	logOff := JobAuditLogEnd(builder)

	JobAuditLogListStartLogsVector(builder, 1)
	builder.PrependUOffsetT(logOff)
	vecOff := builder.EndVector(1)

	JobAuditLogListStart(builder)
	JobAuditLogListAddLogs(builder, vecOff)
	rootOff := JobAuditLogListEnd(builder)
	builder.Finish(rootOff)

	list := GetRootAsJobAuditLogList(builder.FinishedBytes(), 0)
	if list.LogsLength() != 1 {
		t.Fatalf("expected 1 log, got %d", list.LogsLength())
	}

	var l JobAuditLog
	if !list.Logs(&l, 0) {
		t.Fatalf("failed to get log 0")
	}
	if l.RunId() != 100 {
		t.Errorf("expected run_id 100, got %d", l.RunId())
	}
	if string(l.Component()) != "job-runner" {
		t.Errorf("expected component job-runner, got %s", string(l.Component()))
	}
}

func TestAdminAuditLogSerialization(t *testing.T) {
	builder := flatbuffers.NewBuilder(1024)

	userOff := builder.CreateString("admin")
	actOff := builder.CreateString("delete_job")
	detOff := builder.CreateString("{\"name\": \"test-job\"}")
	tsOff := builder.CreateString("2026-07-26T12:00:00Z")

	AdminAuditLogStart(builder)
	AdminAuditLogAddId(builder, 5)
	AdminAuditLogAddUsername(builder, userOff)
	AdminAuditLogAddAction(builder, actOff)
	AdminAuditLogAddDetails(builder, detOff)
	AdminAuditLogAddTs(builder, tsOff)
	AdminAuditLogAddTsUnixMs(builder, 555)
	logOff := AdminAuditLogEnd(builder)

	AdminAuditLogListStartLogsVector(builder, 1)
	builder.PrependUOffsetT(logOff)
	vecOff := builder.EndVector(1)

	AdminAuditLogListStart(builder)
	AdminAuditLogListAddLogs(builder, vecOff)
	rootOff := AdminAuditLogListEnd(builder)
	builder.Finish(rootOff)

	list := GetRootAsAdminAuditLogList(builder.FinishedBytes(), 0)
	if list.LogsLength() != 1 {
		t.Fatalf("expected 1 log, got %d", list.LogsLength())
	}

	var l AdminAuditLog
	if !list.Logs(&l, 0) {
		t.Fatalf("failed to get log 0")
	}
	if string(l.Username()) != "admin" {
		t.Errorf("expected username admin, got %s", string(l.Username()))
	}
	if string(l.Action()) != "delete_job" {
		t.Errorf("expected action delete_job, got %s", string(l.Action()))
	}
	if string(l.Details()) != "{\"name\": \"test-job\"}" {
		t.Errorf("expected details {\"name\": \"test-job\"}, got %s", string(l.Details()))
	}
}

func TestTransformationErrorSerialization(t *testing.T) {
	builder := flatbuffers.NewBuilder(1024)

	idOff := builder.CreateString("err-1")
	corrOff := builder.CreateString("corr-1")
	topOff := builder.CreateString("employee_data")
	fldOff := builder.CreateString("email")
	ruleOff := builder.CreateString("email_format")
	msgOff := builder.CreateString("invalid email format")
	crOff := builder.CreateString("2026-07-26T12:00:00Z")

	TransformationErrorStart(builder)
	TransformationErrorAddId(builder, idOff)
	TransformationErrorAddCorrelationId(builder, corrOff)
	TransformationErrorAddTopic(builder, topOff)
	TransformationErrorAddFailedField(builder, fldOff)
	TransformationErrorAddRuleName(builder, ruleOff)
	TransformationErrorAddErrorMessage(builder, msgOff)
	TransformationErrorAddCreatedAt(builder, crOff)
	TransformationErrorAddCreatedAtUnixMs(builder, 777)
	errOff := TransformationErrorEnd(builder)

	TransformationErrorListStartErrorsVector(builder, 1)
	builder.PrependUOffsetT(errOff)
	vecOff := builder.EndVector(1)

	TransformationErrorListStart(builder)
	TransformationErrorListAddErrors(builder, vecOff)
	rootOff := TransformationErrorListEnd(builder)
	builder.Finish(rootOff)

	list := GetRootAsTransformationErrorList(builder.FinishedBytes(), 0)
	if list.ErrorsLength() != 1 {
		t.Fatalf("expected 1 error, got %d", list.ErrorsLength())
	}

	var e TransformationError
	if !list.Errors(&e, 0) {
		t.Fatalf("failed to get error 0")
	}
	if string(e.Id()) != "err-1" {
		t.Errorf("expected id err-1, got %s", string(e.Id()))
	}
	if string(e.FailedField()) != "email" {
		t.Errorf("expected failed_field email, got %s", string(e.FailedField()))
	}
	if string(e.RuleName()) != "email_format" {
		t.Errorf("expected rule_name email_format, got %s", string(e.RuleName()))
	}
}
