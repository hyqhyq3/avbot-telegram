package irc

import (
	"fmt"
	"testing"
	"time"

	"gopkg.in/sorcix/irc.v1"
)

func TestIrc(t *testing.T) {
	c, err := irc.Dial("chat.freenode.net:6667")
	fmt.Println(err)

	go func() {
		for {
			fmt.Println(c.Decode())
		}
	}()
	err=c.Encode(&irc.Message{Command: irc.PASS, Params: []string{"hyq"}})
	fmt.Println(err)
	err=c.Encode(&irc.Message{Command: irc.NICK, Params: []string{"hyq"}})
	fmt.Println(err)
	err=c.Encode(&irc.Message{Command: irc.USER, Params: []string{"guest 0 * :hyq"}})
	fmt.Println(err)
	err=c.Encode(&irc.Message{Command: irc.JOIN, Params: []string{"#avplayer"}})
	fmt.Println(err)
	for i := 0; i < 10; i++ {
		<-time.After(time.Second )
		err=c.Encode(&irc.Message{Command: irc.PRIVMSG, Params: []string{"#avplayer", "test"}})
		fmt.Println(err)
	}

	<-time.After(time.Second * 10)
	t.Fail()
}
