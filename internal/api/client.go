// internal/api/client.go

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	Base     string
	JWT      string
	hc       *http.Client
	lastBody io.Reader // stores the last response body for debugging
}

// ListAuthors returns all users on the Ghost site. Ghost's Admin API exposes
// authors under /users/ (not /authors/ — that's Content API); any user with
// appropriate permissions can be listed as a post author.
func (c *Client) ListAuthors(ctx context.Context) ([]AuthorRef, error) {
	var res struct {
		Users []AuthorRef `json:"users"`
	}
	if err := c.Get(ctx, "users/?limit=all", &res); err != nil {
		return nil, err
	}
	return res.Users, nil
}

// ListTiers fetches all membership tiers from Ghost
func (c *Client) ListTiers(ctx context.Context) ([]TierRef, error) {
	var res struct {
		Tiers []TierRef `json:"tiers"`
	}
	if err := c.Get(ctx, "tiers/?limit=all", &res); err != nil {
		return nil, err
	}
	return res.Tiers, nil
}

// ListTags fetches all tags defined on the Ghost site.
func (c *Client) ListTags(ctx context.Context) ([]TagRef, error) {
	var res struct {
		Tags []TagRef `json:"tags"`
	}
	if err := c.Get(ctx, "tags/?limit=all", &res); err != nil {
		return nil, err
	}
	return res.Tags, nil
}

func New(base, jwt string) *Client {
	return &Client{
		Base: base,
		JWT:  jwt,
		hc:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, ctype string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Ghost %s", c.JWT))
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	return c.hc.Do(req)
}

// Cap on how much of a non-JSON error body we read/print. Avoids buffering a
// multi-MB HTML error page just to show a snippet.
const errorBodyReadCap = 4096
const errorBodySnippetLen = 300

// nonJSONError formats a compact error for responses whose body is not JSON
// (e.g. HTML error pages served by an edge/CDN layer on 5xx). Includes the
// HTTP status and a short snippet of the body so callers can diagnose without
// dumping an entire HTML page to the terminal.
func nonJSONError(res *http.Response, body []byte) error {
	snippet := bytes.TrimSpace(body)
	if len(snippet) > errorBodySnippetLen {
		snippet = append(snippet[:errorBodySnippetLen:errorBodySnippetLen], []byte("…")...)
	}
	status := http.StatusText(res.StatusCode)
	if status == "" {
		status = "unknown"
	}
	return fmt.Errorf("ghost API error: %d %s (non-JSON response): %s", res.StatusCode, status, snippet)
}

// ghostErrorEnvelope is the canonical Admin API error shape:
//
//	{"errors":[{"message":"...","code":"...","type":"...","id":"..."}]}
type ghostErrorEnvelope struct {
	Errors []struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Type    string `json:"type"`
		ID      string `json:"id"`
	} `json:"errors"`
}

// jsonError parses Ghost's error envelope and returns a compact error string.
// If the body doesn't fit the envelope, falls back to a generic status message.
func jsonError(res *http.Response, body []byte) error {
	var env ghostErrorEnvelope
	if err := json.Unmarshal(body, &env); err == nil && len(env.Errors) > 0 {
		e := env.Errors[0]
		label := e.Code
		if label == "" {
			label = e.Type
		}
		if label == "" {
			label = http.StatusText(res.StatusCode)
		}
		msg := e.Message
		if msg == "" {
			msg = "(no message)"
		}
		return fmt.Errorf("ghost API error: %d %s: %s", res.StatusCode, label, msg)
	}
	return fmt.Errorf("ghost API error: %d %s", res.StatusCode, http.StatusText(res.StatusCode))
}

// isSuccessStatus reports whether an HTTP status code indicates success.
// Ghost uses 2xx for success; everything else (including redirects) is treated
// as an error here since the Admin API isn't expected to redirect.
func isSuccessStatus(code int) bool {
	return code >= 200 && code < 300
}

func (c *Client) Get(ctx context.Context, path string, out any) error {
	res, err := c.do(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return err
	}
	defer res.Body.Close()

	isJSON := strings.HasPrefix(res.Header.Get("Content-Type"), "application/json")

	if !isSuccessStatus(res.StatusCode) {
		body, _ := io.ReadAll(io.LimitReader(res.Body, errorBodyReadCap))
		if isJSON {
			return jsonError(res, body)
		}
		return nonJSONError(res, body)
	}

	if !isJSON {
		body, _ := io.ReadAll(io.LimitReader(res.Body, errorBodyReadCap))
		return nonJSONError(res, body)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (c *Client) Post(ctx context.Context, path string, payload any, out any) error {
	buf, _ := json.Marshal(payload)
	res, err := c.do(ctx, http.MethodPost, path, bytes.NewReader(buf), "application/json")
	if err != nil {
		return err
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	c.lastBody = bytes.NewReader(respBody)

	isJSON := strings.HasPrefix(res.Header.Get("Content-Type"), "application/json")

	if !isSuccessStatus(res.StatusCode) {
		if isJSON {
			return jsonError(res, respBody)
		}
		return nonJSONError(res, respBody)
	}

	if !isJSON {
		return nonJSONError(res, respBody)
	}
	return json.Unmarshal(respBody, out)
}

func (c *Client) Put(ctx context.Context, path string, payload any, out any) error {
	buf, _ := json.Marshal(payload)
	res, err := c.do(ctx, http.MethodPut, path, bytes.NewReader(buf), "application/json")
	if err != nil {
		return err
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	c.lastBody = bytes.NewReader(respBody)

	isJSON := strings.HasPrefix(res.Header.Get("Content-Type"), "application/json")

	if !isSuccessStatus(res.StatusCode) {
		if isJSON {
			return jsonError(res, respBody)
		}
		return nonJSONError(res, respBody)
	}

	if !isJSON {
		return nonJSONError(res, respBody)
	}
	return json.Unmarshal(respBody, out)
}

func (c *Client) UploadImage(ctx context.Context, absPath string) (string, error) {
	f, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, _ := w.CreateFormFile("file", filepath.Base(absPath))
	_, _ = fw.Write(f)
	_ = w.Close()

	var resp struct {
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	}

	if err := c.Post(ctx, "images/upload/", &b, &resp); err != nil {
		return "", err
	}
	if len(resp.Images) == 0 {
		return "", fmt.Errorf("no image returned")
	}
	return resp.Images[0].URL, nil
}

func (c *Client) GetPost(ctx context.Context, id string) (Post, error) {
	var res struct {
		Posts []Post `json:"posts"`
	}
	if err := c.Get(ctx, "posts/"+id+"/", &res); err != nil {
		return Post{}, err
	}
	if len(res.Posts) == 0 {
		return Post{}, fmt.Errorf("post %s not found", id)
	}
	return res.Posts[0], nil
}
