package s3

import (
	"path/filepath"
	"testing"

	"aws-terminal/internal/config"
	domains3 "aws-terminal/internal/domain/s3"
)

type memoryPreferenceStore struct {
	prefs config.Preferences
	saves int
}

func (s *memoryPreferenceStore) Load() (config.Preferences, error) {
	return s.prefs, nil
}

func (s *memoryPreferenceStore) Save(prefs config.Preferences) error {
	s.prefs = prefs
	s.saves++
	return nil
}

func TestNewS3PageWithPreferencesUsesStoredSourceDirectory(t *testing.T) {
	dir := t.TempDir()
	store := &memoryPreferenceStore{prefs: config.Preferences{S3SourceDirectory: dir}}
	page := NewS3PageWithPreferences(staticS3Service{}, store)

	if page.picker.CurrentDirectory != dir {
		t.Fatalf("CurrentDirectory = %q, want %q", page.picker.CurrentDirectory, dir)
	}
}

func TestS3RememberSourceStoresDirectoryForFile(t *testing.T) {
	store := &memoryPreferenceStore{}
	page := NewS3PageWithPreferences(staticS3Service{}, store)
	path := filepath.Join(t.TempDir(), "index.html")

	page.rememberSource(domains3.SourceSelection{Path: path, Kind: domains3.SourceKindFile})

	if store.prefs.S3SourceDirectory != filepath.Dir(path) {
		t.Fatalf("S3SourceDirectory = %q", store.prefs.S3SourceDirectory)
	}
}

func TestS3RememberPrefixStoresRecentPrefix(t *testing.T) {
	store := &memoryPreferenceStore{}
	page := NewS3PageWithPreferences(staticS3Service{}, store)

	page.rememberPrefix(" /site/ ")

	if len(store.prefs.S3RecentPrefixes) != 1 || store.prefs.S3RecentPrefixes[0] != "site" {
		t.Fatalf("recent prefixes = %#v", store.prefs.S3RecentPrefixes)
	}
}
