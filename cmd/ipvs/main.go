package main

import (
	"context"

	"github.com/gradusp/crispy-ipvs/internal/app"
	"github.com/gradusp/crispy-ipvs/internal/config"
	"github.com/gradusp/go-platform/app/tracing/ot"
	"github.com/gradusp/go-platform/logger"
	pkgNet "github.com/gradusp/go-platform/pkg/net"
	"github.com/gradusp/go-platform/server"
	"go.uber.org/zap"
)

func main() {
	setupContext()
	ctx := app.Context()
	logger.SetLevel(zap.InfoLevel)
	logger.Info(ctx, "--== HELLO ==--")
	err := config.InitGlobalConfig(
		config.WithAcceptEnvironment{EnvPrefix: "IPVS"},
		config.WithSourceFile{FileName: app.ConfigFile},
		config.WithDefValue{Key: app.LoggerLevel, Val: "INFO"},
		config.WithDefValue{Key: app.MetricsEnable, Val: false},
		config.WithDefValue{Key: app.TraceEnable, Val: false},
		config.WithDefValue{Key: app.ServerGracefulShutdown, Val: "10s"},
		config.WithDefValue{Key: app.ServerEndpoint, Val: "tcp://127.0.0.1:9006"},
	)
	if err != nil {
		logger.Fatal(ctx, err)
	}
	if err = setupLogger(); err != nil {
		logger.Fatalf(ctx, "setup logger: %v", err)
	}
	if err = setupMetrics(); err != nil {
		logger.Fatalf(ctx, "setup metrics: %v", err)
	}
	if err = setupTracer(); err != nil {
		logger.Fatalf(ctx, "setup tracer: %v", err)
	}
	var srv *server.APIServer
	if srv, err = setupServer(ctx); err != nil {
		logger.Fatalf(ctx, "setup server: %v", err)
	}
	var endPointAddress string
	if endPointAddress, err = app.ServerEndpoint.Maybe(ctx); err != nil {
		logger.Fatalf(ctx, "get server endpoint from config: %v", err)
	}
	var ep *pkgNet.Endpoint
	if ep, err = pkgNet.ParseEndpoint(endPointAddress); err != nil {
		logger.Fatalf(ctx, "parse server endpoint (%s): %v", endPointAddress, err)
	}
	gracefulDuration, _ := app.ServerGracefulShutdown.Maybe(ctx)
	if err = srv.Run(ctx, ep, server.RunWithGracefulStop(gracefulDuration)); err != nil {
		logger.Fatalf(ctx, "run server: %v", err)
	}
	WhenHaveTracerProvider(func(tp ot.TracerProvider) {
		_ = tp.Shutdown(context.Background())
	})
	logger.Info(ctx, "--== BYE ==--")
}

/*//

a := ipvs.VServer_RoundRobin
v := a.Descriptor().Values().ByNumber(a.Number())
o := v.Options()
ff, _ := o.(*descriptorpb.EnumValueOptions)
if ff != nil {
	a, b := proto.GetExtension(ff, ipvs.E_VServer_Name)
	if b == nil {
		ss := a.(*string)
		*ss += ""
		*ss = "12"
		*ss += ""

		a, _ = proto.GetExtension(ff, ipvs.E_VServer_Name)
		ss = a.(*string)
		*ss += ""

	}
}

b := ipvs.VServer_Other
v = b.Descriptor().Values().ByNumber(b.Number())
switch ff := v.Options().(type) {
case nil:
case *descriptorpb.EnumValueOptions:
	if a, b := proto.GetExtension(ff, ipvs.E_VServer_Name); b == nil {
		ss := a.(*string)
		*ss += ""
	}
}
i := 1
i++
*/
