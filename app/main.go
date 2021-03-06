package main

import (
	"context"
	"github.com/caarlos0/env/v6"
	joonix "github.com/joonix/log"
	"github.com/maksimru/event-scheduler/config"
	"github.com/maksimru/event-scheduler/scheduler"
	"github.com/maksimru/event-scheduler/version"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	os.Exit(app(context.Background()))
}

func app(ctx context.Context) int {
	ver := version.GetEventSchedulerVersion()

	cfg := config.Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Warningf("%+v\n", err)
	}

	log.WithFields(log.Fields{
		"os":         ver.OS,
		"build_date": ver.BuildDate,
		"revision":   ver.Revision,
		"version":    ver.Version,
		"go_version": ver.GoVersion,
		"arch":       ver.Arch,
	}).Info("Event Scheduler is starting...")

	setupLogger(cfg)

	if err := scheduler.NewScheduler(ctx, cfg).Run(ctx); err != nil {
		log.Error("scheduler failure: ", err.Error())
		return 1
	}

	return 0
}

func setupLogger(config config.Config) {
	log.SetOutput(os.Stdout)

	switch config.LogFormat {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	case "text":
		log.SetFormatter(&log.TextFormatter{})
	case "gcp":
		log.SetFormatter(joonix.NewFormatter())
	}

	switch config.LogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	case "warning":
		log.SetLevel(log.WarnLevel)
	case "trace":
		log.SetLevel(log.TraceLevel)
	}
}
