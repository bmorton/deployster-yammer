package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	deploy "github.com/bmorton/deployctl/client"
	"github.com/bmorton/go-yammer/cometd"
	"github.com/bmorton/go-yammer/schema"
	"github.com/bmorton/go-yammer/yammer"
)

func main() {
	client := yammer.New(os.Getenv("YAMMER_TOKEN"))
	d := deploy.New(os.Getenv("DEPLOYSTER_URL"), os.Getenv("DEPLOYSTER_USERNAME"), os.Getenv("DEPLOYSTER_PASSWORD"), "")

	groupId, _ := strconv.Atoi(os.Getenv("YAMMER_GROUP"))
	feed, err := client.GroupFeed(groupId)
	if err != nil {
		log.Println(err)
		return
	}

	rt := cometd.New(feed.Meta.Realtime.URI, feed.Meta.Realtime.AuthenticationToken)
	err = rt.Handshake()
	if err != nil {
		log.Println(err)
		return
	}

	rt.Subscribe(feed.Meta.Realtime.ChannelId)
	messageChan := make(chan *schema.MessageFeed, 10)
	stopChan := make(chan bool, 1)

	log.Printf("Polling group ID: %d\n", groupId)
	go rt.Poll(messageChan, stopChan)
	for {
		select {
		case m := <-messageChan:
			message := m.Messages[0]
			args := strings.Split(message.Body.Plain, " ")
			if message.IsThreadStarter() && len(args) > 2 && args[0] == "!deploy" {
				log.Printf("[!deploy] Triggering deploy for %s (version: %s)", args[1], args[2])
				client.PostMessage(&yammer.CreateMessageParams{
					Body:        fmt.Sprintf("Triggering deploy for %s (version: %s)", args[1], args[2]),
					RepliedToId: message.Id,
					GroupId:     groupId,
				})
				d.CreateDeploy(args[1], args[2], true, 0)
			}
		}
	}
}
