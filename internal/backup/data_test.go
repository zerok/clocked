package backup_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zerok/clocked/internal/backup"
)

func TestSnapshotTime(t *testing.T) {
	s := backup.Snapshot{
		RawTime: "2017-10-08T20:29:36.305061148+02:00",
	}
	_, err := s.Time()
	require.NoError(t, err, "no error should have been returned")
}
