package response

type WebResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func SuccessResponse(data any, message ...string) *WebResponse {
	resMessage := "Success"
	if len(message) > 0 {
		resMessage = message[0]
	}
	return &WebResponse{
		Success: true,
		Message: resMessage,
		Data:    data,
	}
}

func ErrorResponse(message ...string) *WebResponse {
	resMessage := "Error"
	if len(message) > 0 {
		resMessage = message[0]
	}
	return &WebResponse{
		Success: false,
		Message: resMessage,
		Data:    nil,
	}
}
