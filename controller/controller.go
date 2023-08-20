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
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type VirtualServicePatchReconciler struct {
	reconciler.Context
	IstioClient    *versionedclient.Clientset
	OldObjectCache cache.Indexer
}

func (r *VirtualServicePatchReconciler) Configure(ctx reconciler.Context) error {
	r.Context = ctx
	return ctx.NewControllerBuilder().
		For(&v1alpha1.VirtualServiceMerge{}, builder.WithPredicates(
			predicate.Funcs{
				CreateFunc: func(e event.CreateEvent) bool {
					return true
				},
				UpdateFunc: func(e event.UpdateEvent) bool {
					_ = r.OldObjectCache.Add(e.ObjectOld)
					return true
				},
				DeleteFunc: func(e event.DeleteEvent) bool {
					return true
				},
				GenericFunc: func(e event.GenericEvent) bool {
					return true
				},
			},
		)).
		Watches(&source.Kind{Type: &istio.VirtualService{}}, handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
			vs := obj.(*istio.VirtualService)
			requests := make([]reconcile.Request, 0)

			// skip if vs is being deleted
			if !vs.GetDeletionTimestamp().IsZero() {
				return requests
			}
			// get all virtual service merge whose target is this virtual service
			vsmegeList := &v1alpha1.VirtualServiceMergeList{}
			if err := r.Client().List(context.TODO(), vsmegeList, &client.ListOptions{
				Namespace: vs.GetNamespace(),
			}); err != nil {
				panic(err)
			}
			for _, vsmerge := range vsmegeList.Items {
				targetNamespace := vsmerge.Spec.Target.Namespace
				if targetNamespace == "" {
					targetNamespace = vsmerge.GetNamespace()
				}
				// only look for vs that is a target for any of the merge
				if vsmerge.Spec.Target.Name == vs.GetName() && targetNamespace == vs.GetNamespace() {
					request := reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: vsmerge.GetNamespace(),
							Name:      vsmerge.GetName(),
						},
					}
					requests = append(requests, request)
				}
			}
			return requests
		}),
			builder.WithPredicates(
				predicate.Funcs{
					CreateFunc: func(e event.CreateEvent) bool {
						return true
					},
					UpdateFunc: func(e event.UpdateEvent) bool {
						// ignore updates where the VirtualService spec does not change
						return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
					},
					DeleteFunc: func(e event.DeleteEvent) bool {
						return false
					},
					GenericFunc: func(e event.GenericEvent) bool {
						return false
					},
				},
			),
		).
		Complete(r)
}

func (r *VirtualServicePatchReconciler) Reconcile(_ context.Context, request reconcile.Request) (reconcile.Result, error) {
	patch := &v1alpha1.VirtualServiceMerge{}
	oldObj, exists, err := r.OldObjectCache.GetByKey(request.NamespacedName.String())
	if err != nil {
		return reconcile.Result{}, err
	}
	result, err := r.Run(request, patch, func(_ bool) error {
		if exists {
			if err := Reconcile(r.Context, r.IstioClient, patch, oldObj); err != nil {
				if kerr.IsNotFound(err) {
					// do not need to panic just log output
					r.Context.Logger().Info("Virtual service not found. Nothing to sync.")
					// update completed, remove key from cache
					_ = r.OldObjectCache.Delete(oldObj)
					return nil
				}
				return err
			}
			// update completed, remove key from cache
			_ = r.OldObjectCache.Delete(oldObj)
		} else {
			if err := Reconcile(r.Context, r.IstioClient, patch, nil); err != nil {
				if kerr.IsNotFound(err) {
					// do not need to panic just log output
					r.Context.Logger().Info("Virtual service not found. Nothing to sync.")
					return nil
				}
				return err
			}
		}
		return nil
	})
	return result, err
}
