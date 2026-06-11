package region

import "strings"

type Region struct {
	ID   string
	Name string
}

func (r Region) DisplayName() string {
	if strings.TrimSpace(r.Name) == "" {
		return r.ID
	}

	return r.ID + " — " + r.Name
}

func DefaultCatalog() []Region {
	return []Region{
		{ID: "af-south-1", Name: "Africa (Cape Town)"},
		{ID: "ap-east-1", Name: "Asia Pacific (Hong Kong)"},
		{ID: "ap-northeast-1", Name: "Asia Pacific (Tokyo)"},
		{ID: "ap-northeast-2", Name: "Asia Pacific (Seoul)"},
		{ID: "ap-northeast-3", Name: "Asia Pacific (Osaka)"},
		{ID: "ap-south-1", Name: "Asia Pacific (Mumbai)"},
		{ID: "ap-south-2", Name: "Asia Pacific (Hyderabad)"},
		{ID: "ap-southeast-1", Name: "Asia Pacific (Singapore)"},
		{ID: "ap-southeast-2", Name: "Asia Pacific (Sydney)"},
		{ID: "ap-southeast-3", Name: "Asia Pacific (Jakarta)"},
		{ID: "ap-southeast-4", Name: "Asia Pacific (Melbourne)"},
		{ID: "ca-central-1", Name: "Canada (Central)"},
		{ID: "ca-west-1", Name: "Canada West (Calgary)"},
		{ID: "cn-north-1", Name: "China (Beijing)"},
		{ID: "cn-northwest-1", Name: "China (Ningxia)"},
		{ID: "eu-central-1", Name: "Europe (Frankfurt)"},
		{ID: "eu-central-2", Name: "Europe (Zurich)"},
		{ID: "eu-north-1", Name: "Europe (Stockholm)"},
		{ID: "eu-south-1", Name: "Europe (Milan)"},
		{ID: "eu-south-2", Name: "Europe (Spain)"},
		{ID: "eu-west-1", Name: "Europe (Ireland)"},
		{ID: "eu-west-2", Name: "Europe (London)"},
		{ID: "eu-west-3", Name: "Europe (Paris)"},
		{ID: "il-central-1", Name: "Israel (Tel Aviv)"},
		{ID: "me-central-1", Name: "Middle East (UAE)"},
		{ID: "me-south-1", Name: "Middle East (Bahrain)"},
		{ID: "mx-central-1", Name: "Mexico (Central)"},
		{ID: "sa-east-1", Name: "South America (São Paulo)"},
		{ID: "us-east-1", Name: "US East (N. Virginia)"},
		{ID: "us-east-2", Name: "US East (Ohio)"},
		{ID: "us-gov-east-1", Name: "AWS GovCloud (US-East)"},
		{ID: "us-gov-west-1", Name: "AWS GovCloud (US-West)"},
		{ID: "us-west-1", Name: "US West (N. California)"},
		{ID: "us-west-2", Name: "US West (Oregon)"},
	}
}
