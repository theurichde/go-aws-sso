package sso

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"runtime"
	"testing"
)

func TestProcessCredentialProcessTemplateBotocoreOnWindows(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		python, err = exec.LookPath("python")
		if err != nil {
			t.Skip("install Python and botocore to run this integration test")
		}
	}
	if err := exec.Command(python, "-c", "from botocore.compat import _windows_shell_split").Run(); err != nil {
		t.Skip("botocore's Windows parser is unavailable in the Python environment")
	}
	exeName := `C:\Program Files\go-aws-sso\go-aws-sso.exe`
	configPath := `C:\Users\test user\SSO configs\portal.yaml`
	template := processCredentialProcessTemplateWithExeName(
		exeName, "accountid", "rolename", "eu-central-1", "portal profile", configPath,
	)
	platform := runtime.GOOS
	if platform == "windows" {
		platform = "win32"
	}
	// Botocore uses _windows_shell_split on Windows and shlex.split elsewhere.
	const test_script = "import json, sys; from botocore.compat import compat_shell_split; " +
		"print(json.dumps(compat_shell_split(sys.argv[1], platform=sys.argv[2])))"
	output, err := exec.Command(python, "-c", test_script, template.CredentialProcess, platform).CombinedOutput()
	if err != nil {
		t.Fatalf("botocore parser failed: %v\n%s", err, output)
	}
	var got []string
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("decoding botocore output: %v", err)
	}
	want := []string{
		exeName, "--config-path", configPath,
		"assume", "-q", "-a", "accountid", "-n", "rolename", "-p", "portal profile",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("botocore arguments = %q, want %q", got, want)
	}
}
