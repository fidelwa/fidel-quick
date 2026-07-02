package email

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSender_DefaultsToStdout(t *testing.T) {
	log := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	assert.IsType(t, &StdoutSender{}, NewSender("", "from@x.com", log))
	assert.IsType(t, &StdoutSender{}, NewSender("stdout", "from@x.com", log))
	// Unknown providers fall back to stdout so a misconfig never breaks boot.
	assert.IsType(t, &StdoutSender{}, NewSender("mystery", "from@x.com", log))
}

func TestStdoutSender_LogsMessage(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	s := NewStdoutSender("no-reply@fidel.app", log)

	err := s.Send(context.Background(), Message{
		To:      "user@test.com",
		Subject: "Reset",
		Body:    "link http://x/reset-password?token=abc",
	})
	require.NoError(t, err)

	out := buf.String()
	assert.True(t, strings.Contains(out, "user@test.com"))
	assert.True(t, strings.Contains(out, "token=abc"))
}
