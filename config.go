package httpserve

import "time"

// Config is the basic configuration
type Config struct {
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
}

type AutoCertConfig struct {
	DirCache string
	Hosts    []string
}
