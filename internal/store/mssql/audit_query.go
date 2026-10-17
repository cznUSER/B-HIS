package mssql

import (
	"context"
	"database/sql"
	"time"

	"github.com/cznUSER/B-HIS/internal/domain"
	"github.com/cznUSER/B-HIS/internal/port"
)

var _ port.UserOperationQueryStore = (*Store)(nil)

func (s *Store) ListUserOperations(ctx context.Context, q port.UserOperationQuery) ([]domain.UserOperationLog, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT TOP (@limit)
			a.audit_id, COALESCE(a.module_id, ''), COALESCE(m.module_name, ''), a.action, a.occurred_at,
			COALESCE(a.actor_id, ''), COALESCE(a.actor_username, ''), COALESCE(a.department_id, ''),
			COALESCE(a.description, a.action), COALESCE(a.client_ip, ''), COALESCE(a.client_mac, ''),
			COALESCE(a.build_version, ''), COALESCE(a.request_id, ''), a.outcome, COALESCE(a.error_code, '')
		FROM dbo.audit_log a
		LEFT JOIN dbo.access_module m ON m.module_id = a.module_id
		WHERE (@module_id IS NULL OR a.module_id = @module_id)
		  AND (@user_id IS NULL OR a.actor_id = @user_id)
		  AND (@begin_at IS NULL OR a.occurred_at >= @begin_at)
		  AND (@end_at IS NULL OR a.occurred_at < @end_at)
		  AND (@cursor_at IS NULL OR a.occurred_at < @cursor_at OR (a.occurred_at = @cursor_at AND a.audit_id < @cursor_id))
		ORDER BY a.occurred_at DESC, a.audit_id DESC
		OPTION (RECOMPILE)`,
		sql.Named("limit", q.Limit),
		sql.Named("module_id", nullIfEmpty(q.ModuleID)),
		sql.Named("user_id", nullIfEmpty(q.UserID)),
		sql.Named("begin_at", auditNullTime(q.Begin)),
		sql.Named("end_at", auditNullTime(q.EndExclusive)),
		sql.Named("cursor_at", auditNullTime(q.CursorOccurredAt)),
		sql.Named("cursor_id", nullIfEmpty(q.CursorID)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.UserOperationLog, 0, q.Limit)
	for rows.Next() {
		var item domain.UserOperationLog
		if err := rows.Scan(
			&item.ID, &item.ModuleID, &item.ModuleName, &item.LogType, &item.OccurredAt,
			&item.ActorUserID, &item.ActorUsername, &item.DepartmentID, &item.Description, &item.ClientIP,
			&item.ClientMAC, &item.BuildVersion, &item.RequestID, &item.Outcome, &item.ErrorCode,
		); err != nil {
			return nil, err
		}
		item.OccurredAt = item.OccurredAt.UTC()
		if item.ModuleName == "" {
			item.ModuleName = map[string]string{"module-outpatient": "门诊业务", "module-master": "系统设置"}[item.ModuleID]
		}
		if item.Description == "" {
			item.Description = domain.AuditDescription(item.LogType)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func auditNullTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC()
}
