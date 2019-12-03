/*
Package jwplatform provides a client to talk to the JW Platform API,

		import (
		  "github.com/jwplayer/jwplatform-go"
		)

		client := jwplatform.NewClient("API_KEY", "API_SECRET")
*/
package jwplatform

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

const (
	APIVersion = "v1"
	APIHost    = "api.jwplatform.com"
	Version    = "0.1.0"
)

// Client represents the JWPlatform client object,
type Client struct {
	APIVersion string
	BaseURL    *url.URL
	UserAgent  string
	Version    string

	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// NewClient creates a new client object.
func NewClient(apiKey string, apiSecret string) *Client {
	return &Client{
		APIVersion: APIVersion,
		BaseURL: &url.URL{
			Scheme: "https",
			Host:   APIHost,
		},
		UserAgent: fmt.Sprintf("jwplatform-go/%s", Version),
		Version:   Version,

		apiKey:     apiKey,
		apiSecret:  apiSecret,
		httpClient: http.DefaultClient,
	}
}

// generateNonce generates a random 8 digit as a string.
func generateNonce() string {
	rand.Seed(time.Now().UTC().UnixNano())
	return fmt.Sprintf("%08d", rand.Intn(100000000))
}

// makeTimestamp gets the unix timestamp in seconds.
func makeTimestamp() int64 {
	return time.Now().UnixNano() / (int64(time.Second) / int64(time.Nanosecond))
}

// buildParams generates all parameters for api request.
func (c *Client) buildParams(params url.Values) url.Values {
	if params == nil {
		params = url.Values{}
	}

	params.Set("api_nonce", generateNonce())
	params.Set("api_key", c.apiKey)
	params.Set("api_format", "json")
	params.Set("api_timestamp", strconv.FormatInt(makeTimestamp(), 10))

	// hash signature base string
	sbs := params.Encode() + c.apiSecret
	h := sha1.New()
	h.Write([]byte(sbs))
	sha := hex.EncodeToString(h.Sum(nil))

	params.Set("api_signature", sha)

	return params
}

// newRequestWithContext create a new request with signed params
func (c *Client) newRequestWithContext(ctx context.Context, method, pathPart string, params url.Values) (*http.Request, error) {
	rel := &url.URL{Path: path.Join(c.APIVersion, pathPart)}
	absoluteURL := c.BaseURL.ResolveReference(rel)
	absoluteURL.RawQuery = c.buildParams(params).Encode()

	req, err := http.NewRequestWithContext(ctx, method, absoluteURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Add("User-Agent", c.UserAgent)

	return req, nil
}

// do decodes response body into v
func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(v)
	return resp, err
}

// MakeRequest requests with api signature and decodes json result into v
func (c *Client) MakeRequest(ctx context.Context, method, pathPart string, params url.Values, v interface{}) (*http.Response, error) {
	req, err := c.newRequestWithContext(ctx, method, pathPart, params)
	if err != nil {
		return nil, err
	}

	return c.do(req, &v)
}