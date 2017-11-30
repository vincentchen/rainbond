// RAINBOND, Application Management Platform
// Copyright (C) 2014-2017 Goodrain Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Rainbond,
// one or multiple Commercial Licenses authorized by Goodrain Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package cmd
import (
	"github.com/urfave/cli"
	"github.com/Sirupsen/logrus"
	"github.com/goodrain/rainbond/pkg/grctl/clients"
	"time"
	"fmt"
	"strings"
	"encoding/json"
	"os"
	"github.com/pquerna/ffjson/ffjson"
	"io/ioutil"
	"github.com/goodrain/rainbond/pkg/node/api/model"
)

func GetCommand(status bool)[]cli.Command  {
	c:=[]cli.Command{
		{
			Name:  "compute",
			Usage: "安装计算节点 compute -h",
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "nodes",
					Usage: "hostID1 hostID2 ...,空表示全部",
				},
			},
			Action: func(c *cli.Context) error {
				return Task(c,"check_compute_services",status)
			},
			Subcommands:[]cli.Command{
				{
					Name:  "storage_client",
					Usage: "step 1 storage_client",
					Action: func(c *cli.Context) error {
						return Task(c,"install_storage_client",status)
					},
				},
				{
					Name:  "kubelet",
					Usage: "need storage_client",
					Action: func(c *cli.Context) error {
						return Task(c,"install_kubelet",status)
					},
				},
				{
					Name:  "network_compute",
					Usage: "need storage_client,kubelet",
					Action: func(c *cli.Context) error {
						return Task(c,"install_network_compute",status)
					},
				},
			},

		},
		{
			Name:  "manage_base",
			Usage: "安装管理节点基础服务。 manage_base -h",
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "nodes",
					Usage: "hostID1 hostID2 ...,空表示全部",
				},
			},
			Action: func(c *cli.Context) error {
				return Task(c,"check_manage_base_services",status)
			},
			Subcommands:[]cli.Command{
				{
					Name:  "docker",
					Usage: "step 1 安装docker",
					Action: func(c *cli.Context) error {
						return Task(c,"install_docker",status)
					},
				},
				{
					Name:  "db",
					Usage: "step 2 安装db",
					Action: func(c *cli.Context) error {
						return Task(c,"install_db",status)
					},

				},
				{
					Name:  "base_plugins",
					Usage: "step 3 基础插件",
					Action: func(c *cli.Context) error {
						return Task(c,"install_base_plugins",status)
					},

				},
				{
					Name:  "acp_plugins",
					Usage: "step 4 acp插件",
					Action: func(c *cli.Context) error {
						return Task(c,"install_acp_plugins",status)
					},

				},
			},
		},
		{
			Name:  "manage",
			Usage: "安装管理节点。 manage -h",
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:  "nodes",
					Usage: "hostID1 hostID2 ...,空表示全部",
				},
			},
			Subcommands:[]cli.Command{
				{
					Name:  "storage",
					Usage: "step 1 安装存储",
					Action: func(c *cli.Context) error {
						return Task(c,"install_storage",status)
					},

				},
				{
					Name:  "k8s",
					Usage: "need storage",
					Action: func(c *cli.Context) error {
						return Task(c,"install_k8s",status)
					},

				},
				{
					Name:  "network",
					Usage: "need storage,k8s",
					Action: func(c *cli.Context) error {
						return Task(c,"install_network",status)
					},

				},
				{
					Name:  "plugins",
					Usage: "need storage,k8s,network",
					Action: func(c *cli.Context) error {
						return Task(c,"install_plugins",status)
					},

				},
			},
			Action:func(c *cli.Context) error {
				return Task(c,"check_manage_services",status)
			},
		},
	}
	return c
}


func NewCmdAddTask() cli.Command {
	c:=cli.Command{
		Name:  "add_task",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "filepath",
				Usage: "task path",
			},

		},
		Usage: "添加task。grctl add_task",
		Action: func(c *cli.Context) error {
			file:=c.String("filepath")
			if file!="" {
				task:=loadFile(file)
				err:=clients.NodeClient.Tasks().Add(task)
				if err != nil {
					logrus.Errorf("error add task from file,details %s",err.Error())
					return nil
				}

			}else {
				logrus.Errorf("error get task from path")
			}
			return nil
		},
	}
	return c
}

func loadFile(path string) *model.Task{
	taskBody, err := ioutil.ReadFile(path)
	if err != nil {
		logrus.Errorf("read static task file %s error.%s", path, err.Error())
		return nil
	}
	var filename string
	index := strings.LastIndex(path, "/")
	if index < 0 {
		filename = path
	}
	filename = path[index+1:]
	if strings.Contains(filename, "group") {
		var group model.TaskGroup
		if err := ffjson.Unmarshal(taskBody, &group); err != nil {
			logrus.Errorf("unmarshal static task file %s error.%s", path, err.Error())
			return nil
		}
		if group.ID == "" {
			group.ID = group.Name
		}
		if group.Name == "" {
			logrus.Errorf("task group name can not be empty. file %s", path)
			return nil
		}
		if group.Tasks == nil {
			logrus.Errorf("task group tasks can not be empty. file %s", path)
			return nil
		}
		//ScheduleGroup(nil, &group)
		logrus.Infof("Load a static group %s.", group.Name)
	}
	if strings.Contains(filename, "task") {
		var task model.Task
		if err := ffjson.Unmarshal(taskBody, &task); err != nil {
			logrus.Errorf("unmarshal static task file %s error.%s", path, err.Error())
			return nil
		}
		if task.ID == "" {
			task.ID = task.Name
		}
		if task.Name == "" {
			logrus.Errorf("task name can not be empty. file %s", path)
			return nil
		}
		if task.Temp == nil {
			logrus.Errorf("task [%s] temp can not be empty.", task.Name)
			return nil
		}
		if task.Temp.ID == "" {
			task.Temp.ID = task.Temp.Name
		}
		//err:=t.AddTask(&task)
		//if err != nil {
		//	logrus.Errorf("error add task,details %s",err.Error())
		//}
		return &task
	}
	return nil
}
func NewCmdInstall() cli.Command {
	c:=cli.Command{
		Name:  "install",
		Usage: "安装命令相关子命令。grctl install  -h",
		Flags: []cli.Flag{
			cli.StringSliceFlag{
				Name:  "nodes",
				Usage: "hostID1 hostID2 ...,空表示全部",
			},
		},
		Subcommands:GetCommand(false),
	}
	return c
}
//func NewCmdStatus() cli.Command {
//	c:=cli.Command{
//		Name:  "status",
//		Usage: "状态命令相关子命令。grctl status  -h",
//		Flags: []cli.Flag{
//			cli.StringSliceFlag{
//				Name:  "nodes",
//				Usage: "hostID1 hostID2 ...,空表示全部",
//			},
//		},
//		Subcommands:GetCommand(true),
//	}
//	return c
//}

func Status(task string) {
	taskE:=clients.NodeClient.Tasks().Get(task)
	lastState:=""
	checkFail:=0
	for checkFail<3  {
		time.Sleep(3*time.Second)
		status,err:=taskE.Status()
		if err != nil||status==nil {
			logrus.Warnf("error get task status,retry")
			checkFail+=1
			if err!=nil {
				logrus.Errorf("error get task status ,details %s",err.Error())
			}
			continue
		}
		for _,v:=range status.Status{
			if v.Status!="complete" {
				if lastState!=v.Status{
					fmt.Printf("task %s is %s\n",task,v.Status)
				}else{
					fmt.Print("..")
				}
				lastState=v.Status

				if strings.Contains(v.Status, "error")||strings.Contains(v.CompleStatus,"Failure")||strings.Contains(v.CompleStatus,"Unknow") {
					checkFail+=1
					//todo add continue ,code behind this line should be placed in line 254
					continue
				}
			}else {
				fmt.Printf("task %s is %s %s\n",task,v.Status,v.CompleStatus)
				lastState=v.Status
				taskFinished:=clients.NodeClient.Tasks().Get(task)
				var  nextTasks []string
				for _,v:=range taskFinished.Task.OutPut{
					for _,sv:=range v.Status{
						if sv.NextTask == nil ||len(sv.NextTask)==0{
							continue
						}else{
							for _,v:=range sv.NextTask{
								nextTasks=append(nextTasks,v)
							}
						}
					}
				}
				if len(nextTasks) > 0 {
					fmt.Printf("next will install %v \n",nextTasks)
					for _,v:=range nextTasks{
						Status(v)
					}
				}
				return
			}
		}
		checkFail=0
	}
	fmt.Printf("task %s 's output \n",taskE.TaskID)
	tb,_:=json.Marshal(taskE)
	fmt.Println("task failed,details is %s",string(tb))
	for _,v:=range taskE.Task.OutPut{
		fmt.Println("on %s :\n %s",v.NodeID,v.Body)
	}
	os.Exit(1)
}

func Task(c *cli.Context,task string,status bool) error   {

	nodes:=c.StringSlice("nodes")
	taskEntity:=clients.NodeClient.Tasks().Get(task)
	if taskEntity==nil {
		logrus.Errorf("error get task entity from server,please check server api")
		return nil
	}
	err:=taskEntity.Exec(nodes)
	if err != nil {
		logrus.Errorf("error exec task:%s,details %s",task,err.Error())
		return err
	}
	Status(task)

	return nil
}
