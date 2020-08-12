//go:generate go run pkg/codegen/cleanup/main.go
//go:generate go run pkg/codegen/main.go

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/rancher/gitjob/pkg/hooks"

	"github.com/rancher/gitjob/pkg/controller"
	"github.com/rancher/gitjob/pkg/types"
	"github.com/rancher/wrangler/pkg/leader"
	"github.com/rancher/wrangler/pkg/resolvehome"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	Version   = "v0.0.0-dev"
	GitCommit = "HEAD"
)

func main() {
	app := cli.NewApp()
	app.Name = "testy"
	app.Version = fmt.Sprintf("%s (%s)", Version, GitCommit)
	app.Usage = "testy needs help!"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "kubeconfig",
			EnvVar: "KUBECONFIG",
		},
		cli.StringFlag{
			Name:  "listen-port",
			Value: ":80",
		},
	}
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) {
	flag.Parse()

	logrus.Info("Starting controller")
	ctx := signals.SetupSignalHandler(context.Background())

	kubeconfig, err := resolvehome.Resolve(c.String("kubeconfig"))
	if err != nil {
		logrus.Fatalf("Error resolving home dir: %s", err.Error())
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logrus.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	ctx, cont := types.BuildContext(ctx, "gitjob", cfg)
	if err := cont.Start(ctx); err != nil {
		logrus.Fatal(err)
	}

	go func() {
		leader.RunOrDie(ctx, "gitjob", "gitjob", cont.K8s, func(ctx context.Context) {
			controller.Register(ctx, cont)
			runtime.Must(cont.Start(ctx))
			<-ctx.Done()
		})
	}()

	logrus.Info("Setting up webhook listener")
	handler := hooks.HandleHooks(cont)
	if err := http.ListenAndServe(":80", handler); err != nil {
		logrus.Fatalf("Failed to listen on %s: %v", "80", err)
	}

	<-ctx.Done()
}
