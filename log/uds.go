package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"net"
	"net/http"
	"sync"
	"time"
)

type udsCore struct {
	client       http.Client
	minimumLevel zapcore.Level
	url          string
	enc          zapcore.Encoder
	buffers      []*buffer.Buffer
	mu           sync.Mutex
}

func teeToUDSServer(baseCore zapcore.Core, address, path string) zapcore.Core {
	c := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", address)
			},
		},
		Timeout: 100 * time.Millisecond,
	}
	uc := &udsCore{
		client:  c,
		url:     "http://unix" + path,
		enc:     zapcore.NewJSONEncoder(defaultEncoderConfig),
		buffers: make([]*buffer.Buffer, 0),
	}
	for l := zapcore.DebugLevel; l <= zapcore.FatalLevel; l++ {
		if baseCore.Enabled(l) {
			uc.minimumLevel = l
			break
		}
	}
	return zapcore.NewTee(baseCore, uc)
}

func (u *udsCore) Enabled(level zapcore.Level) bool {
	return level >= u.minimumLevel
}

func (u *udsCore) With(fields []zapcore.Field) zapcore.Core {
	return &udsCore{
		client:       u.client,
		minimumLevel: u.minimumLevel,
	}
}

func (u *udsCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if u.Enabled(entry.Level) {
		return ce.AddCore(entry, u)
	}
	return ce
}

func (u *udsCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	buffer, err := u.enc.EncodeEntry(entry, fields)
	if err != nil {
		return fmt.Errorf("failed to write log to uds logger: %v", err)
	}
	u.mu.Lock()
	u.buffers = append(u.buffers, buffer)
	u.mu.Unlock()
	return nil
}

func (u *udsCore) Sync() error {
	logs := u.logsFromBuffer()
	msg, err := json.Marshal(logs)
	if err != nil {
		return fmt.Errorf("failed to sync uds log: %v", err)
	}
	resp, err := u.client.Post(u.url, "application/json", bytes.NewReader(msg))
	if err != nil {
		return fmt.Errorf("failed to send logs to uds server %v: %v", u.url, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("uds server returns non-ok status %v: %v", u.url, resp.Status)
	}
	return nil
}

func (u *udsCore) logsFromBuffer() []string {
	u.mu.Lock()
	defer u.mu.Unlock()
	logs := make([]string, 0, len(u.buffers))
	for _, b := range u.buffers {
		logs = append(logs, b.String())
		b.Free()
	}
	u.buffers = make([]*buffer.Buffer, 0)
	return logs
}
