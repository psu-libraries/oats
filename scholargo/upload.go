package scholargo

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/zRedShift/mimemagic"
)

// upload is a *complete* upload
type upload struct {
	ID       string    `json:"id"`
	Storage  string    `json:"storage"`
	Metadata *fileMeta `json:"metadata"`
}

// fileMeta is all the file info sent with deposit
type fileMeta struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MIMEType string `json:"mime_type"`

	// not included in JSON; used for presigned URL
	ext string // file extension
	md5 string // md5 encoded as base64
}

// uploadLocation represents upload location receive from 'upload' endoint.
type uploadLocation struct {
	URL    string // presigned URL to upload file to S3
	ID     string // name of file on S3
	Prefix string // file prefix on S3
}

// upload the file name and returns completed Upload
func (c *Client) upload(name string) (*upload, error) {
	meta, err := newFileMeta(name)
	if err != nil {
		return nil, err
	}
	loc, err := c.uploadLocation(meta)
	if err != nil {
		return nil, err
	}
	reader, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(`PUT`, loc.URL, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-MD5", meta.md5)
	req.ContentLength = meta.Size
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		buff := &bytes.Buffer{}
		io.Copy(buff, resp.Body)
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, buff.String())
	}
	return &upload{
		ID:       loc.ID,
		Storage:  loc.Prefix,
		Metadata: meta,
	}, nil
}

// returns presigned url for uploading file with name.
func (c *Client) uploadLocation(meta *fileMeta) (*uploadLocation, error) {
	reqBody := make(map[string]string)
	if meta.ext == "" {
		return nil, fmt.Errorf("missing file extension: %s", meta.Filename)
	}
	reqBody["extension"] = meta.ext
	reqBody["content_md5"] = meta.md5
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(`POST`, "uploads", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		buff := &bytes.Buffer{}
		io.Copy(buff, resp.Body)
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, buff.String())
	}
	var upload uploadLocation
	err = json.NewDecoder(resp.Body).Decode(&upload)
	if err != nil {
		return nil, err
	}
	return &upload, nil
}

// returns file info for file name
func newFileMeta(name string) (*fileMeta, error) {
	var meta fileMeta
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	mtype, err := mimemagic.MatchFile(file, -1)
	if err != nil {
		return nil, err
	}
	meta.MIMEType = mtype.MediaType()
	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, err
	}
	hash := md5.New()
	meta.Size, err = io.Copy(hash, file)
	if err != nil {
		return nil, err
	}
	meta.md5 = base64.StdEncoding.EncodeToString(hash.Sum(nil))
	meta.Filename = filepath.Base(name)
	if p := strings.Split(meta.Filename, "."); len(p) > 1 {
		meta.ext = p[len(p)-1]
	}
	return &meta, nil
}
