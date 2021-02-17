package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	discord "github.com/bwmarrin/discordgo"
	"github.com/ghodss/yaml"
)

// TODO: impl local storage for:
//         - tracking last seen data
//         - scoreboard
//       only track specific channels?
//       only respond in specific channels?
//       figure out how this data should even be structured...

type config struct {
	Token            string
	TrackedReactions struct {
		Types   map[string]string
		Ratings []string
	}
	CommandPrefix string
}

var cfg config

func readConf(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	return nil
}

type score struct {
	answers   map[string]int
	questions map[string]int
}

func (s score) String() string {
	var sb strings.Builder

	sb.WriteString("```❓")
	for _, rating := range cfg.TrackedReactions.Ratings {
		sb.WriteString(
			fmt.Sprintf("[%s:%d] ", rating, s.questions[rating]),
		)
	}
	sb.WriteString("\n")

	sb.WriteString("❗")
	for _, rating := range cfg.TrackedReactions.Ratings {
		sb.WriteString(
			fmt.Sprintf("[%s:%d] ", rating, s.answers[rating]),
		)
	}
	sb.WriteString("\n```")

	return sb.String()
}

type scoreBoard map[string]score

func main() {
	err := readConf("./config.yaml")
	if err != nil {
		log.Fatalln(err)
	}

	bot, err := discord.New("Bot " + cfg.Token)
	if err != nil {
		log.Fatalln(err)
	}

	scoreboard := make(scoreBoard)

	bot.AddHandler(scoreboard.messageCreate)
	bot.AddHandler(scoreboard.messageReaction)
	bot.Identify.Intents = discord.IntentsGuildMessages |
		discord.IntentsGuildMessageReactions

	if err := bot.Open(); err != nil {
		log.Fatalln(err)
	}
	defer bot.Close()

	log.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

// hook: Tracks ratings made by users that have already been reacted with the appropriate reaction type
func (scoreboard *scoreBoard) messageReaction(s *discord.Session, m *discord.MessageReactionAdd) {
	if m.UserID == s.State.User.ID {
		return
	}

	message, _ := s.ChannelMessage(m.ChannelID, m.MessageID)

	// we only care about messages that are tracked reaction types
	// eg. '!' = "answer", '?' = "question
	var messageType string
	for _, reaction := range message.Reactions {
		if mt, exists := cfg.TrackedReactions.Types[reaction.Emoji.Name]; exists {
			messageType = mt
			break
		}
	}

	if (*scoreboard)[message.Author.ID].answers == nil {
		(*scoreboard)[message.Author.ID] = score{
			make(map[string]int),
			make(map[string]int),
		}
	}
	log.Println(messageType)
	switch messageType {
	case "":
		return
	case "answer":
		(*scoreboard)[message.Author.ID].answers[m.Emoji.Name]++
		log.Println((*scoreboard)[message.Author.ID].answers[m.Emoji.Name])
	case "question":
		(*scoreboard)[message.Author.ID].questions[m.Emoji.Name]++
		log.Println((*scoreboard)[message.Author.ID].answers[m.Emoji.Name])
	}
}

// hook: generic message handling, only used for bot commands and debugging so far
func (scoreboard *scoreBoard) messageCreate(s *discord.Session, m *discord.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, cfg.CommandPrefix) {
		scoreboard.botCommand(s, m)
		return
	}

	channel, _ := s.Channel(m.ChannelID)
	channelName := channel.Name
	log.Printf("[%s] %s: %s\n", channelName, m.Author.Username, m.Content)
}

// bot command handler called from messageCreate hook
func (scoreboard scoreBoard) botCommand(s *discord.Session, m *discord.MessageCreate) {
	line := strings.Fields(m.Content)
	if len(line) < 2 {
		s.MessageReactionAdd(
			m.ChannelID,
			m.Message.ID,
			"⁉️",
		)
		return
	}

	command, args := line[1], line[2:]

	switch command {
	case "stats":
		if len(args) > 0 {
			// TODO get username from @mention, error handling
			s.ChannelMessageSendReply(
				m.ChannelID,
				fmt.Sprintf("%s", scoreboard[args[0]]),
				m.Message.Reference(),
			)
			return
		}
		s.ChannelMessageSendReply(
			m.ChannelID,
			fmt.Sprintf("%s", scoreboard[m.Author.ID]),
			m.Message.Reference(),
		)
	case "ping":
		s.MessageReactionAdd(
			m.ChannelID,
			m.Message.ID,
			"🆗",
		)
		s.ChannelMessageSendReply(
			m.ChannelID,
			"pong",
			m.Message.Reference(),
		)
	default:
		s.MessageReactionAdd(
			m.ChannelID,
			m.Message.ID,
			"⁉️",
		)
	}

}
