// Copyright (c) 2014 The AUTHORS
//
// This file is part of trunk.
//
// trunk is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// trunk is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with trunk.  If not, see <http://www.gnu.org/licenses/>.

package circleci

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
)

const (
	LibraryVersion = "0.0.1"

	defaultBaseURL   = "https://circleci.com/api/v1/"
	defaultUserAgent = "go-circleci/" + LibraryVersion
)

var (
	ErrEmptyToken      = errors.New("circleci: token in not set")
	ErrInvalidToken    = errors.New("circleci: invalid token string")
	ErrNoTrailingSlash = errors.New("circleci: trailing slash missing")
)

type ErrHTTP struct {
	*http.Response
}

func (err *ErrHTTP) Error() string {
	return fmt.Sprintf("%v %v -> %v", err.Request.Method, err.Request.URL, err.Status)
}

type Client struct {
	// Pivotal Tracker access token to be used to authenticate API requests.
	token string

	// HTTP client to be used for communication with the Pivotal Tracker API.
	client *http.Client

	// Base URL of the Pivotal Tracker API that is to be used to form API requests.
	baseURL *url.URL

	// User-Agent header to use when connecting to the Pivotal Tracker API.
	userAgent string
}

func NewClient(apiToken string) (*Client, error) {
	if apiToken == "" {
		return nil, ErrEmptyToken
	}
	if !regexp.MustCompile("^[a-f0-9]{40}$").MatchString(apiToken) {
		return nil, ErrInvalidToken
	}

	baseURL, _ := url.Parse(defaultBaseURL)
	return &Client{
		token:     apiToken,
		client:    http.DefaultClient,
		baseURL:   baseURL,
		userAgent: defaultUserAgent,
	}, nil
}

func (c *Client) SetBaseURL(baseURL string) error {
	u, err := url.Parse(baseURL)
	if err != nil {
		return err
	}

	if u.Path != "" && u.Path[len(u.Path)-1] != '/' {
		return ErrNoTrailingSlash
	}

	c.baseURL = u
	return nil
}

func (c *Client) SetUserAgent(agent string) {
	c.userAgent = agent
}

func (c *Client) NewRequest(method, urlPath string, body interface{}) (*http.Request, error) {
	path, err := url.Parse(urlPath)
	if err != nil {
		return nil, err
	}

	u := c.baseURL.ResolveReference(path)
	if u.RawQuery != "" {
		u.RawQuery += "&"
	}
	u.RawQuery += "circleci-token=" + c.token

	buf := new(bytes.Buffer)
	if body != nil {
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	return req, nil
}

func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return resp, &ErrHTTP{resp}
	}

	if v != nil {
		err = json.NewDecoder(resp.Body).Decode(v)
	}

	return resp, err
}
