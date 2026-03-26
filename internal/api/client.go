package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/jessewaites/cableknit-cli/internal/config"
)

type Client struct {
	http    *http.Client
	baseURL string
	token   string
	debug   bool
}

func NewClient() *Client {
	return &Client{
		http:    &http.Client{Timeout: 30 * time.Second},
		baseURL: config.APIURL(),
		token:   config.Token(),
	}
}

func NewClientWithToken(token string) *Client {
	c := NewClient()
	c.token = token
	return c
}

func (c *Client) SetDebug(d bool) { c.debug = d }

func (c *Client) url(path string) string {
	return c.baseURL + path
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("User-Agent", "cableknit-cli")

	if c.debug {
		fmt.Fprintf(os.Stderr, "[debug] %s %s\n", req.Method, req.URL)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: unable to reach CableKnit API. Check your connection")
	}
	return resp, nil
}

func (c *Client) JSON(method, path string, body any, result any) error {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.url(path), buf)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

func (c *Client) Multipart(path string, fieldName string, fileName string, r io.Reader, result any) error {
	return c.MultipartWithProgress(path, fieldName, fileName, r, 0, nil, result)
}

func (c *Client) MultipartWithProgress(path string, fieldName string, fileName string, r io.Reader, size int64, onProgress func(int64), result any) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, r); err != nil {
		return err
	}
	writer.Close()

	var reqBody io.Reader = body
	if onProgress != nil && size > 0 {
		reqBody = &progressReader{r: body, onProgress: onProgress}
	}

	req, err := http.NewRequest("POST", c.url(path), reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if size > 0 {
		req.ContentLength = int64(body.Len())
	}

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

func (c *Client) SSE(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.url(path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("User-Agent", "cableknit-cli")

	// No timeout for SSE
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: unable to reach CableKnit API. Check your connection")
	}
	return resp, nil
}

func (c *Client) handleResponse(resp *http.Response, result any) error {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if c.debug {
		fmt.Fprintf(os.Stderr, "[debug] status=%d body=%s\n", resp.StatusCode, string(data))
	}

	switch {
	case resp.StatusCode == 401:
		return fmt.Errorf("not logged in. Run `cableknit login` first")
	case resp.StatusCode == 422:
		var apiErr APIError
		if json.Unmarshal(data, &apiErr) == nil && apiErr.Error != "" {
			return fmt.Errorf("%s", apiErr.Error)
		}
		// Try unmarshaling into result directly (validation responses return 422 with structured data)
		if result != nil {
			if err := json.Unmarshal(data, result); err == nil {
				return nil
			}
		}
		return fmt.Errorf("validation error: %s", string(data))
	case resp.StatusCode >= 400:
		var apiErr APIError
		if json.Unmarshal(data, &apiErr) == nil && apiErr.Error != "" {
			return fmt.Errorf("%s", apiErr.Error)
		}
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(data))
	}

	if result != nil {
		return json.Unmarshal(data, result)
	}
	return nil
}

type progressReader struct {
	r          io.Reader
	onProgress func(int64)
	read       int64
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	pr.read += int64(n)
	if pr.onProgress != nil {
		pr.onProgress(pr.read)
	}
	return n, err
}
