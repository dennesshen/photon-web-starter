package server

import (
	"reflect"

	"github.com/gofiber/fiber/v2"
)

func Assemble(ctx *fiber.Ctx, field *reflect.StructField) (values []reflect.Value, err error) {
	funcType := field.Type
	params := make([]reflect.Value, funcType.NumIn())

	// TODO 處理非指針或struct的參數
	for i := 0; i < funcType.NumIn(); i++ {

		paramType := funcType.In(i)
		var value reflect.Value

		switch paramType.Kind() {
		case reflect.Ptr:
			if paramType == reflect.TypeOf(ctx) {
				value = reflect.ValueOf(ctx)
			} else {
				pointerValue := reflect.New(paramType.Elem())
				if err = injectValue(ctx, pointerValue); err != nil {
					return nil, err
				}
				value = pointerValue
			}
		case reflect.Struct:
			pointerValue := reflect.New(paramType)
			value = pointerValue.Elem()
			if err = injectValue(ctx, pointerValue); err != nil {
				return
			}
		}

		params[i] = value

	}
	return params, nil
}

func injectValue(ctx *fiber.Ctx, pointerValue reflect.Value) (err error) {
	t := pointerValue.Elem().Type()
	types := findKind(t)
	for _, kind := range types {
		switch kind {
		case reflect.TypeOf(RequestBody{}):
			err = ctx.BodyParser(pointerValue.Interface())
			return
		case reflect.TypeOf(QueryParam{}):
			err = ctx.QueryParser(pointerValue.Interface())
			return
		case reflect.TypeOf(PathParam{}):
			// TODO 處理PathParam注入
		case reflect.TypeOf(Header{}):
			// TODO 處理Header注入
		}
	}
	return
}

// 尋找嵌套類型 RequestBody, QueryParam , PathParam, HeaderParam
func findKind(t reflect.Type) (types []reflect.Type) {
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Anonymous {
			types = append(types, t.Field(i).Type)
		}
	}
	return
}
