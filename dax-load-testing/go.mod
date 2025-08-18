module dax-load-testing

go 1.24.0

replace github.com/aws/aws-dax-go-v2 => ../../../DAXGo/DaxGoV2Client

require (
	github.com/aws/aws-dax-go-v2 v1.0.0
	github.com/aws/aws-sdk-go-v2 v1.37.1
	github.com/aws/aws-sdk-go-v2/config v1.28.8
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.45.4
	github.com/aws/aws-sdk-go-v2/service/dax v1.25.1
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.44.1
)

require (
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.49 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.4 // indirect
	github.com/aws/smithy-go v1.22.5 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
)
