package winquote

import "testing"

func TestQuotePlain(t *testing.T) {
	if got := QuoteArg(`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`); got != `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe` {
		t.Fatalf("got %q", got)
	}
}

func TestQuoteSpace(t *testing.T) {
	got := QuoteArg(`C:\Program Files\PowerShell\7\pwsh.exe`)
	want := `"C:\Program Files\PowerShell\7\pwsh.exe"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCommandLine(t *testing.T) {
	got := CommandLine([]string{
		`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
		"-NoLogo",
		"-NoExit",
	})
	want := `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -NoLogo -NoExit`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
