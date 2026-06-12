module dax-performance-main

go 1.24.0

//replace github.com/aws/aws-dax-go-v2 => ../../../DAXGo/DaxGoV2Client

require (
	github.com/aws/aws-dax-go-v2 v1.0.0
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/config v1.29.14
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.45.0
	github.com/aws/aws-sdk-go-v2/service/dax v1.24.2
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.39.1
)

require (
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.67 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.19 // indirect
	github.com/aws/smithy-go v1.22.2 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
)
