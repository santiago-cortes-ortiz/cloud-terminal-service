package app

import (
	"aws-terminal/internal/config"
	"aws-terminal/internal/ui/pages"
	cloudfrontpage "aws-terminal/internal/ui/pages/cloudfront"
	ecrpage "aws-terminal/internal/ui/pages/ecr"
	s3page "aws-terminal/internal/ui/pages/s3"
)

func DefaultPages(s3Service s3page.S3Service, cloudFrontService cloudfrontpage.CloudFrontService, ecrService ecrpage.ECRService, preferenceStore config.PreferenceStore) []pages.Page {
	return []pages.Page{
		pages.NewDashboardPage(),
		s3page.NewS3PageWithPreferences(s3Service, preferenceStore),
		cloudfrontpage.NewCloudFrontPage(cloudFrontService),
		ecrpage.NewECRPage(ecrService),
	}
}
