package main

import (
	"bytes"
	"context"
	"encoding/json"

	flag "github.com/spf13/pflag"
	// "runlike/flag"

	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/tidwall/gjson"
)

//1、执行docker命令获取容器信息
//2、解析容器json信息
//3、分解json数据
//4、拼装数据

var (
	container string
	help      bool
	format    bool
	msg       string
	options   []string
)

func parse_hostname() {

	if hostname := gjson.Get(msg, "Config.Hostname").String(); hostname != "" {
		glog.V(3).Info("hostname:", hostname)
		options = append(options, fmt.Sprintf("--hostname=%s ", hostname))
	}

}

func parse_user() {

	if user := gjson.Get(msg, "Config.User").String(); user != "" {
		glog.V(3).Info("user:", user)
		options = append(options, fmt.Sprintf("--user=%s ", user))
	}
}

func parse_macaddress() {

	if macaddress := gjson.Get(msg, "Config.MacAddress").String(); macaddress != "" {
		options = append(options, fmt.Sprintf("--mac-address=%s ", macaddress))
		glog.V(3).Info("Config.MacAddress:", macaddress)
	} else if macaddress = gjson.Get(msg, "NetworkSettings.MacAddress").String(); macaddress != "" {
		options = append(options, fmt.Sprintf("--mac-address=%s ", macaddress))
		glog.V(3).Info("NetworkSettings.MacAddress:", macaddress)
	}

}

func parse_ports() {

	var ports []string

	if port := gjson.Get(msg, "NetworkSettings.Ports").String(); port != "" && port != "{}" {
		ports = append(ports, port)
		glog.V(3).Info("NetworkSettings.Ports:", port)
	}

	if port := gjson.Get(msg, "HostConfig.PortBindings").String(); port != "" && port != "{}" {
		ports = append(ports, port)
		glog.V(3).Info("HostConfig.PortBindings:", port)
	}

	if ports != nil {
		for _, value := range ports {

			// 获取key
			port := JsonToMap(value)
			key := getMapKeys(port)

			glog.V(5).Info("HostConfig.PortBindings:", port)

			containerPort := strings.Split(key[0], "/")[0]
			protocol := strings.Split(key[0], "/")[1]

			optionPart := "-p"
			hostPortPart := ""
			hostnamePart := ""

			protocolPart := ""

			if protocol == "tcp" {
				protocolPart = "tcp"
			} else {
				protocolPart = "udp"
			}

			option := port[key[0]]

			if option == nil {
				optionPart = "--expose="
			} else {

				for _, i2 := range option.([]interface{}) {
					hostIp := i2.(map[string]interface{})["HostIp"]
					hostPort := i2.(map[string]interface{})["HostPort"]

					if hostIp != "0.0.0.0" && hostIp != "::" && hostIp != "" {
						hostnamePart = fmt.Sprintf("%s", hostIp)
					}

					if hostPort != 0 && hostPort != "" {
						hostPortPart = fmt.Sprintf("%s", hostPort)
					}

				}

			}

			if hostnamePart != "" {
				options = append(options, fmt.Sprintf("%s %s:%s:%s %s ", optionPart, hostnamePart, hostPortPart, containerPort, protocolPart))
			} else {
				options = append(options, fmt.Sprintf("%s %s:%s %s ", optionPart, hostPortPart, containerPort, protocolPart))
			}

		}
	}
}

func parse_links() {
	if links := gjson.Get(msg, "HostConfig.Links").String(); links != "" {
		fmt.Println("links 暂时未处理: ", links)
		glog.V(3).Info("links 暂时未处理:", links)
		//for _, link := range links {
		//	src := strings.Split(link, ":")[0]
		//	sdt := strings.Split(link, ":")[1]
		//}

	}

}

func parse_pid() {

	if mode := gjson.Get(msg, "HostConfig.PidMode").String(); mode != "" {
		//fmt.Println("links: ", mode)
		//for _, link := range links {
		//	src := strings.Split(link, ":")[0]
		//	sdt := strings.Split(link, ":")[1]
		//}
		options = append(options, fmt.Sprintf("--pid %s ", mode))
		glog.V(3).Info("HostConfig.PidMode:", mode)

	}

}

func parse_cpuset() {

	if cpusetCpu := gjson.Get(msg, "HostConfig.CpusetCpus").String(); cpusetCpu != "" {

		options = append(options, fmt.Sprintf("--cpuset-cpus= %s", cpusetCpu))
		glog.V(3).Info("HostConfig.CpusetCpus:", cpusetCpu)
	}
	if cpusetMem := gjson.Get(msg, "HostConfig.CpusetMems").String(); cpusetMem != "" {
		options = append(options, fmt.Sprintf("--cpuset-mems= %s ", cpusetMem))
		glog.V(3).Info("HostConfig.CpusetMems:", cpusetMem)
	}

}

func parse_restart() {

	restart := gjson.Get(msg, "HostConfig.RestartPolicy.Name").String()
	if restart == "" {
		return
	} else if restart == "on-failure" {

		if maxRetries := gjson.Get(msg, "HostConfig.RestartPolicy.MaximumRetryCount").Int(); maxRetries > 0 {
			restart = fmt.Sprintf("%s:%d", restart, maxRetries)
		}

	}

	glog.V(3).Info("HostConfig.RestartPolicy.Name:", restart)
	options = append(options, fmt.Sprintf("--restart=%s ", restart))

}

func parse_devices() {

	devices := gjson.Get(msg, "HostConfig.Devices").Array()

	if devices == nil {
		return
	}

	glog.V(5).Info("未处理:", devices)
}

func parse_labels() {

	if labels := gjson.Get(msg, "Config.Labels").String(); labels != "" {

		labelsMap := JsonToMap(labels)

		for label, value := range labelsMap {
			options = append(options, fmt.Sprintf("--label='%s=%s' ", label, value))
		}

		glog.V(5).Info("Config.Labels:", labels)
	}

}

func parse_log() {
	logType := gjson.Get(msg, "HostConfig.LogConfig.Type").String()

	if logType != "json-file" {
		options = append(options, fmt.Sprintf("--log-driver=%s ", logType))
	}

	if logOpts := gjson.Get(msg, "HostConfig.LogConfig.Config").Map(); logOpts != nil {

		for k, v := range logOpts {
			options = append(options, fmt.Sprintf("--log-opt %s=%s ", k, v))
		}

	}

	glog.V(3).Info("HostConfig.LogConfig.Type:", logType)

}

func parse_extra_hosts() {

	if hosts := gjson.Get(msg, "HostConfig.ExtraHosts").Array(); hosts != nil {

		for _, host := range hosts {
			options = append(options, fmt.Sprintf("--add-host %s", host))
		}
		glog.V(3).Info("HostConfig.ExtraHosts:", hosts)
	}

}

func parse_workdir() {

	if workdir := gjson.Get(msg, "Config.WorkingDir").String(); workdir != "" {
		options = append(options, fmt.Sprintf("--workdir=%s ", workdir))
		glog.V(5).Info("Config.WorkingDir:", workdir)
	}

}

func parse_runtime() {

	if runtime := gjson.Get(msg, "HostConfig.Runtime").String(); runtime != "" {
		options = append(options, fmt.Sprintf("--runtime=%s ", runtime))
		glog.V(5).Info("HostConfig.Runtime:", runtime)
	}
}

func parse_memory() {

	if memory := gjson.Get(msg, "HostConfig.Memory").Int(); memory != 0 {
		if memory > 4 {
			options = append(options, fmt.Sprintf("--memory=%d", memory))
		}

		glog.V(5).Info("HostConfig.Memory:", memory)

	}
}

func parse_init() {

	if gjson.Get(msg, "HostConfig.Init").Bool() {
		options = append(options, fmt.Sprintf("--init "))
	}
}

func parse_memory_reservation() {

	if memoryReservation := gjson.Get(msg, "HostConfig.MemoryReservation").Int(); memoryReservation != 0 {
		options = append(options, fmt.Sprintf("--memory-reservation=%d ", memoryReservation))
		glog.V(5).Info("HostConfig.MemoryReservation:", memoryReservation)
	}
}

func multi_option(path, option string) {

	if values := gjson.Get(msg, path).Array(); values != nil {
		for _, val := range values {
			options = append(options, fmt.Sprintf("--%s=%s ", option, val))
		}

		glog.V(5).Info("values:", values, "--", "path")
	}

}

func format_cli() {

	name := gjson.Get(msg, "Name").String()
	name = strings.Split(name, "/")[1]

	if name != "" {
		options = append(options, fmt.Sprintf("--name=%s ", name))
	}

	multi_option("Config.Env", "env")
	multi_option("HostConfig.Binds", "volume")
	multi_option("Config.Volumes", "volume")
	multi_option("HostConfig.VolumesFrom", "volumes-from")
	multi_option("HostConfig.CapAdd", "cap-add")
	multi_option("HostConfig.Dns", "dns")
	multi_option("HostConfig.CapDrop", "cap-drop")

	if networkMode := gjson.Get(msg, "HostConfig.NetworkMode").String(); networkMode != "default" {
		options = append(options, fmt.Sprintf("--network=%s ", networkMode))
	}

	if privileged := gjson.Get(msg, "HostConfig.Privileged").Bool(); privileged {
		options = append(options, "--privileged ")
	}

	if stdoutAttached := gjson.Get(msg, "Config.AttachStdout").Bool(); !stdoutAttached {
		options = append(options, "--detach=true ")
	}

	if gjson.Get(msg, "Config.Tty").Bool() {
		options = append(options, "-t ")
	}

	if image := gjson.Get(msg, "Config.Image").String(); image != "" {
		options = append(options, fmt.Sprintf("%s ", image))
	}

	var cmd []string

	var cmdStr string

	cmdParts := gjson.Get(msg, "Config.Cmd").Array()
	if cmdParts != nil {
		for _, part := range cmdParts {
			cmd = append(cmd, part.String())
		}

		cmdStr = strings.Join(cmd, " ")

	}

	options = append(options, cmdStr)

}

func JsonToMap(str string) map[string]interface{} {

	var tempMap map[string]interface{}

	err := json.Unmarshal([]byte(str), &tempMap)

	if err != nil {
		panic(err)
	}

	return tempMap
}

func getMapKeys(m map[string]interface{}) []string {
	// 数组默认长度为map长度,后面append时,不需要重新申请内存和拷贝,效率很高
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func removeDuplication_map(arr []string) []string {
	// 删除数组重复项
	set := make(map[string]struct{}, len(arr))
	j := 0
	for _, v := range arr {
		_, ok := set[v]
		if ok {
			continue
		}
		set[v] = struct{}{}
		arr[j] = v
		j++
	}
	return arr[:j]
}

func Run(container string) {

	timeout := 5
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout+5)*time.Second)
	defer cancel()

	cmdarray := []string{"-c", fmt.Sprintf("%s %s %s %s ", "docker", "container", "inspect", container)}
	cmd := exec.CommandContext(ctx, "bash", cmdarray...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err.Error())
	}

	msg = string(out)

	glog.V(5).Info("cmd info:", msg)

	execute()

	printCli()
}

func execute() {

	msg = strings.TrimLeft(msg, "[")
	msg = strings.TrimRight(msg, "]")

	parse_init()
	parse_hostname()
	parse_user()
	parse_macaddress()
	parse_ports()
	parse_links()
	parse_devices()
	parse_labels()
	parse_pid()
	parse_cpuset()
	parse_restart()
	parse_log()
	parse_extra_hosts()
	parse_workdir()
	parse_runtime()
	parse_memory()
	parse_memory_reservation()
	format_cli()
}

func printCli() {

	var bt bytes.Buffer

	bt.WriteString("docker run ")

	opt := removeDuplication_map(options)

	lenStr := len(opt)

	var num int
	for _, option := range opt {
		num += 1

		if format && lenStr != num {
			bt.WriteString("\\\n\t" + option)
		} else {
			bt.WriteString(option)
		}

	}

	fmt.Println(bt.String())
}

func init() {

	flag.BoolVarP(&format, "format", "f", false, "Formatted output")
	flag.BoolVarP(&help, "help", "h", false, "this help")

	// flag.BoolVar(&format, "f", false, "Formatted output")
	// flag.BoolVar(&help, "h", false, "this help")

	flag.Usage = usage

}

func main() {

	flag.Parse()
	// flag.ParseToEnd()

	defer glog.Flush()

	var containerList []string
	containerList = flag.Args()

	if help {
		flag.Usage()
		return
	}

	if containerList != nil {

		for _, container := range containerList {
			Run(container)
		}

	} else {
		flag.Usage()
		return
	}

}

func usage() {
	fmt.Fprintf(os.Stderr, `runlike version: 0.0.1
python  https://github.com/lavie/runlike

Usage: runlike [-hf] [-h help] [-f format]

eg:
./runlike -h
./runlike container_id container_name container_id
./runlike -f container_id container_name container_id

`)

	flag.PrintDefaults()

}
