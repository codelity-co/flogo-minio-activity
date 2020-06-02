package minio

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/project-flogo/core/activity"

	"github.com/minio/minio-go/v6"
)

func init() {
	_ = activity.Register(&Activity{}, New) //activity.Register(&Activity{}, New) to create instances using factory method 'New'
}

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

//New function is used as activity factory
func New(ctx activity.InitContext) (activity.Activity, error) {

	logger := ctx.Logger()

	var (
		minioClient *minio.Client
	)

	s := &Settings{}
	err := s.FromMap(ctx.Settings())
	if err != nil {
		return nil, err
	}

	logger.Debugf("From Map Setting: %v", s)

	minioClient, err = minio.New(s.Endpoint, s.AccessKey, s.SecretKey, s.EnableSsl)
	if err != nil {
		logger.Errorf("MinIO connection error: %v", err)
		return nil, err
	}
	logger.Debug("Got MinIO connection")


	act := &Activity{
		activitySettings: s,
		minioClient:      minioClient,
	}

	logger.Debug("Finished New method of activity")
	return act, nil
}

// Activity is an sample Activity that can be used as a base to create a custom activity
type Activity struct {
	activitySettings *Settings
	minioClient      *minio.Client
}

// Metadata returns the activity's metadata
func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

// Eval implements api.Activity.Eval - Logs the Message
func (a *Activity) Eval(ctx activity.Context) (bool, error) {

	var err error
	logger := ctx.Logger()

	logger.Debug("Running Eval method of activity...")
	input := &Input{}
	logger.Debug("Getting Input object from context...")
	err = ctx.GetInputObject(input)
	if err != nil {
		logger.Errorf("Error getting Input object: %v", err)
		_ = a.OutputToContext(ctx, nil, err)
		return true, err
	}
	logger.Debug("Got Input object successfully")
	logger.Debugf("Input: %v", input)

	var dataBytes []byte
	if input.Data != nil {
		dataBytes, err = json.Marshal(input.Data)
		if err != nil {
			logger.Errorf("Error marshalling input data: %v", err)
			_ = a.OutputToContext(ctx, nil, err)
			return true, err
		}
	}

	logger.Debugf("a.activitySettings: %v", a.activitySettings)
	switch a.activitySettings.MethodName {
	case "PutObject":
		logger.Debug("Call MinIO PutObject method")
		numberOfBytes, err := a.minioClient.PutObject(a.activitySettings.BucketName, input.ObjectName, bytes.NewReader(dataBytes), int64(len(dataBytes)), minio.PutObjectOptions{})
		if err != nil {
			logger.Errorf("Error in MinIO PutObject method: %v", err)
			_ = a.OutputToContext(ctx, nil, err)
			return true, err
		}
		err = a.OutputToContext(ctx, map[string]interface{}{"bytes": numberOfBytes}, nil)
		if err != nil {
			logger.Errorf("Error setting output object in context: %v", err)
			return true, err
		}
	}

	return true, nil
}

// Cleanup method of Activity
func (a *Activity) Cleanup(ctx activity.Context) error {
	return nil
}

// OutputToContext method of Activity
func (a *Activity) OutputToContext(ctx activity.Context, result map[string]interface{}, err error) error {
	logger := ctx.Logger()
	logger.Debug("Createing Ouptut struct...")
	var output *Output
	if err != nil {
		output = &Output{Status: "ERROR", Result: map[string]interface{}{"errorMessage": fmt.Sprintf("%v", err)}}
	} else {
		output = &Output{Status: "SUCCESS", Result: result}
	}
	logger.Debug("Setting output object in context...")
	return ctx.SetOutputObject(output)
}
