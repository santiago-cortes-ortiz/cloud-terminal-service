package ecr

import (
	"context"
	"reflect"
	"testing"

	domainecr "aws-terminal/internal/domain/ecr"
)

type fakeRepositoryAPI struct {
	repositories []domainecr.Repository
	images       []domainecr.RepositoryImage
	auth         domainecr.AuthorizationToken
}

func (f *fakeRepositoryAPI) ListRepositories(ctx context.Context, profileName, region string) ([]domainecr.Repository, error) {
	return append([]domainecr.Repository(nil), f.repositories...), nil
}
func (f *fakeRepositoryAPI) CreateRepository(ctx context.Context, input domainecr.CreateRepositoryInput) (domainecr.Repository, error) {
	return domainecr.Repository{Name: input.Name, URI: "123.dkr.ecr." + input.Region + ".amazonaws.com/" + input.Name}, nil
}
func (f *fakeRepositoryAPI) ListImages(ctx context.Context, profileName, region, repositoryName string) ([]domainecr.RepositoryImage, error) {
	return append([]domainecr.RepositoryImage(nil), f.images...), nil
}
func (f *fakeRepositoryAPI) GetAuthorizationToken(ctx context.Context, profileName, region string) (domainecr.AuthorizationToken, error) {
	return f.auth, nil
}

type fakeDockerAPI struct{ images []domainecr.LocalImage }

func (f *fakeDockerAPI) ListLocalImages(ctx context.Context) ([]domainecr.LocalImage, error) {
	return append([]domainecr.LocalImage(nil), f.images...), nil
}
func (f *fakeDockerAPI) Login(ctx context.Context, auth domainecr.AuthorizationToken) error {
	return nil
}
func (f *fakeDockerAPI) TagImage(ctx context.Context, sourceImage, destinationImage string) error {
	return nil
}
func (f *fakeDockerAPI) PushImage(ctx context.Context, destinationImage string, progress chan<- domainecr.PushProgress) (string, error) {
	return "sha256:abc", nil
}

func TestListRepositoriesSortsByName(t *testing.T) {
	svc := NewService(&fakeRepositoryAPI{repositories: []domainecr.Repository{{Name: "z"}, {Name: "a"}, {Name: "m"}}}, &fakeDockerAPI{})
	repos, err := svc.ListRepositories(context.Background(), "pre", "eu-west-1")
	if err != nil {
		t.Fatal(err)
	}
	got := []string{repos[0].Name, repos[1].Name, repos[2].Name}
	if !reflect.DeepEqual(got, []string{"a", "m", "z"}) {
		t.Fatalf("unexpected sort: %v", got)
	}
}

func TestBuildPushPlanDefaultsAndValidates(t *testing.T) {
	svc := NewService(&fakeRepositoryAPI{}, &fakeDockerAPI{})
	plan, err := svc.BuildPushPlan(BuildPushPlanInput{Profile: "pre", Region: "eu-west-1", RepositoryName: "app/api", RepositoryURI: "123.dkr.ecr.eu-west-1.amazonaws.com/app/api", SourceImage: "local/app:v1"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.DestinationTag != "v1" || plan.DestinationImage != "123.dkr.ecr.eu-west-1.amazonaws.com/app/api:v1" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if _, err := svc.BuildPushPlan(BuildPushPlanInput{Profile: "pre", Region: "eu-west-1", RepositoryName: "Bad/Name", RepositoryURI: "uri", SourceImage: "img", DestinationTag: "bad tag"}); err == nil {
		t.Fatal("expected invalid repository/tag error")
	}
}

func TestSearchRepositoriesFiltersNameAndURI(t *testing.T) {
	svc := NewService(&fakeRepositoryAPI{repositories: []domainecr.Repository{{Name: "api", URI: "111.dkr/repo"}, {Name: "web", URI: "111.dkr/frontend"}}}, &fakeDockerAPI{})
	repos, err := svc.SearchRepositories(context.Background(), "pre", "us-east-1", "front")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Name != "web" {
		t.Fatalf("unexpected repos: %#v", repos)
	}
}
