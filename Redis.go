package main

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"go-simplejson"
)

var pool = &redis.Pool{
	MaxIdle:     3, /*最大的空闲连接数*/
	MaxActive:   8, /*最大的激活连接数*/
	Dial: func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", "localhost:6379")
		if err != nil {
			fmt.Println("redis数据库连接出错,", err)
			return nil, err
		}
		return c, nil
	},
}


// func LinkRedis() {
// 	conn, err := redis.Dial("tcp", "localhost:6379")
// 	if err != nil {
// 		fmt.Println("redis数据库连接出错,", err)
// 		return
// 	}

// 	_, err = conn.Do("SET", "username", "nick")
// 	if err != nil {
// 	fmt.Println("redis set failed:", err)
// 	}
// 	username, err := redis.String(conn.Do("GET", "username"))
// 	if err != nil {
// 	fmt.Println("redis get failed:", err)
// 	} else {
// 	fmt.Printf("Got username %v \n", username)
// 	}
// }

func SetRedis(key string,value string){
	conn := pool.Get()
	defer conn.Close()
	_, err = conn.Do("SET",key, value)
	if err != nil {
		fmt.Println("redis set failed:", err)
	}
}

func GetRedis(key string)(value string){
	conn := pool.Get()
	defer conn.Close()
	data, err := redis.String(conn.Do("GET", key))
	if err != nil {
		fmt.Println("redis set failed:", err)
	}
	return data
}

func CheckRedis(key string)(exists bool){
	conn := pool.Get()
	defer conn.Close()
	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		fmt.Println("illegal exception:", err)
	}
	return exists
}

func FindMaxExp(name string,exp int)(isTop bool){
	names := []string{"何朝阳", "周宏亮" ,"陈晨"}
	for _,j := range names{
		if j == name{
			continue
		}
		userDat := GetRedis(name)
		js, err := simplejson.NewJson([]byte(userDat))
		panicErr(err)
		pkExp := js.Get("exp").MustInt()
		if pkExp > exp {
			return false
		}
	}
	return true
}

