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

package controllers

import (
	"context"
	"github.com/monimesl/istio-virtualservice-merger/api/v1alpha1"
	"github.com/monimesl/operator-helper/reconciler"
	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type VirtualServicePatchReconciler struct {
	reconciler.Context
	IstioClient *versionedclient.Clientset
}

func (r *VirtualServicePatchReconciler) Configure(ctx reconciler.Context) error {
	r.Context = ctx
	return ctx.NewControllerBuilder().
		For(&v1alpha1.VirtualServiceMerge{}).
		Complete(r)
}

func (r *VirtualServicePatchReconciler) Reconcile(_ context.Context, request reconcile.Request) (reconcile.Result, error) {
	patch := &v1alpha1.VirtualServiceMerge{}
	return r.Run(request, patch, func(_ bool) error {
		return Reconcile(r.Context, r.IstioClient, patch)
	})
}
