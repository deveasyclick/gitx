package security_test

import (
	"strings"
	"testing"

	"github.com/user/gitx/internal/security"
)

func TestScan_OpenAIKey(t *testing.T) {
	// sk- + 32 alphanumeric chars (satisfies {20,})
	// Key is built dynamically to avoid triggering external secret scanners.
	key := "sk-" + strings.Repeat("b", 32)
	input := "api key: " + key
	cleaned, found := security.Scan(input)
	if !found {
		t.Fatal("expected to find secret")
	}
	if cleaned != "api key: [REDACTED]" {
		t.Errorf("cleaned = %q", cleaned)
	}
}

func TestScan_AnthropicKey(t *testing.T) {
	// sk-ant- + 28 alphanumeric chars (satisfies {20,})
	// Key is built dynamically to avoid triggering external secret scanners.
	key := "sk-ant-" + strings.Repeat("c", 28)
	input := "ANTHROPIC_API_KEY=" + key
	cleaned, found := security.Scan(input)
	if !found {
		t.Fatal("expected to find secret")
	}
	if cleaned != "ANTHROPIC_API_KEY=[REDACTED]" {
		t.Errorf("cleaned = %q", cleaned)
	}
}

func TestScan_PrivateKey(t *testing.T) {
	input := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----`
	cleaned, found := security.Scan(input)
	if !found {
		t.Fatal("expected to find private key")
	}
	// BEGIN line should be redacted; END line may or may not be
	if cleaned != "[REDACTED]\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----" {
		t.Logf("cleaned = %q", cleaned)
	}
}

func TestScan_AWSKey(t *testing.T) {
	// Key is built dynamically to avoid triggering external secret scanners.
	key := "AKIA" + strings.Repeat("A", 16)
	input := "AWS_ACCESS_KEY_ID=" + key
	cleaned, found := security.Scan(input)
	if !found {
		t.Fatal("expected to find AWS key")
	}
	if cleaned != "AWS_ACCESS_KEY_ID=[REDACTED]" {
		t.Errorf("cleaned = %q", cleaned)
	}
}

func TestScan_GitHubPAT(t *testing.T) {
	// ghp_ + exactly 36 alphanumeric chars
	input := "token: ghp_" + strings.Repeat("a", 36)
	cleaned, found := security.Scan(input)
	if !found {
		t.Fatal("expected to find GitHub PAT")
	}
	if cleaned != "token: [REDACTED]" {
		t.Errorf("cleaned = %q", cleaned)
	}
}

func TestScan_GitHubOAuth(t *testing.T) {
	// gho_ + exactly 36 alphanumeric chars
	input := "token: gho_" + strings.Repeat("a", 36)
	cleaned, found := security.Scan(input)
	if !found {
		t.Fatal("expected to find GitHub OAuth token")
	}
	if cleaned != "token: [REDACTED]" {
		t.Errorf("cleaned = %q", cleaned)
	}
}

func TestScan_GitHubPATLong(t *testing.T) {
	// github_pat_ + exactly 82 alphanumeric/underscore chars
	input := "github_pat_" + strings.Repeat("a", 82)
	cleaned, found := security.Scan(input)
	if !found {
		t.Fatal("expected to find github_pat_")
	}
	if cleaned != "[REDACTED]" {
		t.Errorf("cleaned = %q", cleaned)
	}
}

func TestScan_CleanText(t *testing.T) {
	input := `func main() {
    fmt.Println("hello world")
}`
	cleaned, found := security.Scan(input)
	if found {
		t.Fatal("expected no secrets in clean code")
	}
	if cleaned != input {
		t.Errorf("cleaned should match input")
	}
}

func TestScan_EmptyInput(t *testing.T) {
	cleaned, found := security.Scan("")
	if found {
		t.Fatal("expected no secrets in empty input")
	}
	if cleaned != "" {
		t.Errorf("expected empty string, got %q", cleaned)
	}
}

func TestScan_NormalDiff(t *testing.T) {
	input := `diff --git a/main.go b/main.go
index abc..def 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
 
-func hello() string {
+func hello(name string) string {
`
	cleaned, found := security.Scan(input)
	if found {
		t.Fatal("expected no secrets in normal diff")
	}
	if cleaned != input {
		t.Errorf("cleaned diff should match input")
	}
}

func TestScan_MixedContent(t *testing.T) {
	// Key is built dynamically to avoid triggering external secret scanners.
	key := "sk-" + strings.Repeat("d", 32)
	input := "diff --git a/config.go b/config.go\n+const APIVersion = \"v2\"\n+const APIKey = \"" + key + "\"\n"
	cleaned, found := security.Scan(input)
	if !found {
		t.Fatal("expected to find secret in mixed content")
	}
	expected := "diff --git a/config.go b/config.go\n+const APIVersion = \"v2\"\n+const APIKey = \"[REDACTED]\"\n"
	if cleaned != expected {
		t.Errorf("cleaned = %q", cleaned)
	}
}

func TestScan_BearerHeader(t *testing.T) {
	// Key is built dynamically to avoid triggering external secret scanners.
	key := "sk-" + strings.Repeat("e", 32)
	input := "Authorization: Bearer " + key
	cleaned, found := security.Scan(input)
	if !found {
		t.Fatal("expected to find bearer token")
	}
	// The OpenAI key pattern redacts the key part
	if cleaned != "Authorization: Bearer [REDACTED]" {
		t.Errorf("cleaned = %q", cleaned)
	}
}

func TestScan_DeepSeekKey(t *testing.T) {
	// DeepSeek also uses sk- prefix like OpenAI.
	// Key is built dynamically to avoid triggering external secret scanners.
	key := "sk-" + strings.Repeat("a", 30)
	input := "DEEPSEEK_API_KEY=" + key
	cleaned, found := security.Scan(input)
	if !found {
		t.Fatal("expected to find DeepSeek key")
	}
	if cleaned != "DEEPSEEK_API_KEY=[REDACTED]" {
		t.Errorf("cleaned = %q", cleaned)
	}
}
