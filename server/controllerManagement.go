package server

var controllerBeans []ControllerBean

type ControllerBean interface {
	GetControllerPath() string
}

func RegisterController(bean ControllerBean) {
	controllerBeans = append(controllerBeans, bean)
}

func getControllerBeans() []ControllerBean {
	return controllerBeans
}
