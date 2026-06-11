package ecr

import "time"

type Repository struct {
	Name        string
	URI         string
	RegistryID  string
	Region      string
	CreatedAt   time.Time
	ImageCount  int
	ScanOnPush  bool
	MutableTags bool
}

type RepositoryImage struct {
	RepositoryName     string
	Digest             string
	Tags               []string
	SizeBytes          int64
	PushedAt           time.Time
	LastRecordedPullAt time.Time
}

type LocalImage struct {
	ID         string
	Repository string
	Tag        string
	Reference  string
	SizeBytes  int64
	CreatedAt  time.Time
}

type CreateRepositoryInput struct {
	Profile string
	Region  string
	Name    string
}

type AuthorizationToken struct {
	Username      string
	Password      string
	ProxyEndpoint string
	ExpiresAt     time.Time
}

type PushPlan struct {
	Profile          string
	Region           string
	RepositoryName   string
	RepositoryURI    string
	SourceImage      string
	DestinationTag   string
	DestinationImage string
}

type PushProgress struct {
	Status  string
	ID      string
	Detail  string
	Current int64
	Total   int64
	Error   string
}

type PushResult struct {
	SourceImage      string
	DestinationImage string
	RepositoryName   string
	Tag              string
	Digest           string
	CompletedAt      time.Time
}
