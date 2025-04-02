package parameters

import (
	"os"

	cmdcommon_types "github.com/kyma-project/cli.v3/internal/cmdcommon/types"
	"github.com/spf13/pflag"
)

type ConfigFieldType string

const (
	StringCustomType ConfigFieldType = "string"
	PathCustomType   ConfigFieldType = "path"
	IntCustomType    ConfigFieldType = "int"
	BoolCustomType   ConfigFieldType = "bool"
	// TODO: support other types e.g. float and stringArray
)

var (
	ValidTypes = []ConfigFieldType{
		StringCustomType,
		PathCustomType,
		IntCustomType,
		BoolCustomType,
	}
)

type Value interface {
	pflag.Value
	SetValue(string) error
	GetValue() interface{}
	GetPath() string
}

func NewTyped(paramType ConfigFieldType, resourcepath string) Value {
	switch paramType {
	case PathCustomType:
		return &pathValue{stringValue: &stringValue{path: resourcepath}}
	case IntCustomType:
		return &int64Value{path: resourcepath}
	case BoolCustomType:
		return &boolValue{path: resourcepath}
	default:
		return &stringValue{path: resourcepath}
	}
}

type boolValue struct {
	*cmdcommon_types.NullableBool
	path string
}

func (v *boolValue) GetValue() interface{} {
	if v.Value == nil {
		return nil
	}

	return *v.Value
}

func (v *boolValue) GetPath() string {
	return v.path
}

func (v *boolValue) SetValue(value string) error {
	return v.Set(value)
}

type int64Value struct {
	cmdcommon_types.NullableInt64
	path string
}

func (v *int64Value) GetValue() interface{} {
	if v.Value == nil {
		return nil
	}

	return *v.Value
}

func (v *int64Value) GetPath() string {
	return v.path
}

func (v *int64Value) SetValue(value string) error {
	return v.Set(value)
}

type stringValue struct {
	cmdcommon_types.NullableString
	path string
}

func (v *stringValue) GetValue() interface{} {
	if v.Value == nil {
		return nil
	}

	return *v.Value
}

func (sv *stringValue) GetPath() string {
	return sv.path
}

func (sv *stringValue) SetValue(value string) error {
	return sv.Set(value)
}

type pathValue struct {
	*stringValue
}

func (pv *pathValue) Set(value string) error {
	bytes, err := os.ReadFile(value)
	if err != nil {
		return err
	}

	return pv.SetValue(string(bytes))
}

func (pv *pathValue) SetValue(value string) error {
	return pv.stringValue.Set(value)
}
