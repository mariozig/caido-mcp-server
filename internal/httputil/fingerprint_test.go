package httputil

import (
	"testing"
)

func TestFingerprintFromHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  []Header
		bodySize int
		wantKind ContentKind
	}{
		{
			name:     "json content type",
			headers:  []Header{{Name: "Content-Type", Value: "application/json; charset=utf-8"}},
			bodySize: 500,
			wantKind: KindJSON,
		},
		{
			name:     "html content type",
			headers:  []Header{{Name: "Content-Type", Value: "text/html"}},
			bodySize: 1000,
			wantKind: KindHTML,
		},
		{
			name:     "binary image",
			headers:  []Header{{Name: "Content-Type", Value: "image/png"}},
			bodySize: 50000,
			wantKind: KindBinary,
		},
		{
			name:     "pdf binary",
			headers:  []Header{{Name: "Content-Type", Value: "application/pdf"}},
			bodySize: 100000,
			wantKind: KindBinary,
		},
		{
			name:     "empty body",
			headers:  []Header{{Name: "Content-Type", Value: "text/html"}},
			bodySize: 0,
			wantKind: KindEmpty,
		},
		{
			name:     "xml content type",
			headers:  []Header{{Name: "Content-Type", Value: "application/xml"}},
			bodySize: 200,
			wantKind: KindXML,
		},
		{
			name:     "javascript",
			headers:  []Header{{Name: "Content-Type", Value: "application/javascript"}},
			bodySize: 300,
			wantKind: KindText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := FingerprintFromHeaders(tt.headers, tt.bodySize)
			if fp.Kind != tt.wantKind {
				t.Fatalf("want kind %q, got %q", tt.wantKind, fp.Kind)
			}
		})
	}
}

func TestFingerprintFromBody(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		wantKind ContentKind
	}{
		{
			name:     "json object",
			body:     []byte(`{"status": "ok"}`),
			wantKind: KindJSON,
		},
		{
			name:     "json array",
			body:     []byte(`[{"id": 1}]`),
			wantKind: KindJSON,
		},
		{
			name:     "html document",
			body:     []byte(`<!DOCTYPE html><html><body>hi</body></html>`),
			wantKind: KindHTML,
		},
		{
			name:     "xml document",
			body:     []byte(`<?xml version="1.0"?><root/>`),
			wantKind: KindXML,
		},
		{
			name:     "binary data",
			body:     []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10},
			wantKind: KindBinary,
		},
		{
			name:     "plain text",
			body:     []byte("hello world"),
			wantKind: KindText,
		},
		{
			name:     "empty",
			body:     []byte{},
			wantKind: KindEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := FingerprintFromBody(tt.body)
			if fp.Kind != tt.wantKind {
				t.Fatalf("want kind %q, got %q", tt.wantKind, fp.Kind)
			}
		})
	}
}

func TestAdaptiveBodyLimit(t *testing.T) {
	tests := []struct {
		name      string
		kind      ContentKind
		requested int
		want      int
	}{
		{"json default", KindJSON, 0, 4000},
		{"html default", KindHTML, 0, 3000},
		{"binary default", KindBinary, 0, 200},
		{"empty", KindEmpty, 0, 0},
		{"text default", KindText, 0, DefaultBodyLimit},
		{"explicit override", KindJSON, 1000, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := Fingerprint{Kind: tt.kind}
			got := AdaptiveBodyLimit(fp, tt.requested)
			if got != tt.want {
				t.Fatalf("want %d, got %d", tt.want, got)
			}
		})
	}
}
