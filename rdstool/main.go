package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	_ "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/manifoldco/promptui"
	"github.com/olekukonko/tablewriter"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	AppVersion = "0.0.1"
)

var (
	argVersion         = flag.Bool("version", false, "バージョンを出力.")
	argModify          = flag.Bool("modify", false, "パラメータの更新.")
	argFailover        = flag.Bool("failover", false, "DB インスタンスのフェイルオーバーを開始.")
	argInstances       = flag.Bool("instances", false, "クラスタの DB インスタンス一覧を取得.")
	argInstance        = flag.String("instance", "", "DB インスタンス名を指定.")
	argCluster         = flag.String("cluster", "", "Aurora クラスタ名を指定.")
	argRestart         = flag.Bool("restart", false, "DB インスタンスの再起動を実施.")
	argParamGroup      = flag.String("param-group", "", "DB パラメータグループの名前を指定")
	argParamNamePrefix = flag.String("param-name-prefix", "", "パラメータの名前を指定")
	argRatio           = flag.Float64("rasio", 0, "パラメータの値について, メモリに対してどの程度割り当てるかを指定. (Default = 50%)")

	svc = rds.New(session.New())
)

func main() {
	flag.Parse()

	if *argVersion {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	var clusterName string
	if *argCluster == "" {
		clusterName = os.Getenv("CLUSTER_NAME")
	} else if *argCluster != "" {
		clusterName = *argCluster
	} else {
		fmt.Println("`-cluster` パラメータを指定して下さい.")
		os.Exit(1)
	}

	var paramGroup string
	if *argParamGroup == "" {
		paramGroup = os.Getenv("PARAMETER_NAME")
	} else if *argParamGroup != "" {
		paramGroup = *argParamGroup
	} else {
		fmt.Println("`-param-group` パラメータを指定して下さい.")
		os.Exit(1)
	}

	dbInstances := getClusterInstances(clusterName)
	if *argInstances {
		printTable(dbInstances, "instance")
		os.Exit(0)
	}

	if !*argModify && *argParamNamePrefix != "" {
		params := printParams(paramGroup, *argParamNamePrefix)
		printTable(params, "param")
		os.Exit(0)
	}

	if *argFailover {
		targetDBInstance := selectFailoverTarget(dbInstances)
		fmt.Printf("DB クラスタ %s をフェイルーバーします. フェイルオーバー先は %v です.\n", clusterName, targetDBInstance)
		fmt.Printf("処理を継続しますか? (y/n): ")
		var stdin string
		fmt.Scan(&stdin)
		switch stdin {
		case "y", "Y":
			dbClusterFailoverStatus := executeClusterFailover(clusterName, targetDBInstance)
			if dbClusterFailoverStatus == "" {
				fmt.Printf("DB クラスタのフェイルーバーに失敗しました.")
				os.Exit(1)
			}
			fmt.Printf("DB クラスタのフェイルオーバー実行中")
			for {
				st, _ := getInstanceStatus(targetDBInstance)
				dbInstances := getClusterInstances(clusterName)
				w := getWriteInstance(dbInstances)
				if st == "available" && w == targetDBInstance {
					fmt.Printf("\nDB クラスタフェイルオーバー完了.\n")
					os.Exit(0)
				}
				fmt.Printf(".")
				time.Sleep(time.Second * 5)
			}
		case "n", "N":
			fmt.Println("処理を停止します.")
			os.Exit(0)
		default:
			fmt.Println("処理を停止します.")
			os.Exit(0)
		}
	}

	if *argRestart && *argInstance != "" {
		fmt.Printf("DB インタンス %s を再起動します.\n", *argInstance)
		fmt.Printf("処理を継続しますか? (y/n): ")
		var stdin string
		fmt.Scan(&stdin)
		switch stdin {
		case "y", "Y":
			dbInstanceStatus := restartDBInstance(*argInstance, *argFailover)
			if dbInstanceStatus == "" {
				fmt.Printf("DB インスタンスの再起動に失敗しました.")
				os.Exit(1)
			}
			fmt.Printf("DB インスタンスを再起動中")
			for {
				st, _ := getInstanceStatus(*argInstance)
				if st == "available" {
					fmt.Printf("\nDB インスタンス再起動完了.\n")
					os.Exit(0)
				}
				fmt.Printf(".")
				time.Sleep(time.Second * 5)
			}
		case "n", "N":
			fmt.Println("処理を停止します.")
			os.Exit(0)
		default:
			fmt.Println("処理を停止します.")
			os.Exit(0)
		}
	}

	if *argModify && *argParamNamePrefix != "" {
		var latest_value string
		if *argRatio != 0 {
			latest_value = genParameterValue(*argRatio)
		} else {
			fmt.Println("`-rasio` パラメータを指定して下さい.")
			os.Exit(1)
		}

		params := printParams(paramGroup, *argParamNamePrefix)
		if len(params) != 1 {
			fmt.Println("DB パラメータの指定に誤りがあります. パラメータ名を確認して下さい.")
			os.Exit(1)
		}
		fmt.Printf("DB パラメータを更新します. DB パラメータ %s の新しい値は: %s\n", *argParamNamePrefix, latest_value)
		fmt.Printf("処理を継続しますか? (y/n): ")
		var stdin string
		fmt.Scan(&stdin)
		switch stdin {
		case "y", "Y":
			dbInstance := getWriteInstance(dbInstances)
			fmt.Println("DB パラメータを更新します.")
			modifyValue(paramGroup, latest_value)
			fmt.Printf("DB パラメータ更新中")
			for {
				if getParameterStatus(dbInstance, paramGroup) == "pending-reboot" {
					fmt.Printf("\nDB パラメータ更新完了. DB インスタンスの再起動が必要です.\n")
					break
				} else if getParameterStatus(dbInstance, paramGroup) == "" {
					fmt.Println("DB パラメータの更新に失敗しました.")
					os.Exit(1)
				}
				fmt.Printf(".")
				time.Sleep(time.Second * 5)
			}
			fmt.Printf("DB インタンス %s を再起動します.\n", *argInstance)
			fmt.Printf("処理を継続しますか? (y/n): ")
			var stdin string
			fmt.Scan(&stdin)
			switch stdin {
			case "y", "Y":
				dbInstanceStatus := restartDBInstance(*argInstance, *argFailover)
				if dbInstanceStatus == "" {
					fmt.Printf("DB インスタンスの再起動に失敗しました.")
					os.Exit(1)
				}
				fmt.Printf("DB インスタンスを再起動中")
				for {
					st, _ := getInstanceStatus(*argInstance)
					if st == "available" {
						fmt.Printf("\nDB インスタンス再起動完了.\n")
						os.Exit(0)
					}
					fmt.Printf(".")
					time.Sleep(time.Second * 5)
				}
			case "n", "N":
				fmt.Println("処理を停止します.")
				os.Exit(0)
			default:
				fmt.Println("処理を停止します.")
				os.Exit(0)
			}
		case "n", "N":
			fmt.Println("処理を停止します.")
			os.Exit(0)
		default:
			fmt.Println("処理を停止します.")
			os.Exit(0)
		}
	}
	flag.PrintDefaults()
}

func printTable(data [][]string, t string) {
	table := tablewriter.NewWriter(os.Stdout)
	if t == "instance" {
		table.SetHeader([]string{"InstanceIdentifier", "InstanceStatus", "Writer", "ParameterApplyStatus", "ClusterParameterGroupStatus", "PromotionTier"})
		for _, value := range data {
			if value[2] == "true" {
				for i, e := range value {
					value[i] = fmt.Sprintf("\x1b[31m%s\x1b[0m", e)
				}
			}
			table.Append(value)
		}
	} else {
		table.SetHeader([]string{"ParameterName", "ParameterValue"})
		table.AppendBulk(data)
	}

	table.Render()
}

func genParameterValue(value float64) string {
	v := 8192.0 / value
	parameter := fmt.Sprintf("{DBInstanceClassMemory/%s}", fmt.Sprint(int(v)))

	return parameter
}

func getParameterStatus(dbInstance string, paramGroup string) string {
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(dbInstance),
	}

	result, err := svc.DescribeDBInstances(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return ""
	}

	var st string
	for _, r := range result.DBInstances {
		for _, p := range r.DBParameterGroups {
			if *p.DBParameterGroupName == paramGroup {
				st = *p.ParameterApplyStatus
			}
		}
	}

	return st
}

// Restart DB Instance
func restartDBInstance(dbInstance string, failover bool) string {
	input := &rds.RebootDBInstanceInput{
		DBInstanceIdentifier: aws.String(dbInstance),
		// Aurora クラスタではエラーになるので, 何らかの回避方法で RDS と Aurora 両方に対応出来るようにする... いつか
		// ForceFailover:    aws.Bool(failover),
	}

	result, err := svc.RebootDBInstance(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return ""
	}

	st := *result.DBInstance.DBInstanceStatus
	return st
}

// Failover DB Cluster
func executeClusterFailover(clusterName string, targetDBInstance string) string {
	input := &rds.FailoverDBClusterInput{
		DBClusterIdentifier:        aws.String(clusterName),
		TargetDBInstanceIdentifier: aws.String(targetDBInstance),
	}

	result, err := svc.FailoverDBCluster(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return ""
	}

	st := *result.DBCluster.Status
	return st
}

func getClusterInstances(clusterName string) [][]string {
	input := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterName),
	}

	result, err := svc.DescribeDBClusters(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return nil
	}

	// fmt.Println(result.DBClusters[0].DBClusterMembers)
	var instances [][]string
	for _, i := range result.DBClusters[0].DBClusterMembers {
		tier := strconv.FormatInt(*i.PromotionTier, 10)
		st, ps := getInstanceStatus(*i.DBInstanceIdentifier)
		instance := []string{
			*i.DBInstanceIdentifier,
			st,
			strconv.FormatBool(*i.IsClusterWriter),
			ps,
			*i.DBClusterParameterGroupStatus,
			tier,
		}
		instances = append(instances, instance)
	}
	return instances
}

func getWriteInstance(dbInstances [][]string) string {
	var writer string
	for _, i := range dbInstances {
		if i[2] == "true" {
			writer = i[0]
		}
	}
	return writer
}

func selectFailoverTarget(dbInstances [][]string) string {
	var targets []string
	for _, i := range dbInstances {
		if i[2] != "true" {
			targets = append(targets, i[0])
		}
	}
	prompt := promptui.Select{
		Label: "フェイルオーバー先の DB インスタンスを選択して下さい.",
		Items: targets,
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return ""
	}

	return result
}

func getInstanceStatus(dbInstance string) (string, string) {
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(dbInstance),
	}

	result, err := svc.DescribeDBInstances(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return "", ""
	}

	st := *result.DBInstances[0].DBInstanceStatus
	ps := *result.DBInstances[0].DBParameterGroups[0].ParameterApplyStatus
	return st, ps
}

func modifyValue(paramGroup string, param string) {
	input := &rds.ModifyDBParameterGroupInput{
		DBParameterGroupName: aws.String(paramGroup),
		Parameters: []*rds.Parameter{
			{
				ApplyMethod:    aws.String("pending-reboot"),
				ParameterName:  aws.String("shared_buffers"),
				ParameterValue: aws.String(param),
			},
		},
	}

	_, err := svc.ModifyDBParameterGroup(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}
		return
	}
}

func printParams(paramGroup string, paramNamePrefix string) [][]string {
	input := &rds.DescribeDBParametersInput{
		DBParameterGroupName: aws.String(paramGroup),
	}

	var params [][]string
	for {
		result, err := svc.DescribeDBParameters(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				fmt.Println(aerr.Error())
			} else {
				fmt.Println(err.Error())
			}
			return nil
		}
		for _, p := range result.Parameters {
			if paramNamePrefix != "" {
				if strings.Contains(*p.ParameterName, paramNamePrefix) {
					var pv string
					if p.ParameterValue == nil {
						pv = "N/A"
					} else {
						pv = *p.ParameterValue
					}
					param := []string{*p.ParameterName, pv}
					params = append(params, param)
				}
			} else {
				// Bug
				param := []string{*p.ParameterName, *p.ParameterValue}
				params = append(params, param)
			}
		}
		if result.Marker == nil {
			break
		}
		input.SetMarker(*result.Marker)
		continue
	}

	return params
}
