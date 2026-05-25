// Пакет jsonrpc реализует JSON-RPC 2.0 протокол поверх HTTP.
// Спецификация: https://www.jsonrpc.org/specification
package jsonrpc

// Стандартные коды ошибок JSON-RPC 2.0 (диапазон -32768 .. -32000 зарезервирован).
// Коды от -32099 до -32000 — для кастомных серверных ошибок приложения.
const (
	CodeParseError     = -32700 // Невалидный JSON в теле запроса
	CodeInvalidRequest = -32600 // Запрос не соответствует спецификации JSON-RPC 2.0
	CodeMethodNotFound = -32601 // Метод с таким именем не зарегистрирован
	CodeInvalidParams  = -32602 // Параметры метода невалидны
	CodeInternalError  = -32603 // Внутренняя ошибка сервера

	// Кастомные коды приложения:
	CodeUnauthorized = -32001 // Сессия не найдена или истекла
	CodeConflict     = -32002 // Конфликт состояния (например, запись уже идёт)
	CodeNotFound     = -32003 // Ресурс не найден
)

// Error — объект ошибки JSON-RPC 2.0, отправляемый в поле "error" ответа.
// Реализует интерфейс error, чтобы можно было использовать как обычную Go-ошибку.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Message }

// Преднастроенные ошибки для типичных ситуаций.
// Используй их вместо создания новых, если нет особой причины.
var (
	ErrParseError     = &Error{CodeParseError, "parse error"}
	ErrInvalidRequest = &Error{CodeInvalidRequest, "invalid request"}
	ErrMethodNotFound = &Error{CodeMethodNotFound, "method not found"}
	ErrInvalidParams  = &Error{CodeInvalidParams, "invalid params"}
	ErrInternalError  = &Error{CodeInternalError, "internal error"}
	ErrUnauthorized   = &Error{CodeUnauthorized, "unauthorized"}
)

// InvalidParams возвращает ошибку -32602 с детальным сообщением.
func InvalidParams(msg string) *Error {
	return &Error{Code: CodeInvalidParams, Message: msg}
}

// Internal возвращает ошибку -32603 с детальным сообщением.
// Внимание: в продакшне не передавай сюда внутренние детали — логируй их,
// а клиенту отправляй generic-сообщение.
func Internal(msg string) *Error {
	return &Error{Code: CodeInternalError, Message: msg}
}
