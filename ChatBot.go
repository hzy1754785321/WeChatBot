package main

import (
	//	"bytes"
	"fmt"
	"go-simplejson"
	"io/ioutil"
	e "itchat4go/enum"
	m "itchat4go/model"
	s "itchat4go/service"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"

	// "regexp"
	"strings"
	"time"
)

func GetChat() {
	/* 从微信服务器获取UUID */
	uuid, err = s.GetUUIDFromWX()
	if err != nil {
		panicErr(err)
	}

	/* 根据UUID获取二维码 */
	err = s.DownloadImagIntoDir(e.QRCODE_URL+uuid, "./qrcode")
	panicErr(err)
	cmd := exec.Command(`cmd`, `/c start ./qrcode/qrcode.jpg`)
	err = cmd.Run()
	panicErr(err)

	/* 轮询服务器判断二维码是否扫过暨是否登陆了 */
	for {
		fmt.Println("正在验证登陆... ...")
		status, msg := s.CheckLogin(uuid)

		if status == 200 {
			fmt.Println("登陆成功,处理登陆信息...")
			loginMap, err = s.ProcessLoginInfo(msg)
			if err != nil {
				panicErr(err)
			}

			fmt.Println("登陆信息处理完毕,正在初始化微信...")
			err = s.InitWX(&loginMap)
			if err != nil {
				panicErr(err)
			}

			fmt.Println("初始化完毕,通知微信服务器登陆状态变更...")
			err = s.NotifyStatus(&loginMap)
			if err != nil {
				panicErr(err)
			}

			fmt.Println("通知完毕,本次登陆信息：")
			fmt.Println(e.SKey + "\t\t" + loginMap.BaseRequest.SKey)
			fmt.Println(e.PassTicket + "\t\t" + loginMap.PassTicket)
			break
		} else if status == 201 {
			fmt.Println("请在手机上确认")
		} else if status == 408 {
			fmt.Println("请扫描二维码")
		} else {
			fmt.Println(msg)
		}
	}

	fmt.Println("开始获取联系人信息...")
	contactMap, err = s.GetAllContact(&loginMap)
	if err != nil {
		panicErr(err)
	}
	fmt.Printf("成功获取 %d个 联系人信息,开始整理群组信息...\n", len(contactMap))

	groupMap = s.MapGroupInfo(contactMap)
	groupSize := 0
	for _, v := range groupMap {
		groupSize += len(v)
	}
	fmt.Printf("整理完毕，共有 %d个 群组是焦点群组，它们是：\n", groupSize)
	for key, v := range groupMap {
		fmt.Println(key)
		for _, user := range v {
			fmt.Println("========>" + user.NickName)
		}
	}

	fmt.Println("开始监听消息响应...")
	var retcode, selector int64

	regGroup := regexp.MustCompile(`^@@`)
	for {
		retcode, selector, err = s.SyncCheck(&loginMap)
		if err != nil {
			fmt.Println(retcode, selector)
			if retcode == 1101 {
				fmt.Println("帐号已在其他地方登陆，程序将退出。")
				os.Exit(2)
			}
			continue
		}

		if retcode == 0 && selector != 0 {
			fmt.Printf("selector=%d,有新消息产生,准备拉取...\n", selector)
			wxRecvMsges, err := s.WebWxSync(&loginMap)
			panicErr(err)
			for i := 0; i < wxRecvMsges.MsgCount; i++ {
				if wxRecvMsges.MsgList[i].MsgType == 1 {
					//	imageFile, err := os.Open("D:/Go/work/WeChatBot/test.jpg")

					// imageStream, err := ioutil.ReadFile("D:/Go/Work/WeChatBot/test.jpg")
					// size := getFileSize("D:/Go/Work/WeChatBot/test.jpg")
					// panicErr(err)
					// retImg, err := s.UploadImg(&loginMap, imageStream,size)
					// panicErr(err)
					// println(retImg.MediaId)
					regAt := "@"
					/* 普通文本消息 */
					fmt.Println(wxRecvMsges.MsgList[i].FromUserName+":", wxRecvMsges.MsgList[i].Content)
					regAt += loginMap.SelfNickName
					tmp := strings.Split(wxRecvMsges.MsgList[i].Content, regAt)
					var reply string
					var cont string
					if len(tmp) > 1 {
						cont = tmp[1]
					} else {
						cont = tmp[0]
					}
					if strings.Contains(wxRecvMsges.MsgList[i].Content, regAt) && strings.Contains(cont, "签到") {
						tmpId := tmp[0]
						userId := strings.Replace(tmpId, ":", "", -1)
						userId = strings.Replace(userId, "<br/>", "", -1)
						var user m.User
						var userTmp userCache
						var pass bool
						for _, v := range contactMap {
							if v.UserName == userId {
								user = v
								break
							}
						}
						if user.RemarkName != "" {
							if !CheckRedis(user.RemarkName) {
								dat := []byte(datJSON)
								jsDat, err := simplejson.NewJson(dat)
								panicErr(err)
								jsDat.Set("userName", user.NickName)
								jsDat.Set("city", user.City)
								todayTime := time.Now().Format("2006-01-02 15:04:05")
								jsDat.Set("signTime", todayTime)
								jsDat.Set("signCount", 1)
								friendliNessAdd := rand.Intn(30)
								friendliNess := friendliNessAdd
								jsDat.Set("Friendliness", friendliNess)
								userTmp.userName = user.NickName
								userTmp.city = user.City
								userTmp.signTime = todayTime
								userTmp.signCount = 1
								userTmp.Friendliness = friendliNess
								userTmp.FriendlinessAdd = friendliNessAdd
								unjs, err := jsDat.MarshalJSON()
								panicErr(err)
								SetRedis(user.RemarkName, string(unjs))
								pass = true
							} else {
								userDat := GetRedis(user.RemarkName)
								js, err := simplejson.NewJson([]byte(userDat))
								panicErr(err)
								signTimeTmp := js.Get("signTime").MustString()
								signTime, _ := time.ParseInLocation("2006-01-02 15:04:05", signTimeTmp, time.Local)
								if signTime.Day() != time.Now().Day() {
									pass = true
								} else if signTime.Month() != time.Now().Month() {
									pass = true
								}
								if pass {
									signCount := js.Get("signCount").MustInt()
									friendliNess := js.Get("Friendliness").MustInt()
									friendliNessAdd := rand.Intn(30)
									friendliNess += friendliNessAdd
									js.Set("Friendliness", friendliNess)
									js.Set("userName", user.NickName)
									js.Set("city", user.City)
									todayTime := time.Now().Format("2006-01-02 15:04:05")
									js.Set("signTime", todayTime)
									signCount = signCount + 1
									js.Set("signCount", signCount)
									userTmp.userName = user.NickName
									userTmp.city = user.City
									userTmp.signTime = todayTime
									userTmp.signCount = signCount
									userTmp.Friendliness = friendliNess
									userTmp.FriendlinessAdd = friendliNessAdd
									unjs, err := js.MarshalJSON()
									panicErr(err)
									SetRedis(user.RemarkName, string(unjs))
								}
							}
							var contents string
							content1 := fmt.Sprintf("@%s 签到成功！今天已经是你的第%d次签到啦～魔理沙酱对乃的好感度[+%d]2019年，要加油哦~", userTmp.userName, userTmp.signCount, userTmp.FriendlinessAdd)
							content2 := fmt.Sprintf("@%s 真是健忘呐！！今天签到过了哟！", user.NickName)
							if pass {
								contents = content1
							} else {
								contents = content2
							}
							wxSendMsg := m.WxSendMsg{}
							wxSendMsg.Type = 1
							wxSendMsg.Content = contents
							wxSendMsg.FromUserName = wxRecvMsges.MsgList[i].ToUserName
							wxSendMsg.ToUserName = wxRecvMsges.MsgList[i].FromUserName
							wxSendMsg.LocalID = fmt.Sprintf("%d", time.Now().Unix())
							wxSendMsg.ClientMsgId = wxSendMsg.LocalID

							/* 加点延时，避免消息次序混乱，同时避免微信侦察到机器人 */
							time.Sleep(time.Second)

							go s.SendMsg(&loginMap, wxSendMsg)

							if pass && userTmp.city != "" {
								msg := GetWeather(userTmp.city)
								content := fmt.Sprintf("下面是魔理沙酱精心为你收集的天气情况哦，满怀感激的收下吧~\n%s", msg)
								wxSendMsg := m.WxSendMsg{}
								wxSendMsg.Type = 1
								wxSendMsg.Content = content
								wxSendMsg.FromUserName = wxRecvMsges.MsgList[i].ToUserName
								wxSendMsg.ToUserName = wxRecvMsges.MsgList[i].FromUserName
								wxSendMsg.LocalID = fmt.Sprintf("%d", time.Now().Unix())
								wxSendMsg.ClientMsgId = wxSendMsg.LocalID

								time.Sleep(time.Second)
								go s.SendMsg(&loginMap, wxSendMsg)
							}
						}
					} else if strings.Contains(wxRecvMsges.MsgList[i].Content, regAt) && strings.Contains(cont, "好感度") {
						tmpId := tmp[0]
						userId := strings.Replace(tmpId, ":", "", -1)
						userId = strings.Replace(userId, "<br/>", "", -1)
						var content1 string
						var content2 string
						var user m.User
						var pass bool
						for _, v := range contactMap {
							if v.UserName == userId {
								user = v
								break
							}
						}
						if !CheckRedis(user.RemarkName) {
							content1 = fmt.Sprintf("@%s 不好意思哟,未找到你的信息,请先签到呢", user.NickName)
						} else {
							userDat := GetRedis(user.RemarkName)
							js, err := simplejson.NewJson([]byte(userDat))
							panicErr(err)
							friendliNess := js.Get("Friendliness").MustInt()
							content2 = fmt.Sprintf("@%s 魔理沙酱对你的好感度已经有 %d 呦,请继续加油哒", user.NickName, friendliNess)
							pass = true
						}
						var content string
						if pass {
							content = content2
						} else {
							content = content1
						}
						wxSendMsg := m.WxSendMsg{}
						wxSendMsg.Type = 1
						wxSendMsg.Content = content
						wxSendMsg.FromUserName = wxRecvMsges.MsgList[i].ToUserName
						wxSendMsg.ToUserName = wxRecvMsges.MsgList[i].FromUserName
						wxSendMsg.LocalID = fmt.Sprintf("%d", time.Now().Unix())
						wxSendMsg.ClientMsgId = wxSendMsg.LocalID
						time.Sleep(time.Second)
						go s.SendMsg(&loginMap, wxSendMsg)
					}  else if strings.Contains(wxRecvMsges.MsgList[i].Content, regAt) && strings.Contains(cont, "经验值") ||
						strings.Contains(wxRecvMsges.MsgList[i].ToUserName, "@@") && strings.Contains(cont, "我的经验值") {
						tmpId := tmp[0]
						userId := strings.Replace(tmpId, ":", "", -1)
						userId = strings.Replace(userId, "<br/>", "", -1)
						var content1 string
						var content2 string
						var user m.User
						var pass bool
						if len(tmp) > 1 {
						for _, v := range contactMap {
							if v.UserName == userId {
								user = v
								break
							}
						}
					}else{
						user.RemarkName = "何朝阳"
					}
						if !CheckRedis(user.RemarkName) {
							content1 = fmt.Sprintf("@%s 不好意思哟,未找到你的信息,请先签到呢", user.NickName)
						} else {
							userDat := GetRedis(user.RemarkName)
							js, err := simplejson.NewJson([]byte(userDat))
							panicErr(err)
							exp := js.Get("exp").MustInt()
							if user.RemarkName == "何朝阳" {
								content2 = fmt.Sprintf("@%s 你的经验值已经有 %d点了呢", "嗨，阳光", exp)
							} else {
								content2 = fmt.Sprintf("@%s 你的经验值已经有 %d点了呢", user.NickName, exp)
							}
							topContent := []string{"哇!你是本群最能水的呢", "你好强哦", "无敌总是寂寞的,看来你已经很寂寞了呢", "恭喜，你以后就是老大了"}
							isTop := FindMaxExp(user.RemarkName, exp)
							if isTop {
								randIndex := rand.Intn(3)
								content2 = fmt.Sprintf("%s,%s", content2, topContent[randIndex])
							}
							pass = true
						}
						var content string
						if pass {
							content = content2
						} else {
							content = content1
						}
						wxSendMsg := m.WxSendMsg{}
						wxSendMsg.Type = 1
						wxSendMsg.Content = content
						if user.RemarkName == "何朝阳" {
							wxSendMsg.FromUserName = wxRecvMsges.MsgList[i].FromUserName
							wxSendMsg.ToUserName = wxRecvMsges.MsgList[i].ToUserName
						} else {
							wxSendMsg.FromUserName = wxRecvMsges.MsgList[i].ToUserName
							wxSendMsg.ToUserName = wxRecvMsges.MsgList[i].FromUserName
						}
						wxSendMsg.LocalID = fmt.Sprintf("%d", time.Now().Unix())
						wxSendMsg.ClientMsgId = wxSendMsg.LocalID
						time.Sleep(time.Second)
						go s.SendMsg(&loginMap, wxSendMsg)
					}else if regGroup.MatchString(wxRecvMsges.MsgList[i].FromUserName) && strings.Contains(wxRecvMsges.MsgList[i].Content, regAt) {
						if len(tmp) > 1 {
							reply = GetReply(tmp[1])
						}
						wxSendMsg := m.WxSendMsg{}
						wxSendMsg.Type = 1
						wxSendMsg.Content = reply
						wxSendMsg.FromUserName = wxRecvMsges.MsgList[i].ToUserName
						wxSendMsg.ToUserName = wxRecvMsges.MsgList[i].FromUserName
						wxSendMsg.LocalID = fmt.Sprintf("%d", time.Now().Unix())
						wxSendMsg.ClientMsgId = wxSendMsg.LocalID

						/* 加点延时，避免消息次序混乱，同时避免微信侦察到机器人 */
						time.Sleep(time.Second)

						go s.SendMsg(&loginMap, wxSendMsg)
					}
				}
				if wxRecvMsges.MsgList[i].MsgType == 1 || wxRecvMsges.MsgList[i].MsgType == 3 || wxRecvMsges.MsgList[i].MsgType == 47 {
					regGroup := "@@"
					if strings.Contains(wxRecvMsges.MsgList[i].ToUserName, regGroup) {
						var user m.User
						regAt := "@"
						tmp := strings.Split(wxRecvMsges.MsgList[i].Content, regAt)
						tmpId := tmp[0]
						userId := strings.Replace(tmpId, ":", "", -1)
						userId = strings.Replace(userId, "<br/>", "", -1)
						if len(tmp) > 1 {
							for _, v := range contactMap {
								if v.UserName == userId {
									user = v
									break
								}
							}
						} else {
							user.RemarkName = "何朝阳"
							if tmpId == "我要签到" {
								Sign(user, wxRecvMsges.MsgList[i].ToUserName, wxRecvMsges.MsgList[i].FromUserName)
							}
						}
						if !CheckRedis(user.RemarkName) {
							content1 := fmt.Sprintf("@%s 不好意思哟,未找到你的信息,请先签到呢", user.NickName)
							wxSendMsg := m.WxSendMsg{}
							wxSendMsg.Type = 1
							wxSendMsg.Content = content1
							wxSendMsg.FromUserName = wxRecvMsges.MsgList[i].ToUserName
							wxSendMsg.ToUserName = wxRecvMsges.MsgList[i].FromUserName
							wxSendMsg.LocalID = fmt.Sprintf("%d", time.Now().Unix())
							wxSendMsg.ClientMsgId = wxSendMsg.LocalID
							time.Sleep(time.Second)
							go s.SendMsg(&loginMap, wxSendMsg)
						} else {
							userDat := GetRedis(user.RemarkName)
							js, err := simplejson.NewJson([]byte(userDat))
							panicErr(err)
							var userTmp userCache
							userTmp.Friendliness = js.Get("Friendliness").MustInt()
							userTmp.city = user.City
							userTmp.signCount = js.Get("signCount").MustInt()
							userTmp.signTime = js.Get("signTime").MustString()
							userTmp.userName = user.NickName
							userTmp.exp = js.Get("exp").MustInt() + 1
							js.Set("exp", userTmp.exp)
							unjs, err := js.MarshalJSON()
							panicErr(err)
							SetRedis(user.RemarkName, string(unjs))
							//			content2 = fmt.Sprintf("@%s 魔理沙酱对你的好感度已经有 %d 呦,请继续加油哒", user.NickName, friendliNess)
						}
					}
				}
			}
		}
	}
}

//GetWeather 获取天气消息
func GetWeather(city string) (msg string) {
	client := &http.Client{}
	var cityCode string
	if _, ok := cityDict[city]; ok {
		cityCode = cityDict[city]
	} else {
		cityCode = "101020100"
	}
	url := fmt.Sprintf("http://t.weather.sojson.com/api/weather/city/%s", cityCode)
	reqest, err := http.NewRequest("GET", url, nil)
	panicErr(err)

	//处理返回结果
	response, _ := client.Do(reqest)

	//返回的状态码
	status := response.StatusCode
	weatherJSON, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(status)
		panic(err)
	}

	//获取json格式页面数据
	js, err := simplejson.NewJson(weatherJSON)
	panicErr(err)

	now := time.Now()
	//今日天气
	todayWeather := js.Get("data").Get("forecast").GetIndex(0)
	//天气类型
	weatherType := todayWeather.Get("type").MustString()
	weatherType = fmt.Sprintf("天气 : %s", weatherType)
	//时间
	todayTime := now.Format("2006年01月02日 15:04:05")
	//天气注意事项
	notice := todayWeather.Get("notice").MustString()
	//温度
	high := todayWeather.Get("high").MustString()
	highSplit := strings.Split(high, " ")
	low := todayWeather.Get("low").MustString()
	lowSplit := strings.Split(low, " ")
	temperature := fmt.Sprintf("温度: %s/%s", lowSplit[1], highSplit[1])
	//风
	fx := todayWeather.Get("fx").MustString()
	fl := todayWeather.Get("fl").MustString()
	wind := fmt.Sprintf("%s : %s", fx, fl)
	//空气指数
	quality := js.Get("data").Get("quality").MustString()
	weatherQuality := fmt.Sprintf("空气质量 : %s", quality)
	if city == "" {
		city = "未知地区"
	}
	cityName := fmt.Sprintf("地区：%s", city)
	lastMsg := fmt.Sprintf("%s\n%s\n%s, %s。\n%s\n%s\n%s\n", todayTime, cityName, weatherType, notice, temperature, wind, weatherQuality)
	return lastMsg
}

func GetReply(msg string) (reply string) {
	// dat := []byte(datJSON)
	// jsDat, err := simplejson.NewJson(dat)
	// panicErr(err)
	var con conf
	// conf := con.getConf()
	// apiKey := conf.ApiKey
	// datAPIKey := jsDat.Get("userInfo")
	// datAPIKey.Set("apiKey", apiKey)
	// datText := jsDat.Get("perception").Get("inputText")
	// datText.Set("text", msg)
	// dat, err = jsDat.MarshalJSON()
	// panicErr(err)
	// url := "http://openapi.tuling123.com/openapi/api/v2"
	// reqest, err := http.Post(url, 	, bytes.NewBuffer(dat))
	// panicErr(err)
	// retJSON, err := ioutil.ReadAll(reqest.Body)
	// panicErr(err)
	// js, err := simplejson.NewJson(retJSON)
	// panicErr(err)
	// ret := js.Get("results").GetIndex(0)
	// text := ret.Get("values").Get("text").MustString()
	conf := con.getConf()
	apiKey := conf.ApiKey
	client := &http.Client{}
	var url = fmt.Sprintf("http://www.tuling123.com/openapi/api?key=%s&info=%s", apiKey, msg)
	reqest, err := http.NewRequest("GET", url, nil)
	panicErr(err)

	//处理返回结果
	response, _ := client.Do(reqest)

	//返回的状态码
	status := response.StatusCode
	jsdat, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(status)
		panic(err)
	}

	//获取json格式页面数据
	js, err := simplejson.NewJson(jsdat)
	panicErr(err)
	text := js.Get("text").MustString()
	return text
}

func main() {
	rand.Seed(time.Now().UnixNano())
	GetChat()
	//	LinkRedis()
	//  var reply = GetReply("")
	//   println(reply)
}

func Sign(user m.User, toUserName string, fromUserName string) {
	if user.RemarkName != "" {
		pass := false
		var userTmp userCache
		if !CheckRedis(user.RemarkName) {
			dat := []byte(datJSON)
			jsDat, err := simplejson.NewJson(dat)
			panicErr(err)
			jsDat.Set("userName", user.NickName)
			jsDat.Set("city", "上海")
			todayTime := time.Now().Format("2006-01-02 15:04:05")
			jsDat.Set("signTime", todayTime)
			jsDat.Set("signCount", 1)
			friendliNessAdd := rand.Intn(30)
			friendliNess := friendliNessAdd
			jsDat.Set("Friendliness", friendliNess)
			userTmp.userName = user.NickName
			userTmp.city = user.City
			userTmp.signTime = todayTime
			userTmp.signCount = 1
			userTmp.Friendliness = friendliNess
			userTmp.FriendlinessAdd = friendliNessAdd
			unjs, err := jsDat.MarshalJSON()
			panicErr(err)
			SetRedis(user.RemarkName, string(unjs))
			pass = true
		} else {
			userDat := GetRedis(user.RemarkName)
			js, err := simplejson.NewJson([]byte(userDat))
			panicErr(err)
			signTimeTmp := js.Get("signTime").MustString()
			signTime, _ := time.ParseInLocation("2006-01-02 15:04:05", signTimeTmp, time.Local)
			if signTime.Day() != time.Now().Day() {
				pass = true
			} else if signTime.Month() != time.Now().Month() {
				pass = true
			}
			if pass {
				signCount := js.Get("signCount").MustInt()
				friendliNess := js.Get("Friendliness").MustInt()
				friendliNessAdd := rand.Intn(30)
				friendliNess += friendliNessAdd
				js.Set("Friendliness", friendliNess)
				js.Set("userName", user.NickName)
				js.Set("city", user.City)
				todayTime := time.Now().Format("2006-01-02 15:04:05")
				js.Set("signTime", todayTime)
				signCount = signCount + 1
				js.Set("signCount", signCount)
				userTmp.userName = user.NickName
				userTmp.city = "上海"
				userTmp.signTime = todayTime
				userTmp.signCount = signCount
				userTmp.Friendliness = friendliNess
				userTmp.FriendlinessAdd = friendliNessAdd
				unjs, err := js.MarshalJSON()
				panicErr(err)
				SetRedis(user.RemarkName, string(unjs))
			}
		}
		var contents string
		content1 := fmt.Sprintf("@%s 签到成功！今天已经是你的第%d次签到啦～魔理沙酱对乃的好感度[+%d]2019年，要加油哦~", "嗨，阳光", userTmp.signCount, userTmp.FriendlinessAdd)
		content2 := fmt.Sprintf("@%s 真是健忘呐！！今天签到过了哟！", "嗨，阳光")
		if pass {
			contents = content1
		} else {
			contents = content2
		}
		wxSendMsg := m.WxSendMsg{}
		wxSendMsg.Type = 1
		wxSendMsg.Content = contents
		wxSendMsg.FromUserName = fromUserName
		wxSendMsg.ToUserName = toUserName
		wxSendMsg.LocalID = fmt.Sprintf("%d", time.Now().Unix())
		wxSendMsg.ClientMsgId = wxSendMsg.LocalID

		/* 加点延时，避免消息次序混乱，同时避免微信侦察到机器人 */
		time.Sleep(time.Second)

		go s.SendMsg(&loginMap, wxSendMsg)
	}
}
