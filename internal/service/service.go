package service

import (
	"context"
	"github.com/google/uuid"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-webhook-notifier/internal/handler"
	"github.com/paysuper/paysuper-webhook-notifier/pkg/proto/grpc"
	"net/http"
	"time"
)

type Service struct {
	sender handler.HttpSender
}

func (s *Service) CheckUser(ctx context.Context, req *grpc.CheckUserRequest, res *grpc.CheckUserResponse) error {
	res.Status = pkg.ResponseStatusOk

	msg := &UserNotificationMessage{
		Id:          uuid.New().String(),
		Type:        "check_user",
		CreatedAt:   time.Now().Format(time.RFC3339),
		DeliveryTry: 1,
		Live:        true,
		Object: 	 req.User,
	}

	resp, err := s.sender.Send(req.Url, msg, handler.NotificationActionCheckUser, req.SecretKey)
	if err != nil {
		res.Status = pkg.ResponseStatusSystemError
		res.Message = err.Error()
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		res.Status = pkg.ResponseStatusBadData
		return nil
	}

	return nil
}

type UserNotificationMessage struct {
	Id          string           `json:"id"`
	Type        string           `json:"type"`
	Event       string           `json:"event"`
	Live        bool             `json:"live"`
	CreatedAt   string           `json:"created_at"`
	ExpiresAt   string           `json:"expires_at"`
	DeliveryTry int32            `json:"delivery_try"`
	Object      *grpc.User		 `json:"object"`
}
