// sentiric-user-service/internal/logger/events.go
package logger

// SUTS v4.0 Standard Event IDs for user-service
// [ARCH-COMPLIANCE] Eksik olan tüm sistem ve veritabanı olayları registry'ye eklendi.
const (
	EventSystemStartup  = "SYSTEM_STARTUP"
	EventSystemShutdown = "SYSTEM_SHUTDOWN"

	// Sunucu Olayları
	EventGrpcServerStart = "GRPC_SERVER_START"
	EventGrpcServerFail  = "GRPC_SERVER_FAIL"
	EventGrpcServerStop  = "GRPC_SERVER_STOP"
	EventHttpServerStart = "HTTP_SERVER_START"
	EventHttpServerFail  = "HTTP_SERVER_FAIL"
	EventHttpServerStop  = "HTTP_SERVER_STOP"
	EventTlsLoadFail     = "TLS_LOAD_FAILED"

	// Veritabanı Olayları
	EventDatabaseConnSuccess = "DB_CONN_SUCCESS"
	EventDatabaseConnFailed  = "DB_CONN_FAILED"
	EventDatabaseError       = "DB_ERROR"

	// Gelen İstek Olayları
	EventGrpcRequest = "GRPC_REQUEST_RECEIVED"

	// Kullanıcı ve Ajan Olayları
	EventUserLookup        = "USER_LOOKUP"
	EventUserLookupFailed  = "USER_LOOKUP_FAILED"
	EventUserCreated       = "USER_CREATED"
	EventUserUpdated       = "USER_UPDATED"
	EventUserConflict      = "USER_CREATION_CONFLICT"
	EventUserCreationFail  = "USER_CREATION_FAIL"
	EventAgentProfileError = "AGENT_PROFILE_ERROR"

	// SIP Kimlik Olayları
	EventSipAuthAttempt  = "SIP_AUTH_ATTEMPT"
	EventSipAuthSuccess  = "SIP_AUTH_SUCCESS"
	EventSipAuthFailure  = "SIP_AUTH_FAILURE"
	EventSipCredCreated  = "SIP_CREDENTIAL_CREATED"
	EventSipCredConflict = "SIP_CREDENTIAL_CONFLICT"
	EventSipCredDeleted  = "SIP_CREDENTIAL_DELETED"
)
