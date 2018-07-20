package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type serviceMetadata struct {
	stack          string
	service        string
	serviceVersion string
	serviceMode    string
	customLabels   map[string]string
}

var (
	metadataCache = make(map[string]serviceMetadata)
)

func buildMetadata(svc swarm.Service) serviceMetadata {
	md := serviceMetadata{
		stack:          svc.Spec.Labels["com.docker.stack.namespace"],
		service:        svc.Spec.Name,
		serviceVersion: fmt.Sprint(svc.Meta.Version.Index),
		serviceMode:    serviceMode(svc),
		customLabels:   make(map[string]string),
	}

	for _, label := range customLabels {
		if val, ok := svc.Spec.Annotations.Labels[label]; ok {
			md.customLabels[label] = val
		} else {
			md.customLabels[label] = ""
		}
	}

	return md
}

func serviceMode(svc swarm.Service) string {
	if svc.Spec.Mode.Replicated != nil {
		return "replicated"
	}

	return "global"
}

func sanitizeLabelNames(orig []string) []string {
	dst := make([]string, 0, len(orig))

	for _, label := range orig {
		s := strings.Replace(label, ".", "_", -1)
		dst = append(dst, s)
	}

	return dst
}

func sanitizeMetricLabels(orig prometheus.Labels) prometheus.Labels {
	dst := make(prometheus.Labels)

	for name, val := range orig {
		s := strings.Replace(name, ".", "_", -1)
		dst[s] = val
	}

	return dst
}

func configureLogger() {
	switch *logFormat {
	case "text":
		logrus.SetFormatter(new(logrus.TextFormatter))
	case "json":
		logrus.SetFormatter(new(logrus.JSONFormatter))
	default:
		fmt.Fprintf(os.Stderr, "Invalid log format %q. Should be either json or text.", *logFormat)
		os.Exit(1)
	}

	switch *logLevel {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	case "panic":
		logrus.SetLevel(logrus.PanicLevel)
	default:
		fmt.Fprintf(os.Stderr, "Invalid log level %q. Should be either debug, info, warn, error, fatal, panic.", *logLevel)
		os.Exit(1)
	}
}

type stringSlice []string

func (i *stringSlice) String() string {
	return fmt.Sprint(*i)
}

func (i *stringSlice) Set(value string) error {
	if len(value) == 0 {
		return errors.New("empty flag value")
	}

	*i = append(*i, value)

	return nil
}

var (
	listenAddr = flag.String("listen-addr", "0.0.0.0:8888", "IP address and port to bind")
	pollDelay  = flag.Duration("poll-delay", 10*time.Second, "Delay in seconds between two polls")
	logFormat  = flag.String("log-format", "text", "Either json or text")
	logLevel   = flag.String("log-level", "info", "Either debug, info, warn, error, fatal, panic")
	help       = flag.Bool("help", false, "Display help message")

	customLabels stringSlice
)

func usage() {
	fmt.Printf("Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Var(&customLabels, "label", "Name of custom service labels to add to metrics")
	flag.Parse()

	if *help {
		usage()
		os.Exit(0)
	}

	configureLogger()
	configureDesiredReplicasGauge()
	configureReplicasStateGauge()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logrus.Fatal(err)
	}
	defer cli.Close()

	ctx := context.Background()

	go func() {
		if err := initDesiredReplicasGauge(ctx, cli); err != nil {
			logrus.Fatal(err)
		}

		if err := listenSwarmEvents(ctx, cli); err != nil {
			logrus.Fatal(err)
		}
	}()

	go func() {
		logrus.Info("Start polling replicas state every ", *pollDelay)

		for {
			logrus.Info("Polling replicas state...")

			polled, err := pollReplicasState(ctx, cli)
			if err != nil {
				logrus.Error(err)
			}

			updateReplicasStateGauge(polled)
			time.Sleep(*pollDelay)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	logrus.Infof("Start HTTP server on %q.", *listenAddr)

	if err := http.ListenAndServe(*listenAddr, mux); err != nil {
		logrus.Fatal(err)
	}
}
