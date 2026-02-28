package service

import (
	"net/http"
	"testing"
)

func TestParseSubscriptionUsageMeta(t *testing.T) {
	headers := http.Header{}
	headers.Set("subscription-userinfo", "upload=1024; download=2048; total=10240; expire=1767225600")
	headers.Set("profile-web-page", "https://example.com/user")
	headers.Set("profile-update-interval", "3600")

	meta := parseSubscriptionUsageMeta(headers)

	if meta.UploadBytes == nil || *meta.UploadBytes != 1024 {
		t.Fatalf("unexpected upload bytes: %#v", meta.UploadBytes)
	}
	if meta.DownloadBytes == nil || *meta.DownloadBytes != 2048 {
		t.Fatalf("unexpected download bytes: %#v", meta.DownloadBytes)
	}
	if meta.TotalBytes == nil || *meta.TotalBytes != 10240 {
		t.Fatalf("unexpected total bytes: %#v", meta.TotalBytes)
	}
	if meta.ExpireUnix == nil || *meta.ExpireUnix != 1767225600 {
		t.Fatalf("unexpected expire: %#v", meta.ExpireUnix)
	}
	if meta.ProfileWebPage == nil || *meta.ProfileWebPage != "https://example.com/user" {
		t.Fatalf("unexpected profile web page: %#v", meta.ProfileWebPage)
	}
	if meta.ProfileUpdateSeconds == nil || *meta.ProfileUpdateSeconds != 3600 {
		t.Fatalf("unexpected profile interval: %#v", meta.ProfileUpdateSeconds)
	}
	if meta.UserinfoRaw == nil || *meta.UserinfoRaw == "" {
		t.Fatalf("userinfo raw should be captured")
	}
	if meta.UserinfoUpdatedAt == nil || *meta.UserinfoUpdatedAt == "" {
		t.Fatalf("userinfo updated at should be set")
	}
}

func TestParseSubscriptionUsageMeta_EmptyHeaders(t *testing.T) {
	meta := parseSubscriptionUsageMeta(http.Header{})
	if meta.UploadBytes != nil || meta.DownloadBytes != nil || meta.TotalBytes != nil || meta.ExpireUnix != nil {
		t.Fatalf("expected quota fields nil, got %+v", meta)
	}
	if meta.ProfileWebPage != nil || meta.ProfileUpdateSeconds != nil || meta.UserinfoRaw != nil || meta.UserinfoUpdatedAt != nil {
		t.Fatalf("expected profile fields nil, got %+v", meta)
	}
}
