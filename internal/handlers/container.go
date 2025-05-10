package handlers

type Container struct {
	AppStart         *ApplicationStartHandler
	AppStop          *ApplicationStopHandler
	CommandProcessor *CommandProcessorHandler
}
