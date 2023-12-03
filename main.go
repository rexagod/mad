/*
Copyright 2023 The Kubernetes mad Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"

	"github.com/rexagod/mad/internal"
	v "github.com/rexagod/mad/internal/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	clientset "github.com/rexagod/mad/pkg/generated/clientset/versioned"
	"github.com/rexagod/mad/pkg/signals"
)

func main() {

	// Set up flags.
	klog.InitFlags(nil)
	klog.SetOutput(os.Stdout)
	kubeconfig := *flag.String("kubeconfig", os.Getenv("KUBECONFIG"), "Path to a kubeconfig. Only required if out-of-cluster.")
	masterURL := *flag.String("master", os.Getenv("KUBERNETES_MASTER"), "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	workers := *flag.Int("workers", 2, "Number of workers processing the queue. Defaults to 2.")
	version := *flag.Bool("version", false, "Print version information and quit")
	flag.Parse()

	// Print version information.
	v.Println()

	// Quit if only version flag is set.
	if version && flag.NFlag() == 1 {
		os.Exit(0)
	}

	// Set up signals, so we can handle the shutdown signal gracefully.
	ctx := signals.SetupSignalHandler()
	logger := klog.FromContext(ctx)

	// Build client-sets.
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		logger.Error(err, "Error building kubeconfig")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	kubeClientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	madClientset, err := clientset.NewForConfig(cfg)
	if err != nil {
		logger.Error(err, "Error building mad clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	// Start the controller.
	if err = internal.NewController(ctx, kubeClientset, madClientset).Run(ctx, workers); err != nil {
		logger.Error(err, "Error running controller")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
}
