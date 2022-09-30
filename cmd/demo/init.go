//
// Copyright (c) 2022 Quadient Group AG
//
// This file is subject to the terms and conditions defined in the
// 'LICENSE' file found in the root of this source code package.
//

package demo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/robertwtucker/spt-util/internal/config"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-http-utils/headers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// InitCmd represents the init command
var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "initializes a demo instance",
	Long: `
Initializes a demo instance given the specified release and namespace.
    `,
	Run: func(cmd *cobra.Command, args []string) {
		executeInit()
	},
}

func init() {

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

type WorkflowsResponse struct {
	Workflows []struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Path          string `json:"path"`
		Status        string `json:"status"`
		WorkflowGroup string `json:"workflowGroup"`
	} `json:"workflows"`
}

func executeInit() {
	release := viper.GetString(config.GlobalReleaseKey)
	namespace := viper.GetString(config.GlobalNamespaceKey)
	scalerHost := fmt.Sprintf("http://%s-scaler.%s.svc.cluster.local", release, namespace)
	scalerHost = "http://localhost:30600" // Temporary for testing
	authEncoding := getBasicAuthEncoding(
		viper.GetString(config.DemoUsernameKey),
		viper.GetString(config.DemoPasswordKey),
	)
	authHeader := fmt.Sprintf("Basic %s", authEncoding)
	client := &http.Client{Timeout: time.Second * 2}

	//  Import ICM Environment Variables
	//  PUT {{baseUrl}}/api/content/v1/inspireEnvironments
	url := fmt.Sprintf("%s/%s", scalerHost, "api/content/v1/inspireEnvironments")
	envFileContent, err := os.ReadFile(viper.GetString(config.DemoInitEnvFileKey))
	if err != nil {
		fmt.Println("error reading environment file:", err)
		return
	}
	// request
	importEnvRequest, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(envFileContent))
	if err != nil {
		fmt.Println("error creating import environment request:", err)
		return
	}
	importEnvRequest.Header.Set(headers.Accept, "application/json")
	importEnvRequest.Header.Set(headers.Authorization, authHeader)
	importEnvRequest.Header.Set(headers.ContentType, "application/json")
	// response
	importEnvResponse, err := client.Do(importEnvRequest)
	if err != nil {
		fmt.Println("error sending import env request:", err)
		return
	}
	defer func() { _ = importEnvResponse.Body.Close }()
	// process
	if importEnvResponse.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(importEnvResponse.Body)
		fmt.Println("error: import env request returned non-ok status:", importEnvResponse.StatusCode, string(respBody))
		return
	}

	//  Import Changeset w/workflows for rest of process
	//  {{baseUrl}}/api/content/v1/upload/changesets (multipart/form-data)
	url = fmt.Sprintf("%s/%s", scalerHost, "api/content/v1/upload/changesets")
	chsFilePath := viper.GetString(config.DemoInitChsFileKey)
	// request
	importChangesetRequest, err := newFileUploadRequest(url, chsFilePath)
	if err != nil {
		fmt.Println("error creating upload changeset request:", err)
	}
	importChangesetRequest.Header.Set(headers.Authorization, authHeader)
	// response
	importChangesetResponse, err := client.Do(importChangesetRequest)
	if err != nil {
		fmt.Println("error sending import changeset request:", err)
		return
	}
	defer func() { _ = importChangesetResponse.Body.Close }()
	// process
	if importChangesetResponse.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(importChangesetResponse.Body)
		fmt.Println("error: import changeset request returned non-ok status:", importChangesetResponse.StatusCode, string(respBody))
		return
	}

	//  Find ID of WFS
	//  GET {{baseUrl}}/api/integration/v2/workflows/
	url = fmt.Sprintf("%s/%s", scalerHost, "api/integration/v2/workflows")
	// request
	getWorkflowsRequest, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println("error creating get workflows request", err)
		return
	}
	getWorkflowsRequest.Header.Set(headers.Authorization, authHeader)
	// response
	getWorkflowsResponse, err := client.Do(getWorkflowsRequest)
	if err != nil {
		fmt.Println("error sending get worklfows request", err)
		return
	}
	defer func() { _ = getWorkflowsResponse.Body.Close }()
	// process
	getWorkflowsResponseBody, _ := io.ReadAll(getWorkflowsResponse.Body)
	if getWorkflowsResponse.StatusCode >= http.StatusBadRequest {
		fmt.Println("error: import changeset request returned non-ok status:", getWorkflowsResponse.StatusCode, string(getWorkflowsResponseBody))
		return
	}

	// Serialize the list of workflows from JSON
	var workflowsResponse WorkflowsResponse
	if err := json.Unmarshal(getWorkflowsResponseBody, &workflowsResponse); err != nil {
		fmt.Println("error processing JSON from get workflows response:", err)
		return
	}
	fmt.Println("scaler workflows:", workflowsResponse.Workflows)

	// Find the IDs of the SPT workflows
	targetWorkflows := viper.GetStringSlice(config.DemoInitWorkflowsKey)
	numTargetWorkflows := sort.StringSlice.Len(targetWorkflows)
	fmt.Println("# target workflows:", numTargetWorkflows)
	if numTargetWorkflows > 1 {
		sort.StringSlice(targetWorkflows).Sort()
	}
	fmt.Println("target workflow names:", targetWorkflows)
	var foundWorkflows = make([]string, numTargetWorkflows)
	for i, workflow := range workflowsResponse.Workflows {
		if index := sort.SearchStrings(targetWorkflows, workflow.Name); index < numTargetWorkflows {
			if workflow.Name == targetWorkflows[index] {
				foundWorkflows[i] = workflow.ID
			}
		}
	}
	if sort.StringSlice.Len(foundWorkflows) == 0 {
		fmt.Println("error: no workflows found to deploy!")
		return
	}

	// Deploy the required workflows
	// PATCH {{baseUrl}}/api/integration/v2/workflows/{id}/
	fmt.Println("found workflows:", foundWorkflows)
	for _, id := range foundWorkflows {
		url = fmt.Sprintf("%s/%s/%s", scalerHost, "api/integration/v2/workflows", id)
		requestBody, _ := json.Marshal(map[string]string{"status": "DEPLOYED"})
		// request
		//fmt.Println("patch", url, string(requestBody))
		request, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(requestBody))
		if err != nil {
			fmt.Println("error creating workflow deploy request", err)
			return
		}
		request.Header.Set(headers.Accept, "application/json")
		request.Header.Set(headers.Authorization, authHeader)
		request.Header.Set(headers.ContentType, "application/json")
		// response
		response, err := client.Do(request)
		if err != nil {
			fmt.Println("error sending deploy workflow request:", err)
			return
		}
		_ = response.Body.Close()
		// process
		if response.StatusCode >= http.StatusBadRequest {
			fmt.Println("error deploying scaler workflow:", response.StatusCode, id)
		}
	}
}

func getBasicAuthEncoding(user string, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, password)))
}

func newFileUploadRequest(uri string, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("error opening file " + path)
		return nil, err
	}
	defer func() { _ = file.Close() }()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("changeset", filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, uri, body)
	req.Header.Set(headers.ContentType, writer.FormDataContentType())
	return req, err
}
