package common

import "testing"

func TestNormalizeOpenAIFileData(t *testing.T) {
	tests := []struct {
		name             string
		filename         string
		fallbackMIMEType string
		fileData         string
		wantMIMEType     string
		wantData         string
		wantOK           bool
	}{
		{
			name:             "raw base64 uses supplied MIME",
			filename:         "report.pdf",
			fallbackMIMEType: "application/custom",
			fileData:         "SGVsbG8=",
			wantMIMEType:     "application/custom",
			wantData:         "SGVsbG8=",
			wantOK:           true,
		},
		{
			name:         "raw base64 falls back to lowercase filename extension",
			filename:     "REPORT.PDF",
			fileData:     "SGVsbG8=",
			wantMIMEType: "application/pdf",
			wantData:     "SGVsbG8=",
			wantOK:       true,
		},
		{
			name:             "data URL MIME is authoritative",
			filename:         "report.pdf",
			fallbackMIMEType: "application/custom",
			fileData:         "data:image/png;base64,iVBORw0KGgo=",
			wantMIMEType:     "image/png",
			wantData:         "iVBORw0KGgo=",
			wantOK:           true,
		},
		{
			name:         "data URL metadata is case insensitive",
			fileData:     "DATA: image/jpeg ;charset=utf-8;BASE64,not-decoded%%%",
			wantMIMEType: "image/jpeg",
			wantData:     "not-decoded%%%",
			wantOK:       true,
		},
		{name: "empty file data", filename: "report.pdf"},
		{name: "raw base64 without MIME fallback", fileData: "SGVsbG8="},
		{name: "data URL without comma", filename: "report.pdf", fileData: "data:text/plain;base64"},
		{name: "data URL without MIME", filename: "report.pdf", fileData: "data:;base64,SGVsbG8="},
		{name: "data URL without MIME or metadata", filename: "report.pdf", fileData: "data:,SGVsbG8="},
		{name: "data URL without base64 metadata", filename: "report.pdf", fileData: "data:text/plain,SGVsbG8="},
		{name: "data URL with non-base64 metadata", filename: "report.pdf", fileData: "data:text/plain;charset=utf-8,SGVsbG8="},
		{name: "data URL without payload", filename: "report.pdf", fileData: "data:text/plain;base64,"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mimeType, data, ok := NormalizeOpenAIFileData(tt.filename, tt.fallbackMIMEType, tt.fileData)
			if ok != tt.wantOK {
				t.Fatalf("NormalizeOpenAIFileData() ok = %v, want %v", ok, tt.wantOK)
			}
			if mimeType != tt.wantMIMEType {
				t.Errorf("NormalizeOpenAIFileData() MIME type = %q, want %q", mimeType, tt.wantMIMEType)
			}
			if data != tt.wantData {
				t.Errorf("NormalizeOpenAIFileData() data = %q, want %q", data, tt.wantData)
			}
		})
	}
}
