package handlers

type Container struct {
	AppStart         *AppStartHandler
	AppStop          *AppStopHandler
	CommandProcessor *CommandProcessorHandler
}
