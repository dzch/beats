package http

import (
	"fmt"
	"time"
)

type httpConfig struct {
	Uri               string        `config:"uri"`
	Addrs             []string      `config:"addrs"`
	CTimeout          time.Duration `config:"conn_timeout"`
	ProcTimeout       time.Duration `config:"proc_timeout"`
	WorkerNum         int           `config:"worker_num"`
	MaxRetries        int           `config:"max_retries"`
	QueueSize         int           `config:"queue_size"`
	FailRetryInterval time.Duration `config:"fail_retry_interval"`
}

var (
	defaultConfig = httpConfig{
		Uri:               "",
		CTimeout:          200 * time.Millisecond,
		ProcTimeout:       3 * time.Second,
		MaxRetries:        -1,
		WorkerNum:         1,
		QueueSize:         256,
		FailRetryInterval: 300 * time.Millisecond,
	}
)

func (c *httpConfig) Validate() error {
	if len(c.Uri) == 0 {
		return fmt.Errorf("uri must be set, now is %v", c.Uri)
	}
	if len(c.Addrs) == 0 {
		return fmt.Errorf("addrs must be set, now len is %d", len(c.Addrs))
	}

	return nil
}
