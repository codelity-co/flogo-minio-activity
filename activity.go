package minio

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data"
	"github.com/project-flogo/core/data/mapper"
	"github.com/project-flogo/core/data/property"
	"github.com/project-flogo/core/data/resolve"

	"github.com/jeremywohl/flatten"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/thoas/go-funk"
)

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})
var resolver = resolve.NewCompositeResolver(map[string]resolve.Resolver{
	".":        &resolve.ScopeResolver{},
	"env":      &resolve.EnvResolver{},
	"property": &property.Resolver{},
	"loop":     &resolve.LoopResolver{},
})

func init() {
	_ = activity.Register(&Activity{}, New)
}

//New function is used as activity factory
func New(ctx activity.InitContext) (activity.Activity, error) {

	var (
		minioClient *minio.Client
	)

	s := &Settings{}
	err := s.FromMap(ctx.Settings())
	if err != nil {
		return nil, err
	}

	ctx.Logger().Debugf("Setting: %v", s)

	// Resolving settings
	if s.MethodOptions != nil {
		ctx.Logger().Debugf("methodOpitons settings being resolved: %v", s.MethodOptions)
		methodOptions, err := resolveObject(s.MethodOptions)
		if err != nil {
			return nil, err
		}
		s.MethodOptions = methodOptions
		ctx.Logger().Debugf("methodOpitons settings resolved: %v", s.MethodOptions)
	}

	// Resolving settings
	if s.SslConfig != nil {
		ctx.Logger().Debugf("sslConfig settings being resolved: %v", s.SslConfig)
		sslConfig, err := resolveObject(s.SslConfig)
		if err != nil {
			return nil, err
		}
		s.SslConfig = sslConfig
		ctx.Logger().Debugf("methodOpitons settings resolved: %v", s.SslConfig)
	}

	minioOptions := &minio.Options{
		Creds:  credentials.NewStaticV4(s.AccessKey, s.SecretKey, ""),
		Secure: s.EnableSsl,
	}

	if s.EnableSsl && len(s.SslConfig["caFile"].(string)) > 0 {

		cert, err := tls.LoadX509KeyPair(s.SslConfig["certFile"].(string), s.SslConfig["keyFile"].(string))
		if err != nil {
			return nil, err
		}

		caCert, err := ioutil.ReadFile(s.SslConfig["caFile"].(string))
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		minioOptions.Transport = &http.Transport{
			MaxIdleConns:       int(s.SslConfig["maxIdleConns"].(int64)),
			IdleConnTimeout:    (time.Second * time.Duration(s.SslConfig["idleConnTimeout"].(int64))),
			DisableCompression: s.SslConfig["disableCompression"].(bool),
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				RootCAs:      caCertPool,
			},
		}
	}

	minioClient, err = minio.New(s.Endpoint, minioOptions)
	if err != nil {
		ctx.Logger().Errorf("MinIO connection error: %v", err)
		return nil, err
	}
	ctx.Logger().Debug("Got MinIO connection")

	act := &Activity{
		activitySettings: s,
		minioClient:      minioClient,
	}

	ctx.Logger().Debug("Finished New method of activity")
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

	logger.Debugf("a.activitySettings: %v", a.activitySettings)
	switch a.activitySettings.MethodName {

	case "BucketExists":
		return a.bucketExists(ctx, input)
	case "GetObject":
		return a.getObject(ctx, input)
	case "MakeBucket":
		return a.makeBucket(ctx, input)
	case "PutObject":
		return a.putObject(ctx, input)
	case "RemoveObject":
		return a.removeObject(ctx, input)
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

func (a *Activity) bucketExists(ctx activity.Context, input *Input) (bool, error) {
	var err error
	logger := ctx.Logger()

	logger.Debug("Call MinIO BucketExists method")
	isBucketExisting, err := a.minioClient.BucketExists(context.Background(), a.activitySettings.BucketName)
	if err != nil {
		logger.Errorf("Error in MinIO BucketExists method: %v", err)
		_ = a.OutputToContext(ctx, nil, err)
		return true, err
	}

	err = a.OutputToContext(ctx, map[string]interface{}{"exist": isBucketExisting}, nil)
	if err != nil {
		logger.Errorf("Error setting output object in context: %v", err)
		return true, err
	}

	return true, nil
}

func (a *Activity) getObject(ctx activity.Context, input *Input) (bool, error) {
	var err error
	logger := ctx.Logger()

	logger.Debug("Call MinIO GetObject method")
	minioObject, err := a.minioClient.GetObject(context.Background(), a.activitySettings.BucketName, input.ObjectName, minio.GetObjectOptions{})
	if err != nil {
		logger.Errorf("Error in MinIO GetObject method: %v", err)
		_ = a.OutputToContext(ctx, nil, err)
		return true, err
	}
	objectStat, err := minioObject.Stat()
	if err != nil {
		logger.Errorf("Error in MinIO GetObject method: %v", err)
		_ = a.OutputToContext(ctx, nil, err)
		return true, err
	}
	objectBytes := make([]byte, objectStat.Size)
	numberOfBytes, err := minioObject.Read(objectBytes)
	if err != nil {
		logger.Errorf("Error in MinIO GetObject method: %v", err)
		_ = a.OutputToContext(ctx, nil, err)
		return true, err
	}
	if int64(numberOfBytes) != objectStat.Size {
		err = errors.New("object size does not match")
		logger.Errorf("Error in MinIO GetObject method: %v", err)
		_ = a.OutputToContext(ctx, nil, err)
		return true, err
	}

	err = a.OutputToContext(ctx, map[string]interface{}{"data": string(objectBytes)}, nil)
	if err != nil {
		logger.Errorf("Error setting output object in context: %v", err)
		return true, err
	}

	return true, nil
}

func (a *Activity) makeBucket(ctx activity.Context, input *Input) (bool, error) {
	var err error
	logger := ctx.Logger()

	logger.Debug("Call MinIO BucketExists method")
	err = a.minioClient.MakeBucket(context.Background(), a.activitySettings.BucketName, minio.MakeBucketOptions{Region: a.activitySettings.Region})
	if err != nil {
		logger.Errorf("Error in MinIO MakeBucket method: %v", err)
		_ = a.OutputToContext(ctx, nil, err)
		return true, err
	}

	err = a.OutputToContext(ctx, map[string]interface{}{"created": true}, nil)
	if err != nil {
		logger.Errorf("Error setting output object in context: %v", err)
		return true, err
	}

	return true, nil
}

func (a *Activity) putObject(ctx activity.Context, input *Input) (bool, error) {

	var err error
	var dataBytes []byte
	logger := ctx.Logger()

	if input.Data == nil {
		err = errors.New("Data is nil")
		logger.Errorf("Error in MinIO PutObject method: %v", err)
		_ = a.OutputToContext(ctx, nil, err)
		return true, err
	}

	switch input.Format {
	case "JSON":
		dataBytes, err = json.Marshal(input.Data)
		if err != nil {
			logger.Errorf("Error marshalling input data into JSON: %v", err)
			_ = a.OutputToContext(ctx, nil, err)
			return true, err
		}

	case "CSV":
		// var dataMap map[string]interface{} = make(map[string]interface{})
		// // json.Unmarshal([]byte(input.Data.(string)), &dataMap)
		// json.Unmarshal([]byte(fmt.Sprintf("%v", input.Data)), &dataMap)
		flattenedMap, err := flatten.Flatten(input.Data.(map[string]interface{}), "", flatten.DotStyle)
		if err != nil {
			logger.Errorf("Error flattening input data: %v", err)
			_ = a.OutputToContext(ctx, nil, err)
			return true, err
		}
		logger.Debugf("flattenedMap: %v", flattenedMap)

		var csvHeaders []string = []string{}
		for _, value := range funk.Keys(flattenedMap).([]string) {
			csvHeaders = append(csvHeaders, fmt.Sprintf("%q", value))
		}
		var csvValues []string = []string{}
		for _, value := range funk.Values(flattenedMap).([]interface{}) {
			switch v := value.(type) {
			case string:
				csvValues = append(csvValues, fmt.Sprintf("%q", v))
			case []byte:
				csvValues = append(csvValues, fmt.Sprintf("%q", string(v)))
			case bool:
				csvValues = append(csvValues, fmt.Sprintf("%t", v))
			default:
				csvValues = append(csvValues, fmt.Sprintf("%v", v))
			}
		}
		dataString := strings.Join([]string{strings.Join(csvHeaders, ","), strings.Join(csvValues, ",")}, "\n")
		logger.Debugf("dataString: %v", dataString)
		dataBytes = []byte(dataString)

	}

	logger.Debug("Call MinIO PutObject method")
	numberOfBytes, err := a.minioClient.PutObject(context.Background(), a.activitySettings.BucketName, input.ObjectName, bytes.NewReader(dataBytes), int64(len(dataBytes)), minio.PutObjectOptions{})
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

	return true, nil
}

func (a *Activity) removeObject(ctx activity.Context, input *Input) (bool, error) {
	var err error
	logger := ctx.Logger()

	logger.Debug("Call MinIO GetObject method")
	err = a.minioClient.RemoveObject(context.Background(), a.activitySettings.BucketName, input.ObjectName, minio.RemoveObjectOptions{})
	if err != nil {
		logger.Errorf("Error in MinIO RemoveObject method: %v", err)
		_ = a.OutputToContext(ctx, nil, err)
		return true, err
	}

	err = a.OutputToContext(ctx, map[string]interface{}{"removed": true}, nil)
	if err != nil {
		logger.Errorf("Error setting output object in context: %v", err)
		return true, err
	}

	return true, nil
}

func resolveObject(object map[string]interface{}) (map[string]interface{}, error) {
	var err error

	mapperFactory := mapper.NewFactory(resolver)
	valuesMapper, err := mapperFactory.NewMapper(object)
	if err != nil {
		return nil, err
	}

	objectValues, err := valuesMapper.Apply(data.NewSimpleScope(map[string]interface{}{}, nil))
	if err != nil {
		return nil, err
	}

	return objectValues, nil
}
