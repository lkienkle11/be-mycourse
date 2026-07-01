package infra

import (
	"archive/zip"
	"bytes"
	"testing"

	"mycourse-io-be/internal/shared/constants"
)

func TestMIMEForUploadRouting_ignoresClientMIME(t *testing.T) {
	payload := []byte("%PDF-1.4 test")
	got := MIMEForUploadRouting(payload, "report.pdf", "text/html")
	if got != "application/pdf" {
		t.Fatalf("MIME = %q, want application/pdf", got)
	}
}

func TestMIMEForUploadRouting_blocksHTML(t *testing.T) {
	payload := []byte("<!DOCTYPE html><html><body>x</body></html>")
	got := MIMEForUploadRouting(payload, "page.html", "text/html")
	if got != "" {
		t.Fatalf("MIME = %q, want empty for blocked routing MIME", got)
	}
}

func TestCanonicalStorageMIME_ignoresClientMIMEForPDF(t *testing.T) {
	payload := []byte("%PDF-1.4 test")
	got := CanonicalStorageMIME(payload, "report.pdf", "text/html", constants.FileKindFile)
	if got != "application/pdf" {
		t.Fatalf("MIME = %q, want application/pdf", got)
	}
}

func TestCanonicalStorageMIME_rejectsHTMLPayloadWithPDFExtension(t *testing.T) {
	payload := []byte("<!DOCTYPE html><html><body>x</body></html>")
	got := CanonicalStorageMIME(payload, "report.pdf", "application/pdf", constants.FileKindFile)
	if got != "application/octet-stream" {
		t.Fatalf("MIME = %q, want application/octet-stream", got)
	}
}

func TestCanonicalStorageMIME_webpMagicBytes(t *testing.T) {
	payload := make([]byte, 12)
	copy(payload[0:4], "RIFF")
	copy(payload[8:12], "WEBP")
	got := CanonicalStorageMIME(payload, "photo.webp", "image/png", constants.FileKindFile)
	if got != "image/webp" {
		t.Fatalf("MIME = %q, want image/webp", got)
	}
}

func TestCanonicalStorageMIME_blocksSVG(t *testing.T) {
	payload := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	got := CanonicalStorageMIME(payload, "icon.svg", "image/svg+xml", constants.FileKindFile)
	if got != "application/octet-stream" {
		t.Fatalf("MIME = %q, want application/octet-stream", got)
	}
}

func TestCanonicalStorageMIME_plainText(t *testing.T) {
	payload := []byte("hello world\n")
	got := CanonicalStorageMIME(payload, "notes.txt", "text/html", constants.FileKindFile)
	if got != "text/plain" {
		t.Fatalf("MIME = %q, want text/plain", got)
	}
}

func TestCanonicalStorageMIME_docx(t *testing.T) {
	got := CanonicalStorageMIME(minimalOOXMLPayload("word/document.xml"), "report.docx", "application/zip", constants.FileKindFile)
	want := "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	if got != want {
		t.Fatalf("MIME = %q, want %q", got, want)
	}
}

func TestCanonicalStorageMIME_xlsx(t *testing.T) {
	got := CanonicalStorageMIME(minimalOOXMLPayload("xl/workbook.xml"), "sheet.xlsx", "application/zip", constants.FileKindFile)
	want := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if got != want {
		t.Fatalf("MIME = %q, want %q", got, want)
	}
}

func TestCanonicalStorageMIME_pptx(t *testing.T) {
	got := CanonicalStorageMIME(minimalOOXMLPayload("ppt/presentation.xml"), "deck.pptx", "application/zip", constants.FileKindFile)
	want := "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	if got != want {
		t.Fatalf("MIME = %q, want %q", got, want)
	}
}

func TestCanonicalStorageMIME_zip(t *testing.T) {
	got := CanonicalStorageMIME(minimalZipPayload("data.bin"), "archive.zip", "application/octet-stream", constants.FileKindFile)
	if got != "application/zip" {
		t.Fatalf("MIME = %q, want application/zip", got)
	}
}

func TestCanonicalStorageMIME_zipRenamedAsDocx(t *testing.T) {
	got := CanonicalStorageMIME(minimalZipPayload("data.bin"), "fake.docx", "application/zip", constants.FileKindFile)
	if got != "application/octet-stream" {
		t.Fatalf("MIME = %q, want application/octet-stream", got)
	}
}

func TestCanonicalStorageMIME_mp4VideoKind(t *testing.T) {
	payload := []byte{0, 0, 0, 0x18, 'f', 't', 'y', 'p', 'm', 'p', '4', '2', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	got := CanonicalStorageMIME(payload, "clip.mp4", "application/octet-stream", constants.FileKindVideo)
	if got != "video/mp4" {
		t.Fatalf("MIME = %q, want video/mp4", got)
	}
}

func TestIsWebPPayload(t *testing.T) {
	valid := bytes.Join([][]byte{[]byte("RIFF"), make([]byte, 4), []byte("WEBP")}, nil)
	if !isWebPPayload(valid) {
		t.Fatal("expected valid WebP payload")
	}
	if isWebPPayload([]byte("not-webp")) {
		t.Fatal("expected invalid payload")
	}
}

func minimalOOXMLPayload(entry string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	_, _ = w.Create(entry)
	_, _ = w.Create("[Content_Types].xml")
	_ = w.Close()
	return buf.Bytes()
}

func minimalZipPayload(entry string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	_, _ = w.Create(entry)
	_ = w.Close()
	return buf.Bytes()
}
