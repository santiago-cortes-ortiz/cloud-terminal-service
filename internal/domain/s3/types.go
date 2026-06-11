package s3

import "time"

type Bucket struct {
	Name         string
	CreationDate time.Time
}

type SourceKind string

const (
	SourceKindFile      SourceKind = "file"
	SourceKindDirectory SourceKind = "directory"
)

type SourceFile struct {
	LocalPath      string
	DestinationKey string
	Size           int64
}

type SourceSelection struct {
	Path      string
	Kind      SourceKind
	Files     []SourceFile
	TotalSize int64
}

func (s SourceSelection) FileCount() int {
	return len(s.Files)
}

type UploadPlanningMode string

const (
	UploadPlanningModeFullRefresh UploadPlanningMode = "full-refresh"
	UploadPlanningModeSizeOnly    UploadPlanningMode = "size-only"
)

type RemoteObject struct {
	Key  string
	Size int64
}

type UploadMetadata struct {
	CacheControl    string
	ContentEncoding string
}

type SyncUpload struct {
	LocalPath string
	Key       string
	Size      int64
	Metadata  UploadMetadata
}

type SyncSkip struct {
	LocalPath string
	Key       string
	Size      int64
}

type SyncDelete struct {
	Key  string
	Size int64
}

type SyncPlan struct {
	Profile             string
	Region              string
	Bucket              string
	Prefix              string
	Source              SourceSelection
	DeleteEnabled       bool
	UploadPlanningMode  UploadPlanningMode
	StaticWebsitePreset bool
	Uploads             []SyncUpload
	Skips               []SyncSkip
	Deletes             []SyncDelete
}

func (p SyncPlan) UploadCount() int {
	return len(p.Uploads)
}

func (p SyncPlan) SkipCount() int {
	return len(p.Skips)
}

func (p SyncPlan) DeleteCount() int {
	return len(p.Deletes)
}

type SyncProgress struct {
	Stage            string
	Current          int
	Total            int
	Detail           string
	Uploaded         int
	Deleted          int
	Skipped          int
	UploadedBytes    int64
	TotalUploadBytes int64
}

type SyncResult struct {
	Uploaded int
	Deleted  int
	Skipped  int
}
