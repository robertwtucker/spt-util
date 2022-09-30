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
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-http-utils/headers"
	"github.com/robertwtucker/spt-util/internal/config"
	log "github.com/sirupsen/logrus"
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
	log.Info("starting demo environment initialization")

	release := viper.GetString(config.GlobalReleaseKey)
	log.WithField("release", release).Debug()
	namespace := viper.GetString(config.GlobalNamespaceKey)
	log.WithField("namespace", namespace).Debug()
	scalerHost := fmt.Sprintf("http://%s-scaler.%s.svc.cluster.local", release, namespace)
	scalerHost = "http://localhost:30600" // Temporary for testing
	log.WithField("scalerHost", scalerHost).Info()
	authEncoding := getBasicAuthEncoding(
		viper.GetString(config.DemoUsernameKey),
		viper.GetString(config.DemoPasswordKey),
	)
	authHeader := fmt.Sprintf("Basic %s", authEncoding)
	client := &http.Client{Timeout: time.Second * 2}

	//  Import ICM Environment Variables
	//  PUT {{baseUrl}}/api/content/v1/inspireEnvironments
	url := fmt.Sprintf("%s/%s", scalerHost, "api/content/v1/inspireEnvironments")
	envFilePath := viper.GetString(config.DemoInitEnvFileKey)
	log.WithField("envFilePath", envFilePath).Debug("reading environment file content")
	envFileContent, err := os.ReadFile(envFilePath)
	if err != nil {
		log.Fatal("error reading environment file:", err)
	}
	// request
	importEnvRequest, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(envFileContent))
	if err != nil {
		log.Fatal("error creating import environment request:", err)
	}
	log.WithFields(log.Fields{
		"method": importEnvRequest.Method,
		"url":    importEnvRequest.URL,
	}).Debug("created import environment request")
	importEnvRequest.Header.Set(headers.Accept, "application/json")
	importEnvRequest.Header.Set(headers.Authorization, authHeader)
	importEnvRequest.Header.Set(headers.ContentType, "application/json")
	// response
	log.Info("importing ICM environment settings")
	importEnvResponse, err := client.Do(importEnvRequest)
	if err != nil {
		log.Fatal("error sending import environment request:", err)
	}
	defer func() { _ = importEnvResponse.Body.Close }()
	// process
	if importEnvResponse.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(importEnvResponse.Body)
		log.Fatal("error: import env request returned non-ok status:", importEnvResponse.StatusCode, string(respBody))
	}
	log.Info("ICM environment settings imported successfully")

	//  Import Changeset w/workflows for rest of process
	//  {{baseUrl}}/api/content/v1/upload/changesets (multipart/form-data)
	url = fmt.Sprintf("%s/%s", scalerHost, "api/content/v1/upload/changesets")
	chsFilePath := viper.GetString(config.DemoInitChsFileKey)
	// request
	importChangesetRequest, err := newFileUploadRequest(url, chsFilePath)
	if err != nil {
		log.Fatal("error creating import changeset request:", err)
	}
	log.WithFields(log.Fields{
		"method": importChangesetRequest.Method,
		"url":    importChangesetRequest.URL,
	}).Debug("created import changeset request")
	importChangesetRequest.Header.Set(headers.Authorization, authHeader)
	// response
	log.Info("sending import changeset request")
	importChangesetResponse, err := client.Do(importChangesetRequest)
	if err != nil {
		log.Fatal("error sending import changeset request:", err)
	}
	defer func() { _ = importChangesetResponse.Body.Close }()
	// process
	if importChangesetResponse.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(importChangesetResponse.Body)
		log.Fatal("error: import changeset request returned non-ok status: ", string(respBody))
	}
	log.Info("workflow changeset imported successfully")

	//  Find required workflows
	//  GET {{baseUrl}}/api/integration/v2/workflows/
	url = fmt.Sprintf("%s/%s", scalerHost, "api/integration/v2/workflows")
	// request
	getWorkflowsRequest, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal("error creating get workflows request: ", err)
	}
	log.WithFields(log.Fields{
		"method": getWorkflowsRequest.Method,
		"url":    getWorkflowsRequest.URL,
	}).Debug("created workflows request")
	getWorkflowsRequest.Header.Set(headers.Authorization, authHeader)
	// response
	log.Info("sending workflows request")
	getWorkflowsResponse, err := client.Do(getWorkflowsRequest)
	if err != nil {
		log.Fatal("error sending workflows request: ", err)
	}
	defer func() { _ = getWorkflowsResponse.Body.Close }()
	// process
	getWorkflowsResponseBody, _ := io.ReadAll(getWorkflowsResponse.Body)
	if getWorkflowsResponse.StatusCode >= http.StatusBadRequest {
		log.Fatal("error: workflows request returned non-ok status: ", getWorkflowsResponse.StatusCode, string(getWorkflowsResponseBody))
	}
	log.Info("list of workflows received successfully")

	// Serialize the list of workflows from JSON
	var workflowsResponse WorkflowsResponse
	if err := json.Unmarshal(getWorkflowsResponseBody, &workflowsResponse); err != nil {
		log.Fatal("error processing JSON from get workflows response: ", err)
	}
	log.Debug("# scaler workflows: ", len(workflowsResponse.Workflows))

	// Find the IDs of the SPT workflows
	targetWorkflows := viper.GetStringSlice(config.DemoInitWorkflowsKey)
	numTargetWorkflows := sort.StringSlice.Len(targetWorkflows)
	log.Debug("# target workflows: ", numTargetWorkflows)
	if numTargetWorkflows > 1 {
		sort.StringSlice(targetWorkflows).Sort()
	}
	log.Debug("target workflows:", targetWorkflows)
	var foundWorkflows = make([]string, numTargetWorkflows)
	for i, workflow := range workflowsResponse.Workflows {
		if index := sort.SearchStrings(targetWorkflows, workflow.Name); index < numTargetWorkflows {
			if workflow.Name == targetWorkflows[index] {
				log.WithFields(log.Fields{
					"name": workflow.Name,
					"id":   workflow.ID,
				}).Debug("matched workflow")
				foundWorkflows[i] = workflow.ID
			}
		}
	}
	log.Debug("# workflows found: ", sort.StringSlice.Len(foundWorkflows))
	if sort.StringSlice.Len(foundWorkflows) == 0 {
		log.Fatal("error: no workflows found to deploy!")
	}

	// Deploy the required workflows
	// PATCH {{baseUrl}}/api/integration/v2/workflows/{id}/
	for _, id := range foundWorkflows {
		url = fmt.Sprintf("%s/%s/%s", scalerHost, "api/integration/v2/workflows", id)
		requestBody, _ := json.Marshal(map[string]string{"status": "DEPLOYED"})
		// request
		request, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(requestBody))
		if err != nil {
			log.Fatal("error creating workflow deployment request ", err)
		}
		log.WithFields(log.Fields{
			"method": request.Method,
			"url":    request.URL,
		}).Debug("workflow deployment request")
		request.Header.Set(headers.Accept, "application/json")
		request.Header.Set(headers.Authorization, authHeader)
		request.Header.Set(headers.ContentType, "application/json")
		// response
		log.WithField("id", id).Info("sending workflow deployment request")
		response, err := client.Do(request)
		if err != nil {
			log.Fatal("error sending deploy workflow request: ", err)
		}
		_ = response.Body.Close()
		// process
		if response.StatusCode >= http.StatusBadRequest {
			log.Error("error deploying scaler workflow: ", response.StatusCode, id)
		}
		log.WithField("id", id).Info("workflow deployed successfully")
	}

	log.Info("completed demo environment initialization")
}

func getBasicAuthEncoding(user string, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, password)))
}

func newFileUploadRequest(uri string, path string) (*http.Request, error) {
	log.WithField("path", path).Debug("reading upload file content")
	file, err := os.Open(path)
	if err != nil {
		log.Error("error opening upload file:", err)
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
