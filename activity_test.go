package minio

import (
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/support/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type MinioActivityTestSuite struct {
	suite.Suite
}

func TestMinioActivityTestSuite(t *testing.T) {
	suite.Run(t, new(MinioActivityTestSuite))
}

func (suite *MinioActivityTestSuite) SetupSuite() {
	command := exec.Command("docker", "start", "minio")
	err := command.Run()
	if err != nil {
		fmt.Println(err.Error())
		command := exec.Command("docker", "run", "-p", "9000:9000", "--name", "minio", "-d", "minio/minio", "server", "/data")
		err := command.Run()
		if err != nil {
			fmt.Println(err.Error())
			panic(err)
		}
		time.Sleep(10 * time.Second)
	}
}

func (suite *MinioActivityTestSuite) SetupTest() {}

func (suite *MinioActivityTestSuite) TestMinioActivity_Register() {

	ref := activity.GetRef(&Activity{})
	act := activity.Get(ref)

	assert.NotNil(suite.T(), act)
}

func (suite *MinioActivityTestSuite) TestMinioActivity_Settings() {
	t := suite.T()

	settings := &Settings{
		Endpoint:      "localhost:9000",
		AccessKey:     "minioadmin",
		SecretKey:     "minioadmin",
		EnableSsl:     false,
		BucketName:    "flogo",
		MethodName:    "PutObject",
		MethodOptions: nil,
	}

	iCtx := test.NewActivityInitContext(settings, nil)
	_, err := New(iCtx)
	assert.Nil(t, err)
}

func (suite *MinioActivityTestSuite) TestMinioActivity_PutObject() {
	t := suite.T()

	settings := &Settings{
		Endpoint:   "localhost:9000",
		AccessKey:  "minioadmin",
		SecretKey:  "minioadmin",
		EnableSsl:  false,
		BucketName: "flogo",
		MethodName: "BucketExists",
	}
	iCtx := test.NewActivityInitContext(settings, nil)
	act, err := New(iCtx)
	assert.Nil(t, err)

	tc := test.NewActivityContext(act.Metadata())
	done, err := act.Eval(tc)
	assert.Nil(t, err)
	assert.Truef(t, done, "Not Done")
	status := tc.GetOutput("status").(string)
	assert.Equal(t, "SUCCESS", status)

	result := tc.GetOutput("result").(map[string]interface{})
	if !result["exist"].(bool) {

		settings.MethodName = "MakeBucket"
		iCtx = test.NewActivityInitContext(settings, nil)
		act, err = New(iCtx)
		assert.Nil(t, err)

		tc = test.NewActivityContext(act.Metadata())
		done, err = act.Eval(tc)

		status := tc.GetOutput("status").(string)
		result := tc.GetOutput("result").(map[string]interface{})
		assert.Equal(t, "SUCCESS", status)
		assert.Truef(t, result["created"].(bool), "Bucket not created")
	}

	settings.MethodName = "PutObject"
	iCtx = test.NewActivityInitContext(settings, nil)
	act, err = New(iCtx)
	assert.Nil(t, err)

	tc = test.NewActivityContext(act.Metadata())
	tc.SetInput("objectName", "inbox/testing.json")
	tc.SetInput("format", "JSON")
	tc.SetInput("data", "{\"abc\": \"123\"}")
	done, err = act.Eval(tc)
	assert.Nil(t, err)
	assert.Truef(t, done, "Not Done")

	tc = test.NewActivityContext(act.Metadata())
	tc.SetInput("objectName", "inbox/testing.csv")
	tc.SetInput("format", "CSV")
	tc.SetInput("data", "{\"abc\": \"123\", \"bcd\": {\"cde\": \"123\", \"efg\": false}}")
	done, err = act.Eval(tc)
	assert.Nil(t, err)
	assert.Truef(t, done, "Not Done")
}
