package main

import (
	"github.com/aws/aws-cdk-go/awscdk"
	"github.com/aws/aws-cdk-go/awscdk/awsappsync"
	"github.com/aws/aws-cdk-go/awscdk/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/awslambdago"
	"github.com/aws/aws-cdk-go/awscdk/awslogs"
	"github.com/aws/constructs-go/constructs/v3"
	"github.com/aws/jsii-runtime-go"
)

type CdkStackProps struct {
	awscdk.StackProps
}

func NewCdkStack(scope constructs.Construct, id string, props *CdkStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	matchTable := awsdynamodb.NewTable(stack, jsii.String("Matches"),
		&awsdynamodb.TableProps{
			PartitionKey: &awsdynamodb.Attribute{
				Name: jsii.String("match_id"),
				Type: "STRING",
			},
			SortKey: &awsdynamodb.Attribute{
				Name: jsii.String("sort_key"),
				Type: "STRING",
			},
			BillingMode:   "PAY_PER_REQUEST",
			TableName:     jsii.String("Matches"),
			RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
		})
	matchTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("player_id"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("player_id"),
			Type: "STRING",
		},
	})

	playersTable := awsdynamodb.NewTable(stack, jsii.String("Players"),
		&awsdynamodb.TableProps{
			PartitionKey: &awsdynamodb.Attribute{
				Name: jsii.String("player_id"),
				Type: "STRING",
			},
			SortKey: &awsdynamodb.Attribute{
				Name: jsii.String("email"),
				Type: "STRING",
			},
			BillingMode:   "PAY_PER_REQUEST",
			TableName:     jsii.String("Players"),
			RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
		})
	playersTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("email-index"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("email"),
			Type: "STRING",
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	createPlayerFunction := awslambdago.NewGoFunction(stack, jsii.String("CreatePlayer"), &awslambdago.GoFunctionProps{
		Entry:        jsii.String("../player/lambda/create"),
		FunctionName: jsii.String("CreatePlayer"),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(30)),
		Runtime:      awslambda.Runtime_PROVIDED_AL2(),
		LogRetention: awslogs.RetentionDays_ONE_MONTH,
	})

	matchFunction := awslambdago.NewGoFunction(stack, jsii.String("match"), &awslambdago.GoFunctionProps{
		Entry:        jsii.String("../match/lambda"),
		FunctionName: jsii.String("Match"),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(30)),
		Runtime:      awslambda.Runtime_PROVIDED_AL2(),
		LogRetention: awslogs.RetentionDays_ONE_MONTH,
		Environment: &map[string]*string{
			"DYNAMODB_AWS_REGION": matchTable.Env().Region,
			"DYNAMODB_TABLE":      matchTable.TableName(),
		},
	})

	getMatchByPlayerId := awslambdago.NewGoFunction(stack, jsii.String("GetMatchByPlayerId"), &awslambdago.GoFunctionProps{
		Entry:        jsii.String("../match/lambda/get"),
		FunctionName: jsii.String("GetMatchByPlayerId"),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(30)),
		Runtime:      awslambda.Runtime_PROVIDED_AL2(),
		LogRetention: awslogs.RetentionDays_ONE_MONTH,
	})

	playersTable.GrantFullAccess(createPlayerFunction)
	matchTable.GrantFullAccess(matchFunction)
	matchTable.GrantReadData(getMatchByPlayerId)

	graphqlApi := awsappsync.NewGraphqlApi(stack, jsii.String("CloudDart-API"), &awsappsync.GraphqlApiProps{
		Name:   jsii.String("CloudDart-API"),
		Schema: awsappsync.Schema_FromAsset(jsii.String("../graphql/schema.graphql")),
	})

	matchDS := awsappsync.NewLambdaDataSource(stack, jsii.String("MatchFunctions"), &awsappsync.LambdaDataSourceProps{
		Api:            graphqlApi,
		Name:           jsii.String("MatchFunctions"),
		LambdaFunction: matchFunction,
	})

	matchDS.CreateResolver(&awsappsync.BaseResolverProps{
		FieldName: jsii.String("matches"),
		TypeName:  jsii.String("Query"),
		RequestMappingTemplate: awsappsync.MappingTemplate_FromString(
			jsii.String("{\n                    " +
				"\"version\": \"2018-05-29\",\n" +
				"\"operation\": \"Invoke\",\n" +
				"\"payload\": $util.toJson($context.arguments)\n" +
				"}")),
	})

	return stack
}

func main() {
	app := awscdk.NewApp(nil)

	NewCdkStack(app, "CloudDart-ServerStack", &CdkStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}
