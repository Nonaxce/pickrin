package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

func createUserAgent(projectName, version string) string {
	// This retrieves golangs env, which is very likely but not always the env of the device
	osName := runtime.GOOS
	if osName == "windows" {
		//osName = "Windows NT" // everything windows
	}
	archName := runtime.GOARCH

	return fmt.Sprintf("%s/%s (%s; %s)", projectName, version, osName, archName)
}

type APIClient struct {
	client     http.Client
	userAgent  string
	APIHost    string
	APIVersion string
}

func newAPIClient(apiURL, apiVersion string) *APIClient {
	return &APIClient{
		client:     http.Client{},
		userAgent:  createUserAgent("modpickrin", "0.1.0"),
		APIHost:    apiURL,
		APIVersion: apiVersion,
	}
}

func (a *APIClient) VersionedAPIURL() string {
	return fmt.Sprintf("%s/%s", a.APIHost, a.APIVersion)
}

func (a *APIClient) DownloadFile(ctx context.Context, url string, dest string) (b int64, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return b, err
	}

	req.Header.Add("User-Agent", a.userAgent)

	resp, err := a.client.Do(req)
	if err != nil {
		return b, err
	}
	defer resp.Body.Close()

	err = os.MkdirAll(filepath.Dir(dest), os.ModePerm)
	if err != nil {
		return b, err
	}

	file, err := os.Create(dest)
	if err != nil {
		return b, err
	}
	defer file.Close()

	b, err = io.Copy(file, resp.Body)
	if err != nil {
		return b, err
	}

	return b, nil
}

// 8 Char Base62 Alphanumeric ID
var projectIdRegex, _ = regexp.Compile("^[a-zA-Z0-9]{8}$")

// returns information about a project on modrinth with a project id
func (a *APIClient) GetProject(projectId string) (p Project, e error) {
	// validate project id input
	if ok := projectIdRegex.MatchString(projectId); !ok {
		return p, errors.New("Invalid string for project id")
	}

	reqUrl := fmt.Sprintf("%s/project/%s", a.VersionedAPIURL(), projectId)

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return p, err
	}

	req.Header.Add("User-Agent", a.userAgent)

	resp, err := a.client.Do(req)
	if err != nil {
		return p, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return p, errors.New("404 Project not found")
	}

	decoder := json.NewDecoder(resp.Body)

	p = Project{}
	err = decoder.Decode(&p)
	if err != nil {
		return p, err
	}

	return p, nil
}
