// internal/images/uploader.go

package images

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"regexp"
)

var imgRe = regexp.MustCompile(`!\[[^\]]*]\(([^)]+)\)`)

type Service struct {
	BaseURL  string
	Client   *http.Client
	AdminJWT string
	cache    map[string]string // sha1 → remoteURL
}

func New(base string, jwt string, c *http.Client) *Service {
	return &Service{
		BaseURL:  base,
		Client:   c,
		AdminJWT: jwt,
		cache:    make(map[string]string),
	}
}

func (s *Service) Rewrite(md []byte, root string) ([]byte, error) {
	return imgRe.ReplaceAllFunc(md, func(m []byte) []byte {
		match := imgRe.FindSubmatch(m)
		locPath := string(match[1])
		full := filepath.Join(root, locPath)

		remote, err := s.upload(full)
		if err != nil {
			fmt.Printf("Error uploading file: %s\n", err.Error())
			return m // keep local ref if upload fails
		}
		return bytes.Replace(m, match[1], []byte(remote), 1)
	}), nil
}

func (s *Service) upload(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := fmt.Sprintf("%x", sha1.Sum(raw))
	if url, ok := s.cache[sum]; ok {
		return url, nil
	}

	body := &bytes.Buffer{}
	w, err := imageFormWriter(raw, body, path)
	if err != nil {
		return "", err
	}
	w.Close()

	req, _ := http.NewRequest("POST", s.BaseURL+"images/upload/", body)
	req.Header.Set("Authorization", "Ghost "+s.AdminJWT)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload failed %v", err)
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("status code %d", resp.StatusCode)
	}
	// parse {"images":[{"url":"https://…"}]}
	var r struct {
		Images []struct {
			URL string `json:"url"`
		}
	}
	json.NewDecoder(resp.Body).Decode(&r)
	remote := r.Images[0].URL
	s.cache[sum] = remote
	return remote, nil
}

func imageFormWriter(file []byte, body *bytes.Buffer, path string) (*multipart.Writer, error) {
	w := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", multipart.FileContentDisposition("file", filepath.Base(path)))
	h.Set("Content-Type", mime.TypeByExtension(filepath.Ext(path)))
	part, _ := w.CreatePart(h)
	_, err := io.Copy(part, bytes.NewReader(file))
	if err != nil {
		return nil, fmt.Errorf("error writing form: %s", err.Error())
	}
	w.WriteField("ref", path)
	return w, nil
}
