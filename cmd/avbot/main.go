package main

import (
	"flag"
	"io/ioutil"

	"github.com/go-yaml/yaml"
	"github.com/hyqhyq3/avbot-telegram"
	"github.com/hyqhyq3/avbot-telegram/github-webhook"
	"github.com/hyqhyq3/avbot-telegram/hello"
	"github.com/hyqhyq3/avbot-telegram/irc"
	"github.com/hyqhyq3/avbot-telegram/joke"
	"github.com/hyqhyq3/avbot-telegram/stat"
	"github.com/hyqhyq3/avbot-telegram/ws"
)

type Config struct {
	Secret string

	Github struct {
		Listen string
	}

	WebSocket struct {
		Port int
	}

	Proxy struct {
		Socks5 string
	}
}

var config Config

func init() {

	var configFile string
	flag.StringVar(&configFile, "c", "avbot.yaml", "config file")
	flag.Parse()
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	yaml.Unmarshal(data, &config)

}

func main() {

	token := config.Secret
	bot := avbot.NewBot(token, config.Proxy.Socks5)
	bot.AddMessageHook(irc.New(bot.GetBotApi(), "#avplayer", "avbot-tg"))
	bot.AddMessageHook(ws.New(bot.GetBotApi(), token, config.WebSocket.Port))
	bot.AddMessageHook(joke.New())
	bot.AddMessageHook(hello.New(`
@{{.UserName}}({{.FirstName}}) 你好,欢迎你加入本群.请在十分钟内回答以下问题: (直接回答到本群聊天里,不要回复给机器人) 
1. 你从事什么工作?喜欢什么语言 
2. 怎么看待 C 和 C++ 
3. 从哪里听说本群的? 加入的主要目的是什么 
4. 能否 Show 一段代码展示一下？ 
另外请遵守如下的规则  
* 请仔细阅读新人须知　http://avplayer.org/newbeefaq.html  
* avplayer 社区建立了维基站点 https://wiki.avplayer.org , 有问题先到维基查找相关信息. 如果 wiki 上没有, 你获得了群友的回答,麻烦到维基上建立相关条目, 以方便后来人. wiki 公开编辑权限, 无需注册.  
* 问一些需要时间思考的问题请到论坛里发帖, 特别是 Boost 相关的问题. https://www.avboost.com.  
* 长期潜水的人都会被强制清理. 伸手党会被立即清理. 十分钟内没有回答的人会被管理员请出群, 请不要无视机器人的通告. 管理员都是疯子,做好被虐待的准备,大胆发言.`))
	bot.AddMessageHook(stat.New("stat.dat"))

	bot.AddMessageHook(github.New(bot.GetBotApi(), -1001041314546, config.Github.Listen))
	bot.Run()
}
