package server

import (
	"github.com/interuss/dss/pkg/surveillance/application"
	"github.com/robfig/cron/v3"
)

// Server implements surveillancev0.Implementation.
type Server struct {
	App               application.App
	Locality          string
	AllowHTTPBaseUrls bool
	Cron              *cron.Cron
}
