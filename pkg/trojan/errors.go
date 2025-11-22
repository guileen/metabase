package trojan

import "fmt"

// TrojanError represents a Trojan-specific error
type TrojanError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

func (e *TrojanError) Error() string {
	return fmt.Sprintf("trojan: %s - %s", e.Type, e.Message)
}

// Common Trojan errors
var (
	ErrServerNotStarted     = &TrojanError{Type: "server_not_started", Message: "Trojan server is not running", Code: 5001}
	ErrServerAlreadyRunning = &TrojanError{Type: "server_already_running", Message: "Trojan server is already running", Code: 5002}
	ErrInvalidConfig        = &TrojanError{Type: "invalid_config", Message: "Invalid configuration", Code: 5003}
	ErrClientNotFound       = &TrojanError{Type: "client_not_found", Message: "Client not found", Code: 5004}
	ErrClientExists         = &TrojanError{Type: "client_exists", Message: "Client already exists", Code: 5005}
	ErrConnectionFailed     = &TrojanError{Type: "connection_failed", Message: "Failed to establish connection", Code: 5006}
	ErrTLSHandshakeFailed   = &TrojanError{Type: "tls_handshake_failed", Message: "TLS handshake failed", Code: 5007}
	ErrInvalidPassword      = &TrojanError{Type: "invalid_password", Message: "Invalid authentication password", Code: 5008}
	ErrClientLimitReached   = &TrojanError{Type: "client_limit_reached", Message: "Maximum client limit reached", Code: 5009}
	ErrDataLimitExceeded    = &TrojanError{Type: "data_limit_exceeded", Message: "Client data limit exceeded", Code: 5010}
	ErrClientExpired        = &TrojanError{Type: "client_expired", Message: "Client access has expired", Code: 5011}
	ErrIPNotAllowed         = &TrojanError{Type: "ip_not_allowed", Message: "Client IP is not in whitelist", Code: 5012}
)

// NewTrojanError creates a new Trojan error
func NewTrojanError(errorType, message string, code ...int) *TrojanError {
	err := &TrojanError{
		Type:    errorType,
		Message: message,
	}
	if len(code) > 0 {
		err.Code = code[0]
	}
	return err
}
