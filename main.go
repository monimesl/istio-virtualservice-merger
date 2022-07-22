/*
 * Copyright 2021 - now, the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"flag"
	"log"

	"github.com/monimesl/istio-virtualservice-merger/api/v1alpha1"
	"github.com/monimesl/istio-virtualservice-merger/controller"
	"go.uber.org/zap/zapcore"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	versionedclient "istio.io/client-go/pkg/clientset/versioned"

	"github.com/monimesl/operator-helper/config"
	"github.com/monimesl/operator-helper/reconciler"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha3.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var namespace string
	flag.StringVar(&namespace, "namespace", "istio-merger-operator", "Select which namespace this controller is deployed")
	flag.Parse()

	// set logger
	opts := zap.Options{
		Development: true,
		Encoder:     zapcore.NewJSONEncoder(zapcore.EncoderConfig{}),
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// start manager
	cfg, options := config.GetManagerParams(scheme,
		namespace,
		"istiomerger.monime.sl")
	mgr, err := manager.New(cfg, options)
	if err != nil {
		log.Fatalf("manager create error: %s", err)
	}
	ic, err := versionedclient.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create istio client: %s", err)
	}
	if err = reconciler.Configure(mgr,
		&controllers.VirtualServicePatchReconciler{IstioClient: ic}); err != nil {
		log.Fatalf("reconciler cfg error: %s", err)
	}
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatalf("operator start error: %s", err)
	}
}
