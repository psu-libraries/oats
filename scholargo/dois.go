package scholargo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type DOIMap map[string][]string

func (c *Client) DOIs() (DOIMap, error) {
	req, err := c.NewRequest("GET", "dois", nil)
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
	var data DOIMap
	return data, json.NewDecoder(resp.Body).Decode(&data)
}
