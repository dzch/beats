package http

import (
	"fmt"
	"time"
)

type httpConfig struct {
	Url               string        `config:"url"`
	CTimeout          time.Duration `config:"conn_timeout"`
	ProcTimeout       time.Duration `config:"proc_timeout"`
	WorkerNum         int           `config:"worker_num"`
	MaxRetries        int           `config:"max_retries"`
	QueueSize         int           `config:"queue_size"`
	FailRetryInterval time.Duration `config:"fail_retry_interval"`
}

var (
	defaultConfig = httpConfig{
		Url:               "",
		CTimeout:          200 * time.Millisecond,
		ProcTimeout:       3 * time.Second,
		MaxRetries:        -1,
		WorkerNum:         1,
		QueueSize:         256,
		FailRetryInterval: 300 * time.Millisecond,
	}
)

func (c *httpConfig) Validate() error {
	if len(c.Url) == 0 {
		return fmt.Errorf("url must be set, now is %v", c.Url)
	}
	return nil
}
