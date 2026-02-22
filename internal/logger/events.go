// sentiric-user-service/internal/logger/events.go
package logger

// SUTS v4.0 Standard Event IDs for user-service
const (
	EventSystemStartup    = "SYSTEM_STARTUP"
	EventGrpcRequest      = "GRPC_REQUEST_RECEIVED"
	EventUserLookup       = "USER_LOOKUP"
	EventUserLookupFailed = "USER_LOOKUP_FAILED"
	EventUserCreated      = "USER_CREATED"
	EventUserUpdated      = "USER_UPDATED"
	EventUserConflict     = "USER_CREATION_CONFLICT"
	EventSipAuthAttempt   = "SIP_AUTH_ATTEMPT"
	EventSipAuthSuccess   = "SIP_AUTH_SUCCESS"
	EventSipAuthFailure   = "SIP_AUTH_FAILURE"
	EventSipCredCreated   = "SIP_CREDENTIAL_CREATED"
)
