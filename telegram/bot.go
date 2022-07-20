package telegram

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

type EventHandle[T any] struct {
	// QQClient?
	handlers []HandleFunc[T]
	eventMu  sync.RWMutex
}

type HandleFunc[T any] func(b *Bot, event T) error

func (handle *EventHandle[T]) Subscribe(handler HandleFunc[T]) {
	handle.eventMu.Lock()
	defer handle.eventMu.Unlock()
	// shrink the slice
	newHandlers := make([]HandleFunc[T], len(handle.handlers)+1)
	copy(newHandlers, handle.handlers)
	newHandlers[len(handle.handlers)] = handler
	handle.handlers = newHandlers
}

func (handle *EventHandle[T]) dispatch(b *Bot, event T, onErr func(err error)) {
	handle.eventMu.RLock()
	defer func() {
		handle.eventMu.RUnlock()
		if pan := recover(); pan != nil {
			fmt.Printf("event error: %v\n%s", pan, debug.Stack())
		}
	}()
	for _, handler := range handle.handlers {
		if err := handler(b, event); err != nil {
			log.Error(err)
			onErr(err)
		}
	}
}

func (handle *EventHandle[T]) reset() {
	handle.eventMu.Lock()
	defer handle.eventMu.Unlock()
	handle.handlers = nil
}

type Bot struct {
	bot      *tgbotapi.BotAPI
	commands map[string]Command
	help     string

	TeardownEvent      EventHandle[os.Signal]
	CallBackQueryEvent EventHandle[tgbotapi.CallbackQuery]
	UpdateEvent        EventHandle[tgbotapi.Update]
	ContactEvent       EventHandle[tgbotapi.Contact]
	LocationEvent      EventHandle[tgbotapi.Message]
	commandMatchEvent  map[string]*EventHandle[tgbotapi.Update]

	handleFromNow bool
	developerMode bool
	version       string
	client        *http.Client
	debug         bool
}

func (b *Bot) SetDebug(debug bool) {
	b.debug = debug
}

func (b *Bot) TGCommands() []tgbotapi.BotCommand {
	var cmd []tgbotapi.BotCommand
	for _, v := range b.commands {
		cmd = append(cmd, v.ID())
	}
	return cmd
}

func (b *Bot) ReplyTo(m tgbotapi.Message, msg string, o ...interface{}) {
	var text string
	if len(o) > 0 {
		text = fmt.Sprintf(msg, o...)
	} else {
		text = msg
	}
	_, _ = b.Bot().Send(tgbotapi.NewMessage(
		m.Chat.ID,
		text,
	))
}

func (b *Bot) Sendf(id int64, msg string, o ...interface{}) {
	_, _ = b.Bot().Send(tgbotapi.NewMessage(
		id,
		fmt.Sprintf(msg, o...),
	))
}

func (b *Bot) Debug() {
	b.Bot().Debug = true
}

func (b *Bot) Bot() *tgbotapi.BotAPI {
	return b.bot
}

func (b *Bot) RegisterCommand(commands ...Command) error {
	for _, command := range commands {
		b.commands[command.ID().Command] = command
		err := command.Serve(b)
		if err != nil {
			return err
		}
	}
	return b.Bot().SetMyCommands(b.TGCommands())
}

// Init should be called first in Listen
func (b *Bot) Init() error {
	// check network
	log.Println("check bot network")
	for _, k := range []string{
		"http_proxy",
		"https_proxy",
		"HTTP_PROXY",
		"HTTPS_PROXY",
	} {
		log.Printf("Network env %s: %s", k, os.Getenv(k))
	}
	for id, cmd := range b.commands {
		log.Printf("init command %s", id)
		cmd.Init()
	}
	return nil
}

// Listen starts the bot main server loop
// don't set timeout too large, as the
func (b *Bot) Listen(timeout int) {
	startListenDate := time.Now()
	// a timeout parameter is passed in url, so the server wait Timeout seconds at most, until the first update is available
	updates, _ := b.Bot().GetUpdatesChan(tgbotapi.UpdateConfig{
		Offset:  0,
		Limit:   0,
		Timeout: timeout,
	})

	// execute command teardown when interrupt by os
	c := make(chan os.Signal, 1)
	// handler keyboard interrupt, docker stop, etc
	// The main process inside the container will receive SIGTERM, and after a grace period, SIGKILL
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, os.Kill)
	go func() {
		for sig := range c {
			log.Printf("signal %v received, exiting gracefully...", sig)
			b.TeardownEvent.dispatch(b, sig, ignoreErr)
			b.Bot().StopReceivingUpdates()
			return
		}
	}()

	for update := range updates {
		if b.debug {
			log.Printf("update: %+v", update)
		}
		// send error string to user
		sendErr := func(err error) {
			var id int64
			switch {
			case update.Message != nil:
				id = update.Message.Chat.ID
			case update.CallbackQuery != nil:
				id = update.CallbackQuery.Message.Chat.ID
			}
			_, _ = b.Bot().Send(tgbotapi.NewMessage(id, err.Error()))
		}

		// mandate command
		if update.CallbackQuery != nil {
			b.CallBackQueryEvent.dispatch(b, *update.CallbackQuery, sendErr)
		}
		if update.Message != nil {
			updateTime := time.Unix(int64(update.Message.Date), 0)
			if b.handleFromNow && updateTime.Before(startListenDate) {
				log.Printf("update is too old, ignore %d", update.UpdateID)
				continue
			}
			switch {
			case update.Message.IsCommand():
				cmdString := update.Message.Command()
				if isSysCommand(cmdString) {
					sysCommand(b, update)
				}

				// dispatch to registered command
				if h, ok := b.commandMatchEvent[cmdString]; ok {
					if ok2, reason := b.authorize(cmdString, update); !ok2 {
						b.ReplyTo(*update.Message, fmt.Sprintf("Access denied, reason: %s", reason))
					} else {
						h.dispatch(b, update, sendErr)
					}
				}
			case update.Message.Location != nil:
				b.LocationEvent.dispatch(b, *update.Message, sendErr)
				b.LocationEvent.reset() // clear all actions on location message
			case update.Message.Contact != nil:
				b.ContactEvent.dispatch(b, *update.Message.Contact, sendErr)
			default:
				// other general logic, if any
				b.UpdateEvent.dispatch(b, update, sendErr)
			}
		}
	}
	log.Println("update exhausted, exit")
}

func ignoreErr(err error) {
	log.Error(err)
}

func (b *Bot) SetClient(client *http.Client) {
	b.client = client
	b.bot.Client = client
}

func (b *Bot) Match(c Command) *EventHandle[tgbotapi.Update] {
	e := new(EventHandle[tgbotapi.Update])
	b.commandMatchEvent[c.ID().Command] = e
	return e
}

func (b *Bot) authorize(cmd string, update tgbotapi.Update) (bool, string) {
	command, ok := b.commands[cmd]
	if !ok {
		return false, ""
	}
	return command.Authorize().Validate(update)
}

type BotWrapperConfig func(b *Bot)

type SimpleAuth struct {
	WhiteList     []int64 // 白名单里的ID只要匹配即通过
	DeveloperOnly bool
}

// Validate 鉴权
func (c SimpleAuth) Validate(u tgbotapi.Update) (ok bool, reason string) {
	ok = true
	if c.DeveloperOnly {
		ok = u.Message.From.ID == int(DeveloperChatID)
		if !ok {
			reason = "You are not developer, command open to developer only"
			return
		}
	}
	if len(c.WhiteList) > 0 {
		for _, wl := range c.WhiteList {
			if int64(u.Message.From.ID) == wl {
				return
			}
		}
		ok, reason = false, fmt.Sprintf("failed white list check, id: %d", u.Message.From.ID)
		return
	}
	return
}

func SetHelp(help string) BotWrapperConfig {
	return func(b *Bot) {
		b.help = help
	}
}

// SetHandleFromNow
// if set to true, ignore updates before the bot's Listen loop starts
func SetHandleFromNow(yes bool) BotWrapperConfig {
	return func(b *Bot) {
		b.handleFromNow = yes
	}
}

func setProxy(httpProxy *http.Transport) BotWrapperConfig {
	return func(b *Bot) {
		b.client.Transport = httpProxy
	}
}

func SetProxyFromURL(u *url.URL) BotWrapperConfig {
	return setProxy(&http.Transport{
		Proxy: http.ProxyURL(u),
	})
}

func SetNoProxy() BotWrapperConfig {
	return setProxy(nil)
}

func isSysCommand(cmd string) bool {
	return cmd == "start" || cmd == "help" || cmd == "stop" || cmd == "version"
}

func sysCommand(bw *Bot, u tgbotapi.Update) {
	basicSend := func(s string) {
		_, err := bw.Bot().Send(tgbotapi.NewMessage(u.Message.Chat.ID, s))
		if err != nil {
			log.Println(err)
		}
	}
	switch u.Message.Command() {
	case "start":
		var commands []string
		for _, cmd := range bw.TGCommands() {
			commands = append(commands, "/"+cmd.Command)
		}
		basicSend(fmt.Sprintf("Here are the available commands:\n%s", strings.Join(commands, "\n")))
	case "help":
		if bw.help != "" {
			basicSend(bw.help)
		} else {
			basicSend("the bot does not have help")
		}
	case "version":
		basicSend(bw.version)
	}
}

func proxyTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
}

func setUpBot(bot *tgbotapi.BotAPI, client *http.Client, configs ...BotWrapperConfig) *Bot {
	bw := &Bot{
		bot:               bot,
		commands:          make(map[string]Command),
		commandMatchEvent: make(map[string]*EventHandle[tgbotapi.Update]),
	}
	// keep a reference to the client
	bw.SetClient(client)
	for i := range configs {
		configs[i](bw)
	}
	return bw
}

func NewMessageBot(token string, configs ...BotWrapperConfig) (*Bot, error) {
	// use a proxy client, or you cannot get bot created
	client := &http.Client{
		Transport: proxyTransport(),
	}
	bot, err := tgbotapi.NewBotAPIWithClient(token, tgbotapi.APIEndpoint, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot api: %v", err)
	}
	return setUpBot(bot, client, configs...), nil
}

// NewMessageBotWithURLProxy use fixed url proxy, not reading system env variables
// this is useful when other part of the system is unhappy to have `http_proxy`
func NewMessageBotWithURLProxy(token string, proxy string, configs ...BotWrapperConfig) (*Bot, error) {
	u, err := url.Parse(proxy)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(u),
		},
	}
	bot, err := tgbotapi.NewBotAPIWithClient(token, tgbotapi.APIEndpoint, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot api: %v", err)
	}
	return setUpBot(bot, client, configs...), nil
}
