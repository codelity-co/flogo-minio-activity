package minio

import (
	"github.com/project-flogo/core/data/coerce"
	"github.com/project-flogo/core/data/mapper"
	"github.com/project-flogo/core/data/property"
	"github.com/project-flogo/core/data/resolve"

)

var resolver = resolve.NewCompositeResolver(map[string]resolve.Resolver{
	".":        &resolve.ScopeResolver{},
	"env":      &resolve.EnvResolver{},
	"property": &property.Resolver{},
	"loop":     &resolve.LoopResolver{},
})

// Settings struct
type Settings struct {
	Endpoint string `md:"endpoint,required"`
	AccessKey string `md:"accessKey,required"`
	SecretKey string `md:"secretKey,required"`
	EnableSsl bool `md:"enableSsl"`
	BucketName string `md:"bucketName,required"`
	Region string `md:"region"`
	MethodName string `md:"methodName,required"` 
	MethodOptions map[string]interface{} `md:"methodOptions"`
}

// FromMap method of Settings
func (s *Settings) FromMap(values map[string]interface{}) error {

	var (
		err error
	)

	s.Endpoint, err = coerce.ToString(values["endpoint"])
	if err != nil {
		return err
	}

	s.AccessKey, err = coerce.ToString(values["accessKey"])
	if err != nil {
		return err
	}

	s.SecretKey, err = coerce.ToString(values["secretKey"])
	if err != nil {
		return err
	}

	s.EnableSsl, err = coerce.ToBool(values["enableSsl"])
	if err != nil {
		return err
	}

	s.BucketName, err = coerce.ToString(values["bucketName"])
	if err != nil {
		return err
	}

	s.Region, err = coerce.ToString(values["region"])
	if err != nil {
		return err
	}

	s.MethodName, err = coerce.ToString(values["methodName"])
	if err != nil {
		return err
	}

	mapperFactory := mapper.NewFactory(resolver)
	if values["methodOptions"] != nil {

		var methodOptions map[string]interface{}
		methodOptions, err = coerce.ToObject(values["methodName"])
		if err != nil {
			return err
		}

		var methodOptionsMapper mapper.Mapper
		methodOptionsMapper, err = mapperFactory.NewMapper(methodOptions)
		if err != nil {
			return err
		}

		var methodOptionsValue map[string]interface{}
		methodOptionsValue, err = methodOptionsMapper.Apply(nil)
		if err != nil {
			return err
		}

		s.MethodOptions = methodOptionsValue
	}

	return nil

}

// ToMap method of Settings
func (s *Settings) ToMap() map[string]interface{} {

	return map[string]interface{}{
		"endpoint": s.Endpoint,
		"accessKey":    s.AccessKey,
		"secretKey": s.SecretKey,
		"enableSsl": s.EnableSsl,
		"bucketName": s.BucketName,
		"region": s.Region,
		"methodName": s.MethodName,
		"methodOptions": s.MethodOptions,
	}

}

// Input struct
type Input struct {
	ObjectName string `md:"objectName,required"`
	Data interface{} `md:"data,required"`
}

// FromMap method of Input
func (r *Input) FromMap(values map[string]interface{}) error {
	var err error

	r.ObjectName, err = coerce.ToString(values["objectName"])
	if err != nil {
		return err
	}

	r.Data, err = coerce.ToAny(values["data"])
	if err != nil {
		return err
	}
	
	return nil
}

// ToMap method of Input
func (r *Input) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"objectName": r.ObjectName,
		"data": r.Data,
	}
}

// Output struct
type Output struct {
	Status string `md:"status,required"`
	Result map[string]interface{} `md:"result"`
}
 
// FromMap method of Output
func (o *Output) FromMap(values map[string]interface{}) error {
	var err error

	o.Status, err = coerce.ToString(values["status"])
	if err != nil {
		return err
	}

	o.Result, err = coerce.ToObject(values["result"])
	if err != nil {
		return err
	}

	return nil
}

// ToMap method of Output
func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"status": o.Status,
		"result": o.Result,
	}
}
