package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

var cmdList *CommandList

func init() {
	cmdList = &CommandList{
		commands: make(map[string]Command),
	}
	f := Command{
		Name:        "coinflip",
		Description: "Flip a Coin - 'coinflip'",
		Usage:       "coinflip",
		Run:         coinFlip,
	}
	cmdList.AddCommand(f)
	c := Command{
		Name:        "stock",
		Description: "Get a Stock Quote - 'quote AAPL'",
		Usage:       "quote TICKER",
		Run:         getQuote,
	}
	cmdList.AddCommand(c)

	k := Command{
		Name:        "kudos",
		Description: "Send kudos to a teammate- 'quote @teammate'",
		Usage:       "kudos @teammate",
		Run:         kudos,
	}
	cmdList.AddCommand(k)

	i := Command{
		Name:        "image",
		Description: "Returns the first google image for query- 'image <query>'",
		Usage:       "image kittens",
		Run:         image,
	}
	cmdList.AddCommand(i)
}

type Command struct {
	Name        string
	Description string
	Usage       string
	Run         CommandFunc
}

type CommandFunc func(args []string) string

type CommandList struct {
	commands map[string]Command
}

func (cl *CommandList) AddCommand(c Command) {
	cl.commands[c.Name] = c
}

func (cl *CommandList) Process(ws *websocket.Conn, m Message, id string) {
	logger.Info("processing", m.Text)

	if strings.HasPrefix(m.Text, "<@"+id+">") {

		parts := strings.Fields(m.Text)

		cmd, ok := cl.commands[parts[1]]
		if !ok {
			logger.Info("error", "no command found",
				"args", parts[1],
				"full text", m.Text)
			m.Text = cl.ListCommands()
			postMessage(ws, m)
			return
		}
		// looks good, get the quote and reply with the result
		logger.Info("action", "start processing",
			"args", parts[1],
			"full text", m.Text)
		go func(m Message) {
			logger.Info("action", "executing",
				"full text", m.Text)
			m.Text = cmd.Run(parts[2:])
			postMessage(ws, m)
		}(m)
	} else {
		// casual mention.  What should we do about that?
		go func(m Message) {
			m.Text = "You rang?"
			postMessage(ws, m)
		}(m)

	}

}

func (cl *CommandList) ListCommands() string {
	out := "Here's what I can do:\n"
	for _, cmd := range cl.commands {
		txt := fmt.Sprintf("\t %s - %s\n", cmd.Name, cmd.Description)
		out = out + txt
	}

	return out

}

// send a kudo to a team member
func kudos(args []string) string {
	if len(args) < 1 {
		return "Please tell me who to thank!"
	}
	teammate := args[0]
	return fmt.Sprintf("Hey %s, thanks for being awesome!", teammate)
}

// send a kudo to a team member
func image(args []string) string {
	if len(args) < 1 {
		return "You don't really want me searching for random images, do you?"
	}
	url := fmt.Sprintf("https://ajax.googleapis.com/ajax/services/search/images?v=1.0&q=%s", url.QueryEscape(strings.Join(args, " ")))
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("Google doesn't like you. %s", err)
	}
	defer resp.Body.Close()
	r := make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return fmt.Sprintf("Google doesn't like you. %s", err)
	}
	rd := r["responseData"].(map[string]interface{})
	res := rd["results"].([]interface{})
	con := res[0].(map[string]interface{})
	u := con["url"].(string)
	return fmt.Sprintf("%v", u)
}

// Get the quote via Yahoo. You should replace this method to something
// relevant to your team!
func getQuote(args []string) string {
	if len(args) < 1 {
		return "Please tell me which stock to quote next time!"
	}
	sym := args[0]

	sym = strings.ToUpper(sym)
	url := fmt.Sprintf("http://download.finance.yahoo.com/d/quotes.csv?s=%s&f=nsl1op&e=.csv", sym)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	rows, err := csv.NewReader(resp.Body).ReadAll()
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	if len(rows) >= 1 && len(rows[0]) == 5 {
		return fmt.Sprintf("%s (%s) is trading at $%s", rows[0][0], rows[0][1], rows[0][2])
	}
	return fmt.Sprintf("unknown response format (symbol was \"%s\")", sym)
}

// Get the quote via Yahoo. You should replace this method to something
// relevant to your team!
func coinFlip(args []string) string {

	var heads bool
	rand.Seed(time.Now().UnixNano())
	switch rand.Intn(2) {
	case 0:
		heads = true
	case 1:
		heads = false
	}
	if heads {
		return fmt.Sprintf("the gods of fortune say 'heads'")
	}
	return fmt.Sprintf("'tails' is the result")

}
