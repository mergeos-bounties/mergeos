package core

import (
	"strings"
	"testing"
)

func TestSafeAttachmentDownloadNameStripsHeaderUnsafeCharacters(t *testing.T) {
	name := safeAttachmentDownloadName("report\"\r\nX-Bad: yes\\final/.pdf")

	if strings.ContainsAny(name, "\"\r\n\\/\t") {
		t.Fatalf("download filename still has header-unsafe characters: %q", name)
	}
	if !strings.Contains(name, "report") || !strings.Contains(name, "final") {
		t.Fatalf("download filename lost expected readable parts: %q", name)
	}
}

func TestSafeAttachmentDownloadNameUsesFallback(t *testing.T) {
	if got := safeAttachmentDownloadName("\r\n\t"); got != "attachment" {
		t.Fatalf("download filename fallback = %q, want attachment", got)
	}
}
