package app

import (
	"testing"
	"time"
)

func TestDashboardWindowUsesShanghaiCalendarBoundaries(t *testing.T) {
	now := time.Date(2026, 10, 14, 16, 30, 0, 0, time.UTC)
	tests := []struct {
		period DashboardPeriod
		start  string
		end    string
	}{
		{DashboardToday, "2026-10-15T00:00:00+08:00", "2026-10-16T00:00:00+08:00"},
		{DashboardYesterday, "2026-10-14T00:00:00+08:00", "2026-10-15T00:00:00+08:00"},
		{DashboardWeek, "2026-10-12T00:00:00+08:00", "2026-10-19T00:00:00+08:00"},
		{DashboardMonth, "2026-10-01T00:00:00+08:00", "2026-11-01T00:00:00+08:00"},
	}
	for _, test := range tests {
		start, end, ok := dashboardWindow(now, test.period)
		if !ok || start.Format(time.RFC3339) != test.start || end.Format(time.RFC3339) != test.end {
			t.Errorf("%s range = %s to %s (ok=%t)", test.period, start.Format(time.RFC3339), end.Format(time.RFC3339), ok)
		}
	}
	if _, _, ok := dashboardWindow(now, "CUSTOM"); ok {
		t.Fatal("custom period should be rejected until date selection is implemented")
	}
}
