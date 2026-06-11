package awsclients

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestNormalizeRegionDefaultsEmptyRegion(t *testing.T) {
	if got := NormalizeRegion("  "); got != DefaultRegion {
		t.Fatalf("expected default region %q, got %q", DefaultRegion, got)
	}
}

func TestNormalizeRegionTrimsRegion(t *testing.T) {
	if got := NormalizeRegion(" eu-west-1 "); got != "eu-west-1" {
		t.Fatalf("expected trimmed region, got %q", got)
	}
}

func TestCacheKeyTrimsProfileAndDefaultsRegion(t *testing.T) {
	if got := CacheKey(" prod ", ""); got != "prod|"+DefaultRegion {
		t.Fatalf("unexpected cache key %q", got)
	}
}

func TestNewFactoryWithOptionsAppliesDefaults(t *testing.T) {
	factory := NewFactoryWithOptions(Options{})
	opts := factory.Options()

	if opts.OperationTimeout != DefaultOperationTimeout {
		t.Fatalf("OperationTimeout = %s, want %s", opts.OperationTimeout, DefaultOperationTimeout)
	}
	if opts.UploadTimeout != DefaultUploadTimeout {
		t.Fatalf("UploadTimeout = %s, want %s", opts.UploadTimeout, DefaultUploadTimeout)
	}
	if opts.RetryMaxAttempts != DefaultRetryMaxAttempts {
		t.Fatalf("RetryMaxAttempts = %d, want %d", opts.RetryMaxAttempts, DefaultRetryMaxAttempts)
	}
	if opts.RetryMode != aws.RetryModeStandard {
		t.Fatalf("RetryMode = %q, want %q", opts.RetryMode, aws.RetryModeStandard)
	}
	if opts.AppID != DefaultAppID {
		t.Fatalf("AppID = %q, want %q", opts.AppID, DefaultAppID)
	}
}

func TestNewFactoryWithOptionsKeepsConfiguredValues(t *testing.T) {
	factory := NewFactoryWithOptions(Options{
		OperationTimeout: 5 * time.Second,
		UploadTimeout:    time.Hour,
		RetryMaxAttempts: 5,
		RetryMode:        aws.RetryModeAdaptive,
		AppID:            " custom-app ",
	})
	opts := factory.Options()

	if opts.OperationTimeout != 5*time.Second {
		t.Fatalf("OperationTimeout = %s", opts.OperationTimeout)
	}
	if opts.UploadTimeout != time.Hour {
		t.Fatalf("UploadTimeout = %s", opts.UploadTimeout)
	}
	if opts.RetryMaxAttempts != 5 {
		t.Fatalf("RetryMaxAttempts = %d", opts.RetryMaxAttempts)
	}
	if opts.RetryMode != aws.RetryModeAdaptive {
		t.Fatalf("RetryMode = %q", opts.RetryMode)
	}
	if opts.AppID != "custom-app" {
		t.Fatalf("AppID = %q", opts.AppID)
	}
}

func TestWithTimeoutReturnsDeadlineWhenConfigured(t *testing.T) {
	ctx, cancel := WithTimeout(context.Background(), time.Minute)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline")
	}
	if time.Until(deadline) <= 0 {
		t.Fatalf("expected future deadline, got %s", deadline)
	}
}

func TestWithTimeoutLeavesContextWithoutDeadlineWhenDisabled(t *testing.T) {
	ctx, cancel := WithTimeout(context.Background(), 0)
	defer cancel()

	if _, ok := ctx.Deadline(); ok {
		t.Fatal("did not expect deadline")
	}
}
