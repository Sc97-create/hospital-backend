package dto

type Response struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
	Code    string `json:"code"`
}
