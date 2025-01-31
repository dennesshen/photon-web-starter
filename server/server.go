package server

import (
	"context"
	"errors"
	"fmt"
	photonCoreStarter "github.com/dennesshen/photon-core-starter"
	"log/slog"
	"reflect"
	"slices"
	"sort"
	"strings"
	
	"github.com/dennesshen/photon-core-starter/bean"
	"github.com/dennesshen/photon-core-starter/log"
	"github.com/dlclark/regexp2"
	"github.com/gofiber/fiber/v2"
)

const (
	API        = "api"
	MIDDLEWARE = "middleware"
)

type apiInstance struct {
	Path         string
	Method       string
	Middleware   []fiber.Handler
	FinalHandler fiber.Handler
}

var server = fiber.New()

func init() {
	server.Use(func(ctx *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				log.Logger().Error(ctx.Context(), "panic:", "error", r)
				_ = ctx.Status(500).SendString("Internal Server Error")
			}
		}()
		return ctx.Next()
	})
	bean.RegisterBeanPtr(server)
}

func RunServer(ctx context.Context) error {
	setupGlobalMiddleware(server)
	setupControllers(server)
	
	port := config.Server.Port
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Logger().Error(ctx, "server panic", "error", r)
				ctx.Value(photonCoreStarter.ContextSendSign{}).(func())()
			}
		}()
		err := server.Listen(":" + port)
		if err != nil {
			ctx.Value(photonCoreStarter.ContextSendSign{}).(func())()
		}
	}()
	return nil
}

func ShutdownServer(context context.Context) error {
	log.Logger().Info(context, "server is shutting down")
	err := server.Shutdown()
	if err != nil {
		return err
	}
	return nil
}

func setupControllers(server *fiber.App) {
	contextPath := config.Server.ContextPath
	if err := checkPath(contextPath); err != nil {
		msg := fmt.Sprintf("[web-starter]context path is invalid, path: %s", contextPath)
		panic(msg)
	}
	
	wholeApiInstance := make([]apiInstance, 0)
	for _, controller := range getControllerBeans() {
		slog.Info("Registering controller:", "path", controller.GetControllerPath())
		controllerPath := controller.GetControllerPath()
		if err := checkPath(controllerPath); err != nil {
			msg := fmt.Sprintf("[web-starter]controller path is invalid, path: %s", controllerPath)
			panic(msg)
		}
		controllerValue := reflect.ValueOf(controller).Elem()
		controllerType := controllerValue.Type()
		
		var apiIndex []int
		var middlewareIndex []int
		for i := range controllerType.NumField() {
			
			field := controllerType.Field(i)
			kind := field.Tag.Get("kind")
			switch strings.ToLower(kind) {
			case API:
				apiIndex = append(apiIndex, i)
			case MIDDLEWARE:
				middlewareIndex = append(middlewareIndex, i)
			}
		}
		apiList := make([]apiInstance, 0)
		
		for _, i := range apiIndex {
			field := controllerType.Field(i)
			valueField := controllerValue.Field(i)
			setupApi(&apiList, &field, &valueField)
		}
		
		for _, i := range middlewareIndex {
			field := controllerType.Field(i)
			valueField := controllerValue.Field(i)
			setupMiddleware(&apiList, &field, &valueField)
		}
		
		rootPath := ""
		if contextPath != "" && contextPath != "/" {
			rootPath = contextPath
		}
		if controllerPath != "" && controllerPath != "/" {
			rootPath = rootPath + controllerPath
		}
		for i := range apiList {
			apiList[i].Path = rootPath + apiList[i].Path
		}
		wholeApiInstance = append(wholeApiInstance, apiList...)
	}
	
	apiPathList := make([]string, 0)
	for i := range wholeApiInstance {
		api := wholeApiInstance[i]
		
		if slices.Contains(apiPathList, api.Path) {
			msg := fmt.Sprintf("[web-starter]api path %s is duplicated", api.Path)
			panic(msg)
		}
		handlers := append(api.Middleware, api.FinalHandler)
		server.Add(api.Method, api.Path, handlers...)
		apiPathList = append(apiPathList, api.Path)
	}
}

func setupMiddleware(apiList *[]apiInstance, field *reflect.StructField, valueField *reflect.Value) {
	handler, ok := valueField.Interface().(fiber.Handler)
	if !ok || handler == nil {
		return
	}
	
	path := field.Tag.Get("path")
	prefix := field.Tag.Get("prefix")
	regex := field.Tag.Get("regex")
	
	if path != "" {
		for i := range *apiList {
			if (*apiList)[i].Path == path {
				(*apiList)[i].Middleware = append((*apiList)[i].Middleware, handler)
			}
		}
		return
	}
	regexStr := regex
	if prefix != "" {
		regexStr = fmt.Sprintf("^%s", prefix)
	}
	
	if regexStr == "" {
		msg := fmt.Sprintf("[web-starter]middleware %s : path, prefix or regex should not all be empty", field.Name)
		panic(msg)
	}
	
	pattern, err := regexp2.Compile(regexStr, 0)
	if err != nil {
		msg := fmt.Sprintf("[web-starter]middleware %s : regex is invalid, regex pattern: %s", field.Name, regexStr)
		panic(msg)
	}
	for i := range *apiList {
		if match, _ := pattern.MatchString((*apiList)[i].Path); match {
			(*apiList)[i].Middleware = append((*apiList)[i].Middleware, handler)
		}
	}
}

func setupApi(apiList *[]apiInstance, field *reflect.StructField, valueField *reflect.Value) {
	path := field.Tag.Get("path")
	method := strings.ToUpper(field.Tag.Get("method"))
	var finalHandler fiber.Handler
	
	if method == "" || path == "" {
		return
	}
	
	if handler, ok := valueField.Interface().(fiber.Handler); ok {
		if handler == nil {
			msg := fmt.Sprintf("[web-starter]function %s : fiber handler should not be nil", field.Name)
			panic(msg)
		}
		finalHandler = handler
		
	} else if field.Type.Kind() == reflect.Func {
		
		if valueField.IsNil() {
			msg := fmt.Sprintf("[web-starter]function %s : function should not be nil", field.Name)
			panic(msg)
		}
		if field.Type.NumOut() != 2 {
			msg := fmt.Sprintf("[web-starter]function %s : must have 2 return values", field.Name)
			panic(msg)
		}
		errType := reflect.TypeOf((*error)(nil)).Elem()
		if field.Type.Out(1) != errType {
			msg := fmt.Sprintf("[web-starter]function %s : second return value must be error", field.Name)
			panic(msg)
		}
		
		finalHandler = func(ctx *fiber.Ctx) error {
			params, err := Assemble(ctx, field)
			if err != nil {
				return err
			}
			if err = Validate(params); err != nil {
				return err
			}
			
			results := valueField.Call(params)
			if err := results[1]; err.IsNil() {
				return ctx.JSON(results[0].Interface())
			}
			return results[1].Interface().(error)
		}
	}
	
	if finalHandler == nil {
		msg := fmt.Sprintf("[web-starter]function %s : handler type is unknown, not supported", field.Name)
		panic(msg)
	}
	
	*apiList = append(*apiList, apiInstance{
		Path:         path,
		Method:       method,
		Middleware:   make([]fiber.Handler, 0),
		FinalHandler: finalHandler,
	})
}

func setupGlobalMiddleware(server *fiber.App) {
	middlewares := getGlobalMiddleware()
	sort.Slice(middlewares, func(i, j int) bool {
		return middlewares[i].GetPriority() <= middlewares[j].GetPriority()
	})
	
	for _, middleware := range middlewares {
		server.Use(middleware.GetPathPrefix(), middleware.GetMiddleware())
	}
}

func checkPath(path string) error {
	if path == "" || path == "/" {
		return nil
	}
	reg := regexp2.MustCompile(`^/.+[^/]$`, 0)
	match, _ := reg.MatchString(path)
	if !match {
		return errors.New("path is invalid")
	}
	return nil
}
