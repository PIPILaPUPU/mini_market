package handler

import (
	"minimarket/internal/service"
)

type RegisterHandler struct {
	registerService *service.RegisterService
}

type NewRegisterHandler struct {
	RegisterService *service.RegisterService
}
