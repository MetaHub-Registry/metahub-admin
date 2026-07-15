package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/types/ref"
)

// fakeBlobReg is a minimal OCI registry that answers the /v2/ API ping and HEAD
// blob existence checks, so blobPresent can be exercised without a live
// registry. It returns 200 (present) or 404 (absent) for any HEAD on a blob.
func fakeBlobReg(t *testing.T, present bool) (host string, closeFn func()) {
	t.Helper()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
		if r.Method == http.MethodHead && strings.Contains(r.URL.Path, "/blobs/") {
			if present {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(h)
	return strings.TrimPrefix(srv.URL, "http://"), srv.Close
}

// TestBlobPresent asserts the HEAD-existence predicate that lets pushLayerPlans
// skip the slow S3-egress upload of an already-present layer: 200 → true
// (skip), 404 → false (push).
func TestBlobPresent(t *testing.T) {
	dig := digest.FromString("model-layer-content")
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"present-skips-upload", true},
		{"absent-triggers-push", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host, closeFn := fakeBlobReg(t, tc.want)
			defer closeFn()
			rc := newPushClient(host, "u", "p", false)
			defer rc.Close(context.Background(), ref.Ref{})
			imgRef, err := ref.New(host + "/models/x-layer:tag")
			if err != nil {
				t.Fatalf("ref.New: %v", err)
			}
			if got := blobPresent(context.Background(), rc, imgRef, dig, 19); got != tc.want {
				t.Fatalf("blobPresent = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestBlobPresentUnreachable is the safety-critical case: when the registry
// cannot be reached, blobPresent must report "not present" so the caller falls
// back to a full (always-safe) push instead of skipping a needed upload.
func TestBlobPresentUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rc := newPushClient("127.0.0.1:1", "u", "p", false)
	defer rc.Close(ctx, ref.Ref{})
	imgRef, err := ref.New("127.0.0.1:1/models/x-layer:tag")
	if err != nil {
		t.Fatalf("ref.New: %v", err)
	}
	if blobPresent(ctx, rc, imgRef, digest.FromString("x"), 1) {
		t.Fatal("blobPresent must be false when the registry is unreachable")
	}
}
