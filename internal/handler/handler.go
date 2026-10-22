package handler

import (
	"github.com/ayanosky/url-shortener2.0/internal/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}
