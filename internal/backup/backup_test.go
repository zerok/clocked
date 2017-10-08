package backup_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"github.com/zerok/clocked/internal/backup"
)

const sourceFolder = "sample-source"

func resetSourceFolder(t *testing.T) {
	os.RemoveAll(sourceFolder)
	os.MkdirAll(sourceFolder, 0700)
	if err := ioutil.WriteFile(filepath.Join(sourceFolder, "test.txt"), []byte("hello world"), 0600); err != nil {
		t.Fatalf("failed creating a sample file: %s", err.Error())
	}
}

func removeBackupFolder(t *testing.T) {
	if err := os.RemoveAll(fmt.Sprintf("%s_backups", sourceFolder)); err != nil {
		t.Fatalf("Failed to remove backups folder: %s", err.Error())
	}
}

func createTestRepo(t *testing.T) *backup.Backup {
	b, err := backup.New(&backup.Options{
		SourcePath: sourceFolder,
	})
	if err != nil {
		t.Fatalf("Creating a sample repo should have worked: %s", err.Error())
		return nil
	}
	if !b.Available() {
		t.Log("restic seems not to be available. Skipping.")
		t.SkipNow()
	}
	if err := b.Init(); err != nil {
		t.Fatalf("Initializing the sample repo should not have caused an error: %s", err.Error())
		return nil
	}
	return b
}

func TestBackupInit(t *testing.T) {
	removeBackupFolder(t)
	resetSourceFolder(t)
	opts := backup.Options{}
	_, err := backup.New(&opts)

	// Only the source path is a required option. Everything else will be
	// automatically generated.
	require.Error(t, err, "the source path is a required setting")

	opts.SourcePath = sourceFolder
	b, err := backup.New(&opts)
	require.NoError(t, err, "only SourcePath should have been a mandatory setting")
	require.NotNil(t, b, "a backup instance should have been returned")

	if !b.Available() {
		t.Log("Restic seems not to be available. Skipping these tests.")
		t.Skip("restic is not available")
	}

	// If no PasswordFile is specified, a random password should be generated.
	require.NoError(t, b.Init(), "init should have completed without an error")

	require.Condition(t, fileExists(filepath.Join(sourceFolder, "backups.passwd")), "no random password file was generated")
}

func TestSnapshotCreation(t *testing.T) {
	removeBackupFolder(t)
	resetSourceFolder(t)
	b := createTestRepo(t)

	require.NoError(t, b.CreateSnapshot(), "creating a snapshot should not have caused an error")

	snapshots, err := b.Snapshots()
	require.NoError(t, err, "snapshots should not have returned an error")
	require.Len(t, snapshots, 1, "one snapshot should have been returned")
}

func fileExists(path string) assert.Comparison {
	return func() bool {
		if _, err := os.Stat(path); err != nil {
			return false
		}
		return true
	}
}
