package seedance

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"geekai/core/types"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestClientCreateAssetFetchesAssetWhenCreateOnlyReturnsID(t *testing.T) {
	calls := []string{}
	client := &Client{
		config: types.SeedanceConfig{ApiURL: "https://seedance.example.com", BearerToken: "token"},
		httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls = append(calls, req.URL.Path)
			var body string
			switch req.URL.Path {
			case "/open/CreateAsset":
				body = `{"Id":"asset-created"}`
			case "/open/GetAsset":
				body = `{"Id":"asset-created","Name":"测试人像","URL":"https://cdn.example.com/person.jpg","AssetType":"Image","Status":"Active"}`
			default:
				t.Fatalf("unexpected path %s", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Header:     make(http.Header),
			}, nil
		})},
	}

	got, err := client.CreateAsset(&CreateAssetReq{URL: "https://cdn.example.com/person.jpg", AssetType: "Image", Name: "测试人像"})
	if err != nil {
		t.Fatalf("CreateAsset returned error: %v", err)
	}
	if got.ID != "asset-created" || got.URL != "https://cdn.example.com/person.jpg" || got.Name != "测试人像" || got.Status != "Active" {
		t.Fatalf("unexpected asset response: %+v", got)
	}
	if len(calls) != 2 || calls[0] != "/open/CreateAsset" || calls[1] != "/open/GetAsset" {
		t.Fatalf("expected create then get asset calls, got %v", calls)
	}
}
