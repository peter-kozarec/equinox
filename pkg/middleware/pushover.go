package middleware

import (
	"context"
	"fmt"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Pushover struct {
	ctx    context.Context
	logger *zap.Logger
	user   string
	token  string
	device string
}

func NewPushover(ctx context.Context, logger *zap.Logger, user, token, device string) *Pushover {
	return &Pushover{
		ctx:    ctx,
		logger: logger,
		user:   user,
		token:  token,
		device: device,
	}
}

func (p *Pushover) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(position common.Position) {
		go func() {
			msg := fmt.Sprintf("id = %d\npnl = %s", position.Id.Int64(), position.NetProfit.Rescale(2).String())
			if err := sendPushoverNotification(p.ctx, p.token, p.user, p.device, "Position Closed", msg); err != nil {
				p.logger.Error("sendPushoverNotification", zap.Error(err))
			}
		}()
		handler(position)
	}
}

func sendPushoverNotification(ctx context.Context, token, user, device, title, message string) error {
	data := url.Values{}
	data.Set("token", token)
	data.Set("user", user)
	data.Set("device", device)
	data.Set("title", title)
	data.Set("message", message)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.pushover.net/1/messages.json", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("pushover post failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pushover error: %s", body)
	}

	return nil
}
