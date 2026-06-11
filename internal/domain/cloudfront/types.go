package cloudfront

import "time"

type Distribution struct {
	ID         string
	DomainName string
	Comment    string
	Aliases    []string
	Enabled    bool
}

type Invalidation struct {
	ID             string
	Status         string
	DistributionID string
	Paths          []string
	CreatedAt      time.Time
}
