package cloudfront

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"

	appcloudfront "aws-terminal/internal/application/cloudfront"
	domaincloudfront "aws-terminal/internal/domain/cloudfront"
	"aws-terminal/internal/ui/styles"
)

type CloudFrontService interface {
	ListDistributions(ctx context.Context, profileName, region string) ([]domaincloudfront.Distribution, error)
	CreateInvalidation(ctx context.Context, input appcloudfront.CreateInvalidationInput) (domaincloudfront.Invalidation, error)
	GetInvalidation(ctx context.Context, profileName, region, distributionID, invalidationID string) (domaincloudfront.Invalidation, error)
}

type cloudFrontStage int

const (
	cloudFrontStageDistribution cloudFrontStage = iota
	cloudFrontStagePaths
	cloudFrontStageResult
)

type cloudFrontDistributionsLoadedMsg struct {
	sessionKey    string
	distributions []domaincloudfront.Distribution
	err           error
}

type cloudFrontInvalidationCreatedMsg struct {
	invalidation domaincloudfront.Invalidation
	err          error
}

type cloudFrontCopiedMsg struct {
	err error
}

type cloudFrontInvalidationPolledMsg struct {
	invalidation domaincloudfront.Invalidation
	err          error
}

func (cloudFrontDistributionsLoadedMsg) OwnerPageID() string { return "cloudfront" }
func (cloudFrontInvalidationCreatedMsg) OwnerPageID() string { return "cloudfront" }
func (cloudFrontCopiedMsg) OwnerPageID() string              { return "cloudfront" }
func (cloudFrontInvalidationPolledMsg) OwnerPageID() string  { return "cloudfront" }

type CloudFrontPage struct {
	service              CloudFrontService
	stage                cloudFrontStage
	sessionKey           string
	loadedFor            string
	loading              bool
	loadErr              string
	distributions        []domaincloudfront.Distribution
	distributionIndex    int
	selectedDistribution domaincloudfront.Distribution
	pathsInput           textinput.Model
	spinner              spinner.Model
	creating             bool
	createErr            string
	copiedMessage        string
	invalidation         *domaincloudfront.Invalidation
	loadCancel           context.CancelFunc
	createCancel         context.CancelFunc
	pollCancel           context.CancelFunc
}

func NewCloudFrontPage(service CloudFrontService) *CloudFrontPage {
	pathsInput := textinput.New()
	pathsInput.Prompt = "Paths: "
	pathsInput.Placeholder = "/*"
	pathsInput.CharLimit = 1024
	pathsInput.SetValue("/*")

	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = styles.StatusStyle

	return &CloudFrontPage{
		service:    service,
		stage:      cloudFrontStageDistribution,
		pathsInput: pathsInput,
		spinner:    spin,
	}
}

func (*CloudFrontPage) ID() string {
	return "cloudfront"
}

func (*CloudFrontPage) Title() string {
	return "CloudFront"
}

func (*CloudFrontPage) Description() string {
	return "Select a distribution and create or copy an invalidation."
}
