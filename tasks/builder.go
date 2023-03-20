package tasks

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/cloudradar-monitoring/tacoscript/conv"
	"github.com/cloudradar-monitoring/tacoscript/utils"
)

const (
	UnsetKeyword = "!UNSET!"
)

type Builder interface {
	Build(typeName, path string, params interface{}) (Task, error)
}

type BuildRouter struct {
	Builders map[string]Builder
}

type taskFieldParseFn func(t Task, path string, val interface{}) error

type taskField struct {
	parseFn   taskFieldParseFn
	fieldName string
}

type taskFieldsParserConfig map[string]taskField

func NewBuilderRouter(builders map[string]Builder) BuildRouter {
	return BuildRouter{
		Builders: builders,
	}
}

func (br BuildRouter) Build(typeName, path string, params interface{}) (Task, error) {
	builder, ok := br.Builders[typeName]
	if !ok {
		return nil, fmt.Errorf("no builders registered for task type '%s'", typeName)
	}

	return builder.Build(typeName, path, params)
}

func Build(
	typeName, path string, inputFields interface{}, outputTask Task, taskFields taskFieldsParserConfig) (
	errs *utils.Errors) {
	logrus.Debugf("Parsing task %s, %s", typeName, path)
	errs = &utils.Errors{}

	mapper := outputTask.GetMapper()
	mapper.BuildFieldMap(outputTask)

	var tracker FieldStatusTracker
	taskWithTracker, hasTracker := outputTask.(TaskWithTracker)
	if hasTracker {
		tracker = taskWithTracker.GetTracker()
	}

	outputTaskValues := reflect.Indirect(reflect.ValueOf(outputTask))

	for _, inputItem := range inputFields.([]interface{}) {
		row := inputItem.(yaml.MapSlice)[0]

		inputKey := row.Key.(string)
		inputVal := row.Value

		fieldName := mapper.GetFieldName(inputKey)

		if fieldName != "" {
			if hasTracker {
				tracker.SetFieldStatus(fieldName, FieldStatus{})

				// when unsetting a field then no need to parse value etc. just mark to clear and then
				// continue to the next field.
				if inputVal == UnsetKeyword && !sharedField(inputKey) && taskWithTracker.IsChangeField(fieldName) {
					err := tracker.SetClear(fieldName)
					if err != nil {
						errs.Add(errWithField(err, inputKey))
					}
					continue
				}
			}

			outputFieldVal := outputTaskValues.FieldByName(fieldName)

			// if empty field then we didn't find the field matching the name
			if outputFieldVal == (reflect.Value{}) {
				errs.Add(errWithField(errors.New("field not found in task struct"), inputKey))
				continue
			}

			// if exists in the struct then we can use reflection to parse the value
			err := updateField(outputFieldVal, inputVal)
			if err != nil {
				errs.Add(errWithField(err, inputKey))
				continue
			}

			if hasTracker {
				if !sharedField(inputKey) && taskWithTracker.IsChangeField(fieldName) {
					err = tracker.SetHasNewValue(fieldName)
					if err != nil {
						errs.Add(errWithField(err, inputKey))
						continue
					}
				}
			}

			continue
		}

		// didn't exist in the tracker so we'll be parsing manually
		if taskFields != nil {
			taskParam, ok := taskFields[inputKey]
			if !ok {
				continue
			}

			err := taskParam.parseFn(outputTask, path, inputVal)
			if err != nil {
				errs.Add(errWithField(err, inputKey))
				continue
			}
		}
	}

	return errs
}

func errWithField(err error, field string) (updatedErr error) {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s", err, field)
}

func updateField(outputFieldVal reflect.Value, inputVal any) (err error) {
	switch outputFieldVal.Kind() { //nolint:exhaustive // default handler
	case reflect.Bool:
		valBool, err := conv.ConvertToBool(inputVal)
		if err != nil {
			return err
		}
		outputFieldVal.SetBool(valBool)
	case reflect.String:
		outputFieldVal.SetString(fmt.Sprint(inputVal))
	case reflect.Int:
		valInt, err := conv.ConvertToInt(inputVal)
		if err != nil {
			return err
		}
		outputFieldVal.SetInt(int64(valInt))
	case reflect.Slice:
		switch inputVal.(type) {
		case string:
			setSliceWithSingleElement(outputFieldVal, inputVal)
		default:
			err = setSliceWithElements(outputFieldVal, inputVal)
			if err != nil {
				return err
			}
		}
	default:
		return errors.New("field type not supported")
	}

	return nil
}

func setSliceWithSingleElement(outputFieldVal reflect.Value, inputVal any) {
	newOutputSliceValue := reflect.MakeSlice(outputFieldVal.Type(), 0, 1)
	inputItem := fmt.Sprint(inputVal)
	newSliceValue := reflect.ValueOf(inputItem)
	newOutputSliceValue = reflect.Append(newOutputSliceValue, newSliceValue)
	outputFieldVal.Set(newOutputSliceValue)
}

func setSliceWithElements(outputFieldVal reflect.Value, inputVal any) (err error) {
	inputItems, err := conv.ConvertToValues(inputVal)
	if err != nil {
		return err
	}
	newOutputSliceValue := reflect.MakeSlice(outputFieldVal.Type(), 0, len(inputItems))
	for _, inputItem := range inputItems {
		newSliceValue := reflect.ValueOf(inputItem)
		newOutputSliceValue = reflect.Append(newOutputSliceValue, newSliceValue)
	}
	outputFieldVal.Set(newOutputSliceValue)

	return nil
}
