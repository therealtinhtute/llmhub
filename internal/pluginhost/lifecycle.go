package pluginhost

import "github.com/therealtinhtute/llmhub/sdk/api/handlers"

type RequestLifecyclePlugin = handlers.RequestLifecyclePlugin
type RequestLifecycleRequest = handlers.RequestLifecycleRequest
type RequestLifecycleResponse = handlers.RequestLifecycleResponse
type RequestLifecycleDecision = handlers.RequestLifecycleDecision
type RequestLifecycleTermination = handlers.RequestLifecycleTermination
type SafeStreamEmitter = handlers.SafeStreamEmitter
type StreamEmitResult = handlers.StreamEmitResult

var NewSafeStreamEmitter = handlers.NewSafeStreamEmitter
