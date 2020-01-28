package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	_ "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"os"
)

const (
	AppVersion = "0.0.1"
)

var (
	ParamName = os.Getenv("PARAMETER_NAME")

	argVersion = flag.Bool("version", false, "バージョンを出力.")
	argRatio   = flag.Float64("rasio", 0, "shared_buffers をメモリに対してどの程度割り当てるか (Default = 50%)")
)

func main() {
	flag.Parse()

	if *argVersion {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	current_value := printValue()

	if current_value == "" {
		fmt.Printf("現在の値を取得出来ません.")
		os.Exit(1)
	}

	var latest_value string
	if *argRatio != 0 {
		latest_value = genParameterValue(*argRatio)
	} else {
		fmt.Println("`rasio` パラメータを指定して下さい.")
		fmt.Printf("現在の値は: %s\n", current_value)
		os.Exit(1)
	}

	fmt.Printf("現在の値は: %s\n", current_value)
	fmt.Printf("新しい値は: %s\n", latest_value)
	fmt.Print("パラメータを更新しますか? (y/n): ")
	var stdin string
	fmt.Scan(&stdin)
	switch stdin {
	case "y", "Y":
		fmt.Println("パラメータを更新中...")
		modifyValue(latest_value)
	case "n", "N":
		fmt.Println("処理を停止します.")
		os.Exit(0)
	default:
		fmt.Println("処理を停止します.")
		os.Exit(0)
	}
}

func genParameterValue(value float64) string {
	v := 8192.0 / value
	parameter := fmt.Sprintf("{DBInstanceClassMemory/%s}", fmt.Sprint(int(v)))

	return parameter
}

func modifyValue(param string) {
	svc := rds.New(session.New())

	input := &rds.ModifyDBParameterGroupInput{
		DBParameterGroupName: aws.String(ParamName),
		Parameters: []*rds.Parameter{
			{
				ApplyMethod:    aws.String("pending-reboot"),
				ParameterName:  aws.String("shared_buffers"),
				ParameterValue: aws.String(param),
			},
		},
	}

	result, err := svc.ModifyDBParameterGroup(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case rds.ErrCodeDBParameterGroupNotFoundFault:
				fmt.Println(rds.ErrCodeDBParameterGroupNotFoundFault, aerr.Error())
			case rds.ErrCodeInvalidDBParameterGroupStateFault:
				fmt.Println(rds.ErrCodeInvalidDBParameterGroupStateFault, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println(result)
}

func printValue() string {
	svc := rds.New(session.New())

	input := &rds.DescribeDBParametersInput{
		DBParameterGroupName: aws.String(ParamName),
		Source:               aws.String("system"),
	}

	result, err := svc.DescribeDBParameters(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case rds.ErrCodeDBParameterGroupNotFoundFault:
				fmt.Println(rds.ErrCodeDBParameterGroupNotFoundFault, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return ""
	}

	var para string
	for _, p := range result.Parameters {
		if *p.ParameterName == "shared_buffers" {
			para = *p.ParameterValue
		}
	}

	if para == "" {
		return ""
	}

	return para
}
