mockgen -destination=./mocks/aws-app-mesh-controller-for-k8s/pkg/webhook/mock_mutator.go github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook Mutator
mockgen -destination=./mocks/aws-app-mesh-controller-for-k8s/pkg/webhook/mock_validator.go github.com/aws/aws-app-mesh-controller-for-k8s/pkg/webhook Validator
mockgen -destination=./mocks/aws-app-mesh-controller-for-k8s/pkg/mesh/mock_membership_designator.go github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh MembershipDesignator
mockgen -destination=./mocks/aws-app-mesh-controller-for-k8s/pkg/virtualnode/mock_membership_designator.go github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualnode MembershipDesignator

