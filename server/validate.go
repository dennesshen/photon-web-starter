package server

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type ValidError struct {
	msg string
}

func (e *ValidError) Error() string {
	return e.msg
}

func NewValidError(msg string) *ValidError {
	return &ValidError{msg: msg}
}

type SelfValidator interface {
	SelfValidate() error
}

// validate 採用 fiber 官方推薦的 github.com/go-playground/validator
var validate = validator.New()

func Validate(items []reflect.Value) (err error) {
	for _, item := range items {

		itemPtr, ok := toPointer(item)
		if !ok {
			continue
		}

		if !ifNeedValidate(itemPtr) {
			continue
		}

		var vErrs validator.ValidationErrors
		if ok := errors.As(validate.Struct(itemPtr.Interface()), &vErrs); ok {
			var errMsgs []string
			for _, vErr := range vErrs {
				var limitStr string
				if vErr.Param() != "" {
					limitStr = fmt.Sprintf("=%s", vErr.Param())
				}
				errMsgs = append(errMsgs, fmt.Sprintf(
					"[%s]: '%v' | Needs to implement '%s%s'",
					vErr.Field(),
					vErr.Value(),
					vErr.Tag(),
					limitStr,
				))
				return NewValidError(strings.Join(errMsgs, ", "))
			}
		}

		if sv, ok := itemPtr.Interface().(SelfValidator); ok {
			if err = sv.SelfValidate(); err != nil {
				return NewValidError(err.Error())
			}
		}

	}
	return
}

func toPointer(item reflect.Value) (pointer reflect.Value, ok bool) {
	if item.Kind() == reflect.Ptr {
		return item, true
	}

	// 如果是结构体类型，将其转换为指针类型
	if item.Kind() == reflect.Struct {
		ptrValue := reflect.New(item.Type())
		ptrValue.Elem().Set(item)
		return ptrValue, true
	}

	return pointer, false
}

func ifNeedValidate(value reflect.Value) (need bool) {
	if value.Type() == reflect.TypeOf(&fiber.Ctx{}) {
		return
	}

	t := value.Elem().Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous && f.Type == reflect.TypeOf(Valid{}) {
			return true
		}
	}
	return
}
