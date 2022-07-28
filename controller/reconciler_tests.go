package controllers

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang/mock/gomock"
	msvergealpha1 "github.com/monimesl/istio-virtualservice-merger/api/v1alpha1"
	"github.com/monimesl/istio-virtualservice-merger/tests/mocks"
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
	istio "istio.io/client-go/pkg/apis/networking/v1alpha3"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Reconciler", func() {
	Context("method Reconcile(ctx, client, patch)", func() {
		var pwd string
		var ctrl *gomock.Controller
		var mock_clientset *mocks.MockInterface
		var mock_reconciler_context *mocks.MockContext
		var mock_logger *mocks.MockLogger
		var mock_client *mocks.MockClient
		var mock_network_client *mocks.MockNetworkingV1alpha3Interface
		var mock_vs_interface *mocks.MockVirtualServiceInterface
		var vs istio.VirtualService
		var vsMerge msvergealpha1.VirtualServiceMerge

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			_, filename, _, _ := runtime.Caller(0)
			pwd = filepath.Dir(filename)
			mock_logger = mocks.NewMockLogger(ctrl)
			mock_clientset = mocks.NewMockInterface(ctrl)
			mock_reconciler_context = mocks.NewMockContext(ctrl)
			mock_client = mocks.NewMockClient(ctrl)
			mock_network_client = mocks.NewMockNetworkingV1alpha3Interface(ctrl)
			mock_vs_interface = mocks.NewMockVirtualServiceInterface(ctrl)

			// load virtualservice
			payload, err := os.ReadFile(pwd + "/../tests/data/vs.yaml")
			if err != nil {
				panic(err)
			}
			vs = istio.VirtualService{}
			if jsonByte, err := yaml.YAMLToJSON(payload); err == nil {
				if err := json.Unmarshal(jsonByte, &vs); err != nil {
					panic(err)
				}
			}

			// load virsutalservicemerge
			payload, err = os.ReadFile(pwd + "/../tests/data/vs-merge-1.yaml")
			if err != nil {
				panic(err)
			}
			vsMerge = msvergealpha1.VirtualServiceMerge{}
			if jsonByte, err := yaml.YAMLToJSON(payload); err == nil {
				if err := json.Unmarshal(jsonByte, &vsMerge); err != nil {
					panic(err)
				}
			}
		})

		// =================================================================================
		DescribeTable("will not throw exception",
			func(vsExists bool, e error) {
				vsMerge.Finalizers = append(vsMerge.Finalizers, "istiomerger.monime.sl-finalizer")
				vsMerge.ResourceVersion = "1"

				// setup expectations
				mock_client.EXPECT().Update(gomock.Any(), &vsMerge, gomock.Any()).AnyTimes()
				mock_client.EXPECT().Status().Return(mock_client)

				// expect vs update
				if vsExists {
					mock_vs_interface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(&vs, e)
					mock_vs_interface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any())
				} else {
					mock_logger.EXPECT().Info("Virtual service not found. Nothing to sync.").AnyTimes()
					mock_vs_interface.EXPECT().
						Get(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(&vs, e)
				}

				mock_network_client.EXPECT().VirtualServices(gomock.Any()).Return(mock_vs_interface).AnyTimes()
				mock_clientset.EXPECT().NetworkingV1alpha3().Return(mock_network_client).AnyTimes()

				mock_reconciler_context.EXPECT().Client().Return(mock_client)
				mock_reconciler_context.EXPECT().Logger().Return(mock_logger).AnyTimes()

				// mock_clientset.EXPECT().Logger()
				err := Reconcile(mock_reconciler_context, mock_clientset, &vsMerge)

				Expect(err).To(BeNil())
			},
			Entry("if VirtualService exists", true, nil),
			Entry("if VirtualService does not exists", false, kerr.NewNotFound(schema.GroupResource{}, "vs not found")),
		)

		// =================================================================================
		It("will update finalizers on first run",
			func() {
				vsMerge.Finalizers = make([]string, 0)
				vsMerge.ResourceVersion = "1"

				// setup expectations
				mock_logger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
				mock_client.EXPECT().Update(gomock.Any(), &vsMerge, gomock.Any()).AnyTimes()
				mock_reconciler_context.EXPECT().Logger().Return(mock_logger).AnyTimes()
				mock_reconciler_context.EXPECT().Client().Return(mock_client)

				// mock_clientset.EXPECT().Logger()
				err := Reconcile(mock_reconciler_context, mock_clientset, &vsMerge)

				Expect(err).To(BeNil())
			},
		)

		// =================================================================================
		DescribeTable("will delete virtualservicemerge successfully",
			func(vsExists bool, e error) {
				vsMerge.Finalizers = make([]string, 0)
				vsMerge.Finalizers = append(vsMerge.Finalizers, "istiomerger.monime.sl-finalizer")
				vsMerge.DeletionTimestamp = &v1.Time{
					Time: time.Now(),
				}
				vsMerge.ResourceVersion = "1"

				// setup expectations
				mock_client.EXPECT().Update(gomock.Any(), &vsMerge, gomock.Any()).AnyTimes()
				mock_reconciler_context.EXPECT().Logger().Return(mock_logger).AnyTimes()
				mock_reconciler_context.EXPECT().Client().Return(mock_client)
				// expect vs update
				if vsExists {
					mock_vs_interface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(&vs, e)
					mock_vs_interface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any())
				} else {
					mock_logger.EXPECT().Info("Virtual service not found. Nothing to sync.").AnyTimes()
					mock_vs_interface.EXPECT().
						Get(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(&vs, e)
				}

				mock_network_client.EXPECT().VirtualServices(gomock.Any()).Return(mock_vs_interface).AnyTimes()
				mock_clientset.EXPECT().NetworkingV1alpha3().Return(mock_network_client).AnyTimes()

				// mock_clientset.EXPECT().Logger()
				err := Reconcile(mock_reconciler_context, mock_clientset, &vsMerge)

				Expect(err).To(BeNil())
			},
			Entry("if VirtualService exists", true, nil),
			Entry("if VirtualService does not exists", false, kerr.NewNotFound(schema.GroupResource{}, "vs not found")),
		)

		// =================================================================================
		DescribeTable("will panic with error",
			func(e error) {
				vsMerge.Finalizers = make([]string, 0)
				vsMerge.Finalizers = append(vsMerge.Finalizers, "istiomerger.monime.sl-finalizer")
				vsMerge.DeletionTimestamp = &v1.Time{
					Time: time.Now(),
				}
				vsMerge.ResourceVersion = "1"

				// setup expectations
				mock_reconciler_context.EXPECT().Logger().Return(mock_logger).AnyTimes()
				mock_vs_interface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(&vs, e)

				mock_network_client.EXPECT().VirtualServices(gomock.Any()).Return(mock_vs_interface).AnyTimes()
				mock_clientset.EXPECT().NetworkingV1alpha3().Return(mock_network_client).AnyTimes()

				// mock_clientset.EXPECT().Logger()
				Expect(func() {
					_ = Reconcile(mock_reconciler_context, mock_clientset, &vsMerge)
				}).To(PanicWith(e))
			},
			Entry("for 'bad request' error", kerr.NewBadRequest("bad request")),
			Entry("for 'content expired' error", kerr.NewResourceExpired("expired")),
			Entry("for 'internal server error'", kerr.NewInternalError(errors.New("server error"))),
		)
		// =================================================================================
		// =================================================================================
		AfterEach(func() {
		})
	})
})
