# istio-virtualservice-merger

A simple kubernetes operator to merge seperated
Istio [virtual services](https://istio.io/latest/docs/reference/config/networking/virtual-service/) using the same host.

___Note: This a temporary solution to the [issue](https://github.com/istio/istio/issues/22997)___

__Please read the above [issue](https://github.com/istio/istio/issues/22997) to understand why this is needed__

## Installation

#### Install the CRD

```shell
kubectl apply -f https://raw.githubusercontent.com/monimesl/istio-virtualservice-merger/master/manifest/crd.yaml
```

#### Deploy the operator

```shell
kubectl apply -f https://raw.githubusercontent.com/monimesl/istio-virtualservice-merger/master/manifest/operator.yaml
```

##### Create a target [virtual service](https://istio.io/latest/docs/reference/config/networking/virtual-service/) on which seperated patches are merged into.

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: api-routes
spec:
  gateways:
    - "mesh"
  hosts:
    - "internal-api.monime.sl"
```

##### Create a review VirtualServiceMerge

```yaml
apiVersion: istiomerger.monime.sl/v1alpha1
kind: VirtualServiceMerge
metadata:
  name: review-routes
  namespace: app-space
spec:
  target:
    name: "api-routes" ## the virtual service above
    namespace: "" # empty or omitted means the same as the VirtualServiceMerge
  patch: # same as https://istio.io/latest/docs/reference/config/networking/virtual-service/#VirtualService
    http:
      - match:
          - uri:
              prefix: "/reviews"
        route:
          - destination:
              port:
                number: 8080
              host: "review-service"
```

##### Create a product VirtualServiceMerge

```yaml
apiVersion: istiomerger.monime.sl/v1alpha1
kind: VirtualServiceMerge
metadata:
  name: product-routes
  namespace: app-space
spec:
  target:
    name: "api-routes" ## the virtual service above
    namespace: "" # empty or omitted means the same as the VirtualServiceMerge
  patch: # same as https://istio.io/latest/docs/reference/config/networking/virtual-service/#VirtualService
    http:
      - match:
          - uri:
              prefix: "/products"
        route:
          - destination:
              port:
                number: 8080
              host: "product-service"
```

The two VirtualServiceMerge objects will be merged into the targeted VirtualServices to produce a result similar to
below:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: api-routes
spec:
  gateways:
    - "mesh"
  hosts:
    - "internal-api.monime.sl"
  http:
    - match:
        - uri:
            prefix: "/reviews"
      name: review-routes-0 # 0 is the precedence
      route:
        - destination:
            port:
              number: 8080
            host: "review-service"
    - match:
        - uri:
            prefix: "/products"
      # 1 is the precedence, 'product-routes' route will appear before any routes with lesser 
      # precedence that were merged onto the target even if they're from a different patch.      
      name: product-routes-1
      route:
        - destination:
            port:
              number: 8080
            host: "product-service"
```

#### The merging works for TCP and TLS routes as well
