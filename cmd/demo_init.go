//
// Copyright (c) 2022 Quadient Group AG
//
// This file is subject to the terms and conditions defined in the
// 'LICENSE' file found in the root of this source code package.
//

package cmd

import (
	"bytes"
	"context"
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
	"github.com/pkg/errors"
	"github.com/robertwtucker/spt-util/pkg/constants"
	"github.com/robertwtucker/spt-util/pkg/eventbus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Workflow struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	Status        string `json:"status"`
	WorkflowGroup string `json:"workflowGroup"`
}

type WorkflowsResponse struct {
	Workflows []Workflow `json:"workflows"`
}

type EventData struct {
	AuthHeader            string   `json:"authHeader"`
	ChsFilePath           string   `json:"chsFilePath"`
	EnvFilePath           string   `json:"envFilePath"`
	Namespace             string   `json:"namespace"`
	Release               string   `json:"release"`
	ScalerHost            string   `json:"scalerHost"`
	StartingWorkflowCount int      `json:"startingWorkflowCount"`
	TargetWorkflowNames   []string `json:"targetWorkflowNames"`
	WorkflowsToDeploy     []Workflow
}

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes a demo instance",
	Long: `
Initializes a demo instance given the specified release and namespace.
    `,
	Example: `
# initialize base content for a demo environment with debug logging enabled
spt-util demo init -d
	`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("starting demo environment initialization")

		// Setup context data
		var data = &EventData{
			AuthHeader: fmt.Sprintf(
				"Basic %s",
				getBasicAuthEncoding(
					viper.GetString(constants.DemoUsernameKey),
					viper.GetString(constants.DemoPasswordKey),
				),
			),
			ChsFilePath:         viper.GetString(constants.DemoInitChsFileKey),
			EnvFilePath:         viper.GetString(constants.DemoInitEnvFileKey),
			Namespace:           viper.GetString(constants.GlobalNamespaceKey),
			Release:             viper.GetString(constants.GlobalReleaseKey),
			ScalerHost:          viper.GetString(constants.DemoServerKey),
			TargetWorkflowNames: viper.GetStringSlice(constants.DemoInitWorkflowsKey),
			WorkflowsToDeploy:   []Workflow{},
		}
		data.StartingWorkflowCount = getScalerWorkflowCount(data.ScalerHost, data.AuthHeader)
		log.WithField("data", data).Debug("initial event data")

		// Create an EventBus instance
		eb := eventbus.NewEventBus()

		// Create event subscriptions
		chEnv := eb.SubscribeEvent(eventbus.InitStart)
		chChs := eb.SubscribeEvent(eventbus.InitStart)
		chFind := eb.SubscribeEvent(eventbus.InitFindScalerWorkflows)
		chDeploy := eb.SubscribeEvent(eventbus.InitDeployScalerWorkflows)

		// Start goroutines that receive the triggering events
		go importIcmEnvFile(chEnv, eb)         // <-InitStart
		go uploadIcmChangeSet(chChs, eb)       // <-InitStart
		go findScalerWorkflows(chFind, eb)     // <-InitFindScalerWorkflows
		go deployScalerWorkflows(chDeploy, eb) // <-InitDeployScalerWorkflows

		// Serialize our data and publish the initial event
		jsonData, _ := json.Marshal(data)
		log.Debug("publishing start event")
		eb.PublishEvent(eventbus.InitStart, jsonData)

		log.Info("ending demo environment initialization")
	},
}

//nolint:gochecknoinits // required for proper cobra initialization.
func init() {
	demoCmd.AddCommand(initCmd)
}

// Import the base set of ICM environment variables.
func importIcmEnvFile(channel eventbus.EventChannel, _ *eventbus.EventBus) {
	event := <-channel
	log.WithField(
		"event", event.Name,
	).Debug("received event in importIcmEnvFile")
	defer event.Done()

	data := EventData{}
	if eventData, ok := event.Data.([]byte); ok {
		_ = json.Unmarshal(eventData, &data)
	} else {
		log.Error("error decoding event data: not []byte")
		return
	}

	log.WithField(
		"path", data.EnvFilePath,
	).Debug("reading environment file content")
	envFileContent, err := os.ReadFile(data.EnvFilePath)
	if err != nil {
		log.Error("unable to read environment file: ", err)
		return
	}

	// request
	// PUT {{baseUrl}}/api/content/v1/inspireEnvironments
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPut,
		fmt.Sprintf(
			"%s/%s",
			data.ScalerHost,
			"api/content/v1/inspireEnvironments",
		),
		bytes.NewBuffer(envFileContent),
	)
	if err != nil {
		log.Error("failed to create import environment request: ", err)
		return
	}
	log.WithFields(log.Fields{
		"method": request.Method,
		"url":    request.URL,
	}).Debug("created import environment request")
	request.Header.Set(headers.Accept, "application/json")
	request.Header.Set(headers.Authorization, data.AuthHeader)
	request.Header.Set(headers.ContentType, "application/json")

	// response
	log.Info("importing environment variables")
	//nolint:gomnd // TODO: Externalize constant value in config file.
	client := &http.Client{Timeout: time.Second * 5}
	response, err := client.Do(request)
	if err != nil {
		log.Error("failed to send import environment request: ", err)
		return
	}
	defer func() { _ = response.Body.Close }()

	// process
	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		log.Errorf(
			"received non-ok HTTP status importing environment variables: [%d]:%s",
			response.StatusCode,
			string(body),
		)
		return
	}

	log.Info("environment variables imported successfully")
}

// Upload changeset w/workflows for rest of process.
func uploadIcmChangeSet(channel eventbus.EventChannel, eb *eventbus.EventBus) {
	event := <-channel
	log.WithField(
		"event", event.Name,
	).Debug("received event in uploadIcmChangeSet")
	defer event.Done()

	data := EventData{}
	if eventData, ok := event.Data.([]byte); ok {
		_ = json.Unmarshal(eventData, &data)
	} else {
		log.Error("error decoding event data: not []byte")
		return
	}

	// request
	//  {{baseUrl}}/api/content/v1/upload/changesets (multipart/form-data)
	url := fmt.Sprintf(
		"%s/%s",
		data.ScalerHost,
		"api/content/v1/upload/changesets",
	)
	request, err := newFileUploadRequest(url, data.ChsFilePath)
	if err != nil {
		log.Error("error creating upload changeset request: ", err)
		return
	}
	log.WithFields(log.Fields{
		"method": request.Method,
		"url":    request.URL,
	}).Debug("created import changeset request")
	request.Header.Set(headers.Authorization, data.AuthHeader)

	// response
	log.Info("uploading changeset")
	//nolint:gomnd // TODO: Externalize constant value in config file.
	client := &http.Client{Timeout: time.Second * 5}
	response, err := client.Do(request)
	if err != nil {
		log.Error("error sending import changeset request: ", err)
		return
	}
	defer func() { _ = response.Body.Close }()

	// process
	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		log.Errorf(
			"received non-ok HTTP status importing changeset: [%d]:%s",
			response.StatusCode,
			string(body),
		)
		return
	}
	log.Info("changeset uploaded successfully")

	// Trigger (publish) the next event process. The serialized
	// JSON hasn't changed, pass it as-is.
	eb.PublishEvent(eventbus.InitFindScalerWorkflows, event.Data)
}

// Find required workflows in Scaler.
func findScalerWorkflows(channel eventbus.EventChannel, eb *eventbus.EventBus) {
	event := <-channel
	log.WithField(
		"event", event.Name,
	).Debug("received event in findScalerWorkflows")
	defer event.Done()

	data := EventData{}
	if eventData, ok := event.Data.([]byte); ok {
		_ = json.Unmarshal(eventData, &data)
	} else {
		log.Error("error decoding event data: not []byte")
		return
	}

	// Loop until changeset with needed workflows is applied.
	currentWorkflowCount := data.StartingWorkflowCount
	var tries = 0
	for {
		//nolint:gomnd // TODO: Externalize constant value in config file.
		if tries > 15 {
			log.Error("exceeded try count waiting for workflows to be applied")
			return
		}
		if currentWorkflowCount > data.StartingWorkflowCount {
			log.Info("changeset workflows have been applied")
			break
		}
		currentWorkflowCount = getScalerWorkflowCount(data.ScalerHost, data.AuthHeader)
		log.WithFields(log.Fields{
			"workflows": currentWorkflowCount,
			"retries":   tries,
		}).Info("waiting for new workflows")
		tries++
		//nolint:gomnd // TODO: Externalize constant value in config file.
		time.Sleep(4 * time.Second)
	}

	// Find the required workflows.
	targetWorkflowNames := data.TargetWorkflowNames
	targetWorkflowCount := sort.StringSlice.Len(targetWorkflowNames)
	log.Debug("# target workflows: ", targetWorkflowCount)

	// sort.SearchStrings() used below expects a sorted slice.
	if targetWorkflowCount > 1 {
		sort.StringSlice(targetWorkflowNames).Sort()
	}

	workflows, err := getScalerWorkflows(data.ScalerHost, data.AuthHeader)
	if err != nil {
		log.Error("failed to get workflows to inspect: ", err)
	}

	deployable := []Workflow{}
	for _, workflow := range workflows {
		if index := sort.SearchStrings(targetWorkflowNames, workflow.Name); index < targetWorkflowCount {
			if workflow.Name == targetWorkflowNames[index] {
				log.WithFields(log.Fields{
					"name": workflow.Name,
					"id":   workflow.ID,
				}).Debug("matched workflow")
				if len(deployable) == 0 {
					deployable = []Workflow{workflow}
				} else {
					deployable = append(deployable, workflow)
				}
			}
		}
	}

	workflowsToDeployCount := len(deployable)
	log.Debug("# workflows found: ", workflowsToDeployCount)

	// Add workflows to our data structure.
	data.WorkflowsToDeploy = deployable
	// Re-marshal the data into JSON.
	jsonData, _ := json.Marshal(data)

	// Trigger (publish) the next event process..
	eb.PublishEvent(eventbus.InitDeployScalerWorkflows, jsonData)
}

// Deploy the required workflows in Scaler.
func deployScalerWorkflows(channel eventbus.EventChannel, _ *eventbus.EventBus) {
	event := <-channel
	log.WithField(
		"event", event.Name,
	).Debug("received event in deployScalerWorkflows")
	defer event.Done()

	data := EventData{}
	if eventData, ok := event.Data.([]byte); ok {
		_ = json.Unmarshal(eventData, &data)
	} else {
		log.Error("event data not []byte format")
		return
	}

	jsonBody, _ := json.Marshal(map[string]string{"status": "DEPLOYED"})

	for _, workflow := range data.WorkflowsToDeploy {
		// request
		// PATCH {{baseUrl}}/api/integration/v2/workflows/{id}/
		request, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodPatch,
			fmt.Sprintf(
				"%s/%s/%s",
				data.ScalerHost,
				"api/integration/v2/workflows",
				workflow.ID,
			),
			bytes.NewBuffer(jsonBody),
		)
		if err != nil {
			log.Error("failed to create workflow deployment request: ", err)
			continue
		}
		log.WithFields(log.Fields{
			"method":   request.Method,
			"url":      request.URL,
			"workflow": workflow,
		}).Debug("created workflow deployment request")
		request.Header.Set(headers.Accept, "application/json")
		request.Header.Set(headers.Authorization, data.AuthHeader)
		request.Header.Set(headers.ContentType, "application/json")

		// response
		log.WithFields(log.Fields{
			"id":   workflow.ID,
			"name": workflow.Name,
		}).Info("sending workflow deployment request")
		//nolint:gomnd // TODO: Externalize constant value in config file.
		client := &http.Client{Timeout: time.Second * 5}
		response, err := client.Do(request)
		if err != nil {
			log.Error("failed to send workflow deployment request: ", err)
			return
		}
		defer func() { _ = response.Body.Close() }()

		// process
		if response.StatusCode >= http.StatusBadRequest {
			body, _ := io.ReadAll(response.Body)
			log.WithFields(log.Fields{
				"id":   workflow.ID,
				"name": workflow.Name,
			}).Errorf(
				"received non-ok HTTP status deploying workflow: [%d]:%s",
				response.StatusCode,
				string(body),
			)
			continue
		}
		log.WithFields(log.Fields{
			"id":   workflow.ID,
			"name": workflow.Name,
		}).Info("workflow deployed successfully")
	}

	log.Info("completed Scaler workflow deployment")
}

// getBasicAuthEncoding returns HTTP Basic auth encoding for
// the given user and password.
func getBasicAuthEncoding(user string, password string) string {
	return base64.StdEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%s:%s", user, password)),
	)
}

// Returns a count of workflows in Scaler.
func getScalerWorkflowCount(scalerHost string, authHeader string) int {
	var workflowCount int

	workflows, err := getScalerWorkflows(scalerHost, authHeader)
	if err != nil {
		log.Error("failed to get workflow count: ", err)
	} else {
		workflowCount = len(workflows)
	}

	return workflowCount
}

// Returns a Workflow slice representing the workflows in Scaler.
func getScalerWorkflows(scalerHost string, authHeader string) ([]Workflow, error) {
	// request
	// GET {{baseUrl}}/api/integration/v2/workflows/
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/%s", scalerHost, "api/integration/v2/workflows"),
		nil,
	)
	if err != nil {
		log.Error("failed to create list workflows request: ", err)
		return []Workflow{}, err
	}
	log.WithFields(log.Fields{
		"method": request.Method,
		"url":    request.URL,
	}).Debug("created list workflows request")
	request.Header.Set(headers.Authorization, authHeader)

	// response
	log.Debug("sending list workflows request")
	//nolint:gomnd // TODO: Externalize constant value in config file.
	client := &http.Client{Timeout: time.Second * 5}
	response, err := client.Do(request)
	if err != nil {
		log.Error("failed to send list workflows request: ", err)
		return []Workflow{}, err
	}
	defer func() { _ = response.Body.Close }()

	// process
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode >= http.StatusBadRequest {
		log.WithField("responseBody", string(body)).Errorf(
			"received non-ok HTTP status listing workflows: [%d]:%s",
			response.StatusCode,
			string(body),
		)
		return []Workflow{}, errors.New(string(body))
	}

	// Deserialize the JSON response
	workflowsResponse := WorkflowsResponse{}
	log.Debug("received list of workflows")
	if err = json.Unmarshal(body, &workflowsResponse); err != nil {
		log.Error("failed to process JSON in list workflows response:", err)
		return []Workflow{}, err
	}

	return workflowsResponse.Workflows, nil
}

// newFileUploadRequest is a helper for uploading files via HTTP.
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
	_, _ = io.Copy(part, file)

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, uri, body)
	req.Header.Set(headers.ContentType, writer.FormDataContentType())
	return req, err
}
