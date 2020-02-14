package slack

import (
	"context"
	"log"

	slackbinding "github.com/mattmoor/bindings/pkg/slack"
	"github.com/nlopes/slack"

	"github.com/mattmoor/knobots/pkg/handler"
)

type dm struct{}

var _ handler.Interface = (*dm)(nil)

func New(context.Context) handler.Interface {
	return &dm{}
}

func (*dm) GetType() interface{} {
	return &DirectMessage{}
}

func (*dm) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	log.Printf("Got event: %v", x)
	dm := x.(*DirectMessage)

	api, err := slackbinding.New(ctx)
	if err != nil {
		log.Printf("error creating slack client: %v", err)
		return nil, err
	}

	if len(dm.Emails) != 1 {
		log.Printf("TOO MANY EMAILS: %v", dm)
		return nil, nil
	}
	email := dm.Emails[0]

	user, err := api.GetUserByEmail(email)
	if err != nil {
		log.Printf("error looking up user: %v", err)
		return nil, err
	}

	_, _, channelID, err := api.OpenIMChannel(user.ID)
	if err != nil {
		log.Printf("error opening IM channel: %v", err)
		return nil, err
	}

	for _, line := range dm.Message {
		_, _, err = api.PostMessage(channelID, slack.MsgOptionText(line, false))
		if err != nil {
			log.Printf("error posting message: %v", err)
			return nil, err
		}
	}

	log.Print("Sent message")
	return nil, nil
}

type DirectMessage struct {
	// The email addresses to which we send a message.
	Emails []string `json:"emails"`

	Message []string `json:"message"`
	// TODO(mattmoor): Determine the contents.
}

var _ handler.Response = (*DirectMessage)(nil)

func (*DirectMessage) GetSource() string {
	return "https://github.com/mattmoor/knobots/cmd/slack"
}

func (*DirectMessage) GetType() string {
	return "dev.mattmoor.knobots.slack.direct"
}
