package virtualnode

import (
	"context"
	appmesh "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/equality"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"
	"github.com/aws/aws-sdk-go/aws"
	appmeshsdk "github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func Test_defaultResourceManager_updateCRDVirtualNode(t *testing.T) {
	type args struct {
		vn    *appmesh.VirtualNode
		sdkVN *appmeshsdk.VirtualNodeData
	}
	tests := []struct {
		name    string
		args    args
		wantVN  *appmesh.VirtualNode
		wantErr error
	}{
		{
			name: "virtualNode needs patch both arn and condition",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
					},
					Status: appmesh.VirtualNodeStatus{},
				},
				sdkVN: &appmeshsdk.VirtualNodeData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.VirtualNodeStatus{
						Status: aws.String(appmeshsdk.VirtualNodeStatusCodeActive),
					},
				},
			},
			wantVN: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vn-1",
				},
				Status: appmesh.VirtualNodeStatus{
					VirtualNodeARN: aws.String("arn-1"),
					Conditions: []appmesh.VirtualNodeCondition{
						{
							Type:   appmesh.VirtualNodeActive,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name: "virtualNode needs patch condition only",
			args: args{
				vn: &appmesh.VirtualNode{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vn-1",
					},
					Status: appmesh.VirtualNodeStatus{
						VirtualNodeARN: aws.String("arn-1"),
						Conditions: []appmesh.VirtualNodeCondition{
							{
								Type:   appmesh.VirtualNodeActive,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				sdkVN: &appmeshsdk.VirtualNodeData{
					Metadata: &appmeshsdk.ResourceMetadata{
						Arn: aws.String("arn-1"),
					},
					Status: &appmeshsdk.VirtualNodeStatus{
						Status: aws.String(appmeshsdk.VirtualNodeStatusCodeInactive),
					},
				},
			},
			wantVN: &appmesh.VirtualNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vn-1",
				},
				Status: appmesh.VirtualNodeStatus{
					VirtualNodeARN: aws.String("arn-1"),
					Conditions: []appmesh.VirtualNodeCondition{
						{
							Type:   appmesh.VirtualNodeActive,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k8sSchema := runtime.NewScheme()
			clientgoscheme.AddToScheme(k8sSchema)
			appmesh.AddToScheme(k8sSchema)
			k8sClient := testclient.NewFakeClientWithScheme(k8sSchema)
			m := &defaultResourceManager{
				k8sClient: k8sClient,
				log:       log.NullLogger{},
			}

			err := k8sClient.Create(ctx, tt.args.vn.DeepCopy())
			assert.NoError(t, err)
			err = m.updateCRDVirtualNode(ctx, tt.args.vn, tt.args.sdkVN)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				gotVN := &appmesh.VirtualNode{}
				err = k8sClient.Get(ctx, k8s.NamespacedName(tt.args.vn), gotVN)
				assert.NoError(t, err)
				opts := cmp.Options{
					equality.IgnoreFakeClientPopulatedFields(),
					cmpopts.IgnoreTypes((*metav1.Time)(nil)),
				}
				assert.True(t, cmp.Equal(tt.wantVN, gotVN, opts), "diff", cmp.Diff(tt.wantVN, gotVN, opts))
			}
		})
	}
}

func Test_defaultResourceManager_buildSDKVirtualServiceReferenceConvertFunc(t *testing.T) {
	vsRefWithNamespace := appmesh.VirtualServiceReference{
		Namespace: aws.String("my-ns"),
		Name:      "vs-1",
	}
	vsRefWithoutNamespace := appmesh.VirtualServiceReference{
		Namespace: nil,
		Name:      "vs-2",
	}

	type args struct {
		vsByRef map[appmesh.VirtualServiceReference]*appmesh.VirtualService
	}
	tests := []struct {
		name                  string
		args                  args
		wantAWSNameOrErrByRef map[appmesh.VirtualServiceReference]struct {
			awsName string
			err     error
		}
	}{
		{
			name: "when all virtualServiceReference resolve correctly",
			args: args{
				vsByRef: map[appmesh.VirtualServiceReference]*appmesh.VirtualService{
					vsRefWithNamespace: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vs-1",
						},
						Spec: appmesh.VirtualServiceSpec{
							AWSName: aws.String("vs-1.my-ns"),
						},
					},
					vsRefWithoutNamespace: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vs-2",
						},
						Spec: appmesh.VirtualServiceSpec{
							AWSName: aws.String("vs-2.my-ns"),
						},
					},
				},
			},
			wantAWSNameOrErrByRef: map[appmesh.VirtualServiceReference]struct {
				awsName string
				err     error
			}{
				vsRefWithNamespace: {
					awsName: "vs-1.my-ns",
				},
				vsRefWithoutNamespace: {
					awsName: "vs-2.my-ns",
				},
			},
		},
		{
			name: "when some virtualServiceReference cannot resolve correctly",
			args: args{
				vsByRef: map[appmesh.VirtualServiceReference]*appmesh.VirtualService{
					vsRefWithNamespace: {
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "my-ns",
							Name:      "vs-1",
						},
						Spec: appmesh.VirtualServiceSpec{
							AWSName: aws.String("vs-1.my-ns"),
						},
					},
				},
			},
			wantAWSNameOrErrByRef: map[appmesh.VirtualServiceReference]struct {
				awsName string
				err     error
			}{
				vsRefWithNamespace: {
					awsName: "vs-1.my-ns",
				},
				vsRefWithoutNamespace: {
					err: errors.Errorf("unexpected VirtualServiceReference: %v", vsRefWithoutNamespace),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			m := &defaultResourceManager{}
			convertFunc := m.buildSDKVirtualServiceReferenceConvertFunc(ctx, tt.args.vsByRef)
			for vsRef, wantAWSNameOrErr := range tt.wantAWSNameOrErrByRef {
				var gotAWSName = ""
				gotErr := convertFunc(&vsRef, &gotAWSName, nil)
				if wantAWSNameOrErr.err != nil {
					assert.EqualError(t, gotErr, wantAWSNameOrErr.err.Error())
				} else {
					assert.NoError(t, gotErr)
					assert.Equal(t, wantAWSNameOrErr.awsName, gotAWSName)
				}
			}
		})
	}
}
