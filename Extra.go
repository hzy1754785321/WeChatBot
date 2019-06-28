package main

import (
	"fmt"
	"io/ioutil"
	m "itchat4go/model"
	"path/filepath"
	"github.com/go-yaml/yaml"
	"os"
)

type conf struct {
	ApiKey string `yaml:"apiKey"`
}

var (
	uuid       string
	err        error
	loginMap   m.LoginMap
	contactMap map[string]m.User
	groupMap   map[string][]m.User /* 关键字为key的，群组数组 */
)

var userData string = `{
	         "userName": "",
	         "city": "",
	         "signTime": "",
			 "signCount": 0,
			 "Friendliness": 0
	 }`


type userCache struct {
	userName string
	city string
	signTime string
	signCount int
	Friendliness int
	FriendlinessAdd int
	}

var datJSON string = `{
	"reqType":0,
    "perception": {
        "inputText": {
            "text": ""
        },
        "inputImage": {
            "url": "imageUrl"
        },
        "selfInfo": {
            "location": {
                "city": "上海",
                "province": "上海",
                "street": "桂林路"
            }
        }
    },
    "userInfo": {
        "apiKey": "",
        "userId": "hzy"
    }
}`

func getFileSize(filename string) int64 {
    var result int64
    filepath.Walk(filename, func(path string, f os.FileInfo, err error) error {
        result = f.Size()
        return nil
    })
    return result
}

func (c *conf) getConf() *conf {
	yamlFile, err := ioutil.ReadFile("conf.yaml")
	if err != nil {
		fmt.Println(err.Error())
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		fmt.Println(err.Error())
	}
	return c
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
