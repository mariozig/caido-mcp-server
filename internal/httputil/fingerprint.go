package httputil

import (
	"strings"
	"unicode/utf8"
)

type ContentKind string

const (
	KindJSON   ContentKind = "json"
	KindHTML   ContentKind = "html"
	KindXML    ContentKind = "xml"
	KindText   ContentKind = "text"
	KindBinary ContentKind = "binary"
	KindEmpty  ContentKind = "empty"
)

type Fingerprint struct {
	Kind        ContentKind `json:"kind"`
	ContentType string      `json:"contentType,omitempty"`
	BodySize    int         `json:"bodySize"`
}

func FingerprintFromHeaders(headers []Header, bodySize int) Fingerprint {
	fp := Fingerprint{BodySize: bodySize}

	if bodySize == 0 {
		fp.Kind = KindEmpty
		return fp
	}

	ct := headerValue(headers, "content-type")
	fp.ContentType = ct

	lower := strings.ToLower(ct)
	switch {
	case strings.Contains(lower, "json"):
		fp.Kind = KindJSON
	case strings.Contains(lower, "html"):
		fp.Kind = KindHTML
	case strings.Contains(lower, "xml"):
		fp.Kind = KindXML
	case strings.Contains(lower, "text"):
		fp.Kind = KindText
	case strings.Contains(lower, "javascript"):
		fp.Kind = KindText
	case strings.Contains(lower, "image"),
		strings.Contains(lower, "audio"),
		strings.Contains(lower, "video"),
		strings.Contains(lower, "octet-stream"),
		strings.Contains(lower, "font"),
		strings.Contains(lower, "woff"),
		strings.Contains(lower, "pdf"):
		fp.Kind = KindBinary
	default:
		fp.Kind = KindText
	}

	return fp
}

func FingerprintFromBody(body []byte) Fingerprint {
	fp := Fingerprint{BodySize: len(body)}

	if len(body) == 0 {
		fp.Kind = KindEmpty
		return fp
	}

	if !utf8.Valid(body) {
		fp.Kind = KindBinary
		return fp
	}

	trimmed := strings.TrimSpace(string(body[:min(len(body), 256)]))
	switch {
	case strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "["):
		fp.Kind = KindJSON
	case strings.HasPrefix(strings.ToLower(trimmed), "<!doctype") ||
		strings.HasPrefix(strings.ToLower(trimmed), "<html"):
		fp.Kind = KindHTML
	case strings.HasPrefix(trimmed, "<?xml") || strings.HasPrefix(trimmed, "<"):
		fp.Kind = KindXML
	default:
		fp.Kind = KindText
	}

	return fp
}

func AdaptiveBodyLimit(fp Fingerprint, requestedLimit int) int {
	if requestedLimit > 0 {
		return requestedLimit
	}
	switch fp.Kind {
	case KindJSON:
		return 4000
	case KindHTML:
		return 3000
	case KindXML:
		return 3000
	case KindBinary:
		return 200
	case KindEmpty:
		return 0
	default:
		return DefaultBodyLimit
	}
}

func headerValue(headers []Header, name string) string {
	lower := strings.ToLower(name)
	for _, h := range headers {
		if strings.ToLower(h.Name) == lower {
			return h.Value
		}
	}
	return ""
}
