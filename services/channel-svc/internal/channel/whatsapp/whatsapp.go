// services/channel-svc/internal/channel/whatsapp/whatsapp.go

// Package whatsapp sends notifications via WhatsApp using whatsmeow.
// ponytail: LOG-ONLY stub until whatsmeow credentials land.
package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
)

// Client sends WhatsApp messages.
type Client struct {
	log    *slog.Logger
	dryRun bool
}

// Config holds WhatsApp connection settings.
type Config struct {
	DryRun bool // If true, log messages but don't send
}

// NewClient creates a WhatsApp client stub.
func NewClient(cfg Config, log *slog.Logger) *Client {
	if log == nil {
		log = slog.Default()
	}
	return &Client{
		log:    log,
		dryRun: true, // ponytail: always dry-run until whatsmeow session store exists
	}
}

// SendMessage sends a text message.
func (c *Client) SendMessage(ctx context.Context, to, text string) error {
	c.log.Info("whatsapp_send_stub", "to", to, "text_len", len(text), "dry_run", c.dryRun)
	if !c.dryRun {
		return fmt.Errorf("whatsapp: real sending not implemented")
	}
	return nil
}

// SendTemplate sends a template message with variables.
func (c *Client) SendTemplate(ctx context.Context, to, templateName string, vars map[string]string, language string) error {
	c.log.Info("whatsapp_template_stub", "to", to, "template", templateName, "language", language, "dry_run", c.dryRun)
	if !c.dryRun {
		return fmt.Errorf("whatsapp: real sending not implemented")
	}
	return nil
}
