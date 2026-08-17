package updater

import "testing"

func TestVersionNormalizeLeadingV(t *testing.T) {
	if got, err := normalizeVersion("v1.2.3"); err != nil || got != "v1.2.3" {
		t.Fatalf("normalizeVersion(v1.2.3) = %q, %v", got, err)
	}
}

func TestVersionNormalizeBare(t *testing.T) {
	if got, err := normalizeVersion("1.2.3"); err != nil || got != "v1.2.3" {
		t.Fatalf("normalizeVersion(1.2.3) = %q, %v", got, err)
	}
}

func TestVersionRejectsMalformed(t *testing.T) {
	for _, bad := range []string{"", "1.2", "abc", "v1.2.3.4", "v1.2.3.4.5"} {
		if _, err := normalizeVersion(bad); err == nil {
			t.Fatalf("normalizeVersion(%q) accepted, want error", bad)
		}
	}
}

func TestVersionAllowsPrereleaseCurrent(t *testing.T) {
	if got, err := normalizeVersion("v1.3.0-rc1"); err != nil || got != "v1.3.0-rc1" {
		t.Fatalf("normalizeVersion(v1.3.0-rc1) = %q, %v", got, err)
	}
}

func TestVersionRejectsPrereleaseStable(t *testing.T) {
	if _, err := normalizeStable("v1.3.0-rc1"); err == nil {
		t.Fatal("normalizeStable accepted a prerelease tag")
	}
}

func TestVersionEvaluateUpdateAvailable(t *testing.T) {
	got, err := Evaluate("v1.2.0", "v1.3.0")
	if err != nil || got != DecisionUpdateAvailable {
		t.Fatalf("Evaluate = %q, %v; want update-available", got, err)
	}
}

func TestVersionEvaluateSame(t *testing.T) {
	got, err := Evaluate("1.3.0", "v1.3.0")
	if err != nil || got != DecisionUpToDate {
		t.Fatalf("Evaluate = %q, %v; want up-to-date", got, err)
	}
}

func TestVersionEvaluateDowngrade(t *testing.T) {
	got, err := Evaluate("v2.0.0", "v1.3.0")
	if err != nil || got != DecisionDowngradeRefused {
		t.Fatalf("Evaluate = %q, %v; want downgrade-refused", got, err)
	}
}

func TestVersionEvaluateDevelopment(t *testing.T) {
	for _, dev := range []string{"dev", "", "latest"} {
		got, err := Evaluate(dev, "v1.3.0")
		if err != nil || got != DecisionDevelopmentBuild {
			t.Fatalf("Evaluate(%q) = %q, %v; want development-build", dev, got, err)
		}
	}
}

func TestVersionEvaluateMalformedTag(t *testing.T) {
	if _, err := Evaluate("v1.3.0", "not-a-tag"); err == nil {
		t.Fatal("Evaluate accepted a malformed release tag")
	}
}
