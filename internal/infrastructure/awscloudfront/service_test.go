package awscloudfront

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"

	appcloudfront "aws-terminal/internal/application/cloudfront"
)

func TestCreateInvalidationRequestMapsInput(t *testing.T) {
	request := createInvalidationRequest(appcloudfront.CreateInvalidationInput{
		DistributionID: " DIST123 ",
		Paths:          []string{"/index.html", "/assets/*"},
	}, " caller-ref ")

	if got := aws.ToString(request.DistributionId); got != "DIST123" {
		t.Fatalf("DistributionId = %q", got)
	}
	if got := aws.ToString(request.InvalidationBatch.CallerReference); got != "caller-ref" {
		t.Fatalf("CallerReference = %q", got)
	}
	if got := aws.ToInt32(request.InvalidationBatch.Paths.Quantity); got != 2 {
		t.Fatalf("Quantity = %d, want 2", got)
	}
	if got := request.InvalidationBatch.Paths.Items; !reflect.DeepEqual(got, []string{"/index.html", "/assets/*"}) {
		t.Fatalf("Items = %#v", got)
	}
}

func TestCreateInvalidationRequestCopiesPaths(t *testing.T) {
	paths := []string{"/index.html"}
	request := createInvalidationRequest(appcloudfront.CreateInvalidationInput{DistributionID: "DIST123", Paths: paths}, "caller")
	paths[0] = "/mutated"

	if got := request.InvalidationBatch.Paths.Items[0]; got != "/index.html" {
		t.Fatalf("request path mutated to %q", got)
	}
}

func TestGetInvalidationRequestMapsInput(t *testing.T) {
	request := getInvalidationRequest(" DIST123 ", " INV123 ")

	if got := aws.ToString(request.DistributionId); got != "DIST123" {
		t.Fatalf("DistributionId = %q", got)
	}
	if got := aws.ToString(request.Id); got != "INV123" {
		t.Fatalf("Id = %q", got)
	}
}
