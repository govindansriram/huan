package main

import (
	"agent/chrome"
	"agent/llm"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type Settings struct {
	Timeout     *int16                   `json:"timeout"`
	Headless    bool                     `json:"headless"`
	MaxToken    *int                     `json:"max_tokens"`
	LLMSettings []map[string]interface{} `json:"llm_settings"`
	TryLimit    int16                    `json:"try_limit"`
}

type Command struct {
	CommandName string                 `json:"command_name,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
	MessageType string                 `json:"message_type"`
	Message     map[string]interface{} `json:"message"`
}

type Operation struct {
	Type        string    `json:"type"`
	Settings    Settings  `json:"settings"`
	CommandList []Command `json:"command_list"`
}

func createSessionDirectory(sessionId string) string {
	pth, exists := os.LookupEnv("BENCHAI-SAVEDIR")

	if exists {
		if _, err := os.Stat(pth); err != nil && os.IsNotExist(err) {
			log.Fatalf("directory %s does not exist", pth)
		} else if err != nil {
			log.Fatalf("cannot use directory %s as the save location basepath", pth)
		}
	} else {
		currentUser, err := user.Current()

		if err != nil {
			log.Fatal("was unable to extract the current os user")
		}

		pth = path.Join(currentUser.HomeDir, "/.cache/benchai/agent/")
	}

	pth = path.Join(pth, "sessions")

	if err := os.MkdirAll(pth, 0777); err != nil && !os.IsExist(err) {
		log.Fatalf("session directory at %s does not exist and cannot be created", pth)
	}

	pth = path.Join(pth, sessionId)

	if err := os.Mkdir(pth, 0777); err != nil && os.IsExist(err) {
		log.Fatalf("session: %s, already exists", pth)
	} else if err != nil {
		log.Fatalf("cannot use directory %s as the session save location", pth)
	}

	return pth
}

func runBrowserCommands(settings Settings, commandList []Command, sessionPath string) {
	paramSlice := make([]map[string]interface{}, len(commandList))
	nameSlice := make([]string, len(commandList))

	for index, val := range commandList {
		nameSlice[index] = val.CommandName
		paramSlice[index] = val.Params
	}

	chrome.RunSequentialCommands(settings.Headless, settings.Timeout, sessionPath, paramSlice, nameSlice)
}

// create an array of LLMs and calls exponential backoff on the array of messages built in addLlmOpperations
func runLlmCommands(settings Settings, commandList []Command, sessionPath string) error {

	messageTypeSlice := make([]string, len(commandList))
	messageSlice := make([]map[string]interface{}, len(commandList))
	modelSettingsSlice := make([]map[string]interface{}, len(settings.LLMSettings))

	for index, com := range commandList {
		messageTypeSlice[index] = com.MessageType
		messageSlice[index] = com.Message
	}

	copy(modelSettingsSlice, settings.LLMSettings)

	chat, err := llm.Execute(
		messageTypeSlice,
		messageSlice,
		modelSettingsSlice,
		settings.MaxToken,
		settings.TryLimit,
		settings.Timeout)

	if err != nil {
		return err
	}

	for _, sett := range settings.LLMSettings {
		delete(sett, "api_key")
	}

	type writeStruct struct {
		SettingsSlice []map[string]interface{} `json:"settings"`
		Completion    *llm.ChatCompletion      `json:"completion"`
		MessageList   []Command                `json:"message_list"`
	}

	msg := llm.ConvertChatCompletion(chat)
	commandList = append(commandList, Command{
		Message:     msg,
		MessageType: "assistant",
	})

	writeData := writeStruct{
		SettingsSlice: settings.LLMSettings,
		Completion:    chat,
		MessageList:   commandList,
	}

	b, err := json.MarshalIndent(writeData, "", "    ")

	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(sessionPath, "completion.json"), b, 0666)

	if err != nil {
		return err
	}

	return nil
}

type Configuration struct {
	Operations []Operation `json:"operations"`
	SessionId  string      `json:"session_id"`
}

type runner interface {
	init([]string) error
	run()
	getName() string
}

type liveCommand struct {
	fs              *flag.FlagSet
	headless        bool
	commandLifetime uint64
}

func (l *liveCommand) init(args []string) error {
	return l.fs.Parse(args)
}

func (l *liveCommand) getName() string {
	return l.fs.Name()
}

func newLiveCommand() *liveCommand {
	rc := liveCommand{
		fs: flag.NewFlagSet("live", flag.ExitOnError),
	}

	rc.fs.BoolVar(
		&rc.headless,
		"h",
		false,
		"run in headless")

	rc.fs.Uint64Var(
		&rc.commandLifetime,
		"life",
		0,
		"specify a lifetime in ms for how long commands can run for")

	return &rc
}

func (l *liveCommand) run() {
	if l.fs.Arg(0) == "" {
		log.Fatal("session cannot be empty")
	}

	sessionName := l.fs.Arg(0)

	timeout, err := strconv.ParseInt(l.fs.Arg(1), 10, 64)

	if err != nil {
		log.Fatalf("unable to parse browser lifetime value of %s, got error %v", l.fs.Arg(1), err)
	}

	var clt *uint64

	if l.commandLifetime > 0 {
		clt = &l.commandLifetime
	}

	pth := createSessionDirectory(sessionName)

	RunLive(uint64(timeout), l.headless, clt, pth)
}

type runCommand struct {
	fs *flag.FlagSet
}

func (r *runCommand) init(args []string) error {
	return r.fs.Parse(args)
}

func (r *runCommand) getName() string {
	return r.fs.Name()
}

type sessionCommand struct {
	fs *flag.FlagSet
	rf bool
}

func (s *sessionCommand) init(args []string) error {
	return s.fs.Parse(args)
}

func (s *sessionCommand) getName() string {
	return s.fs.Name()
}

func (s *sessionCommand) run() {

	if s.fs.Arg(0) == "ls" && s.rf {
		log.Fatal("cannot use the list flag and the rf flag together. They are unrelated")
	}

	if s.fs.Arg(0) == "ls" && s.fs.NArg() > 1 {
		log.Fatalf("no arguments can follow past the list flag")
	}

	pth, exists := os.LookupEnv("BENCHAI-SAVEDIR")

	if !exists {
		currentUser, err := user.Current()

		if err != nil {
			log.Fatalf("failed to find current os user")
		}

		pth = path.Join(currentUser.HomeDir, "/.cache/benchai/agent/")
	}

	pth = filepath.Join(pth, "sessions")

	if s.fs.Arg(0) == "ls" {

		if _, err := os.Stat(pth); err == nil {
			dirEntry, err := os.ReadDir(pth)

			if err != nil {
				log.Fatal(err)
			}

			dirList := "["
			for _, f := range dirEntry {
				if f.IsDir() {
					dirList += f.Name() + ", "
				}
			}

			dirList = strings.TrimSuffix(dirList, ", ")

			dirList += "]"
			fmt.Println(dirList)
			return
		} else if os.IsNotExist(err) {
			fmt.Println("[]")
			return
		} else {
			log.Fatalf("error finding directory %s", pth)
		}
	}

	if s.fs.Arg(0) == "rm" {

		if s.fs.Arg(1) == "" && !s.rf {
			log.Fatal("no session was specified to delete")
		} else if s.fs.Arg(1) != "" && s.rf {
			log.Fatalf("rf can not be followed by any sessions")
		} else if !s.rf {
			sessionPath := filepath.Join(pth, s.fs.Arg(1))
			if _, err := os.Stat(sessionPath); err == nil {
				err = os.RemoveAll(sessionPath)

				if err != nil {
					log.Fatalf("unable to delete dir %s", s.fs.Arg(1))
				}
			} else if os.IsNotExist(err) {
				log.Fatalf("session %s can not be removed as it does not exist", s.fs.Arg(1))
			} else {
				log.Fatalf("unable to locate session %s", s.fs.Arg(1))
			}
		} else {
			dirEntry, err := os.ReadDir(pth)

			if err != nil {
				log.Fatal(err)
			}

			for _, entry := range dirEntry {
				err = os.RemoveAll(filepath.Join(pth, entry.Name()))
				if err != nil {
					log.Fatalf("failed to remove session, %s", entry.Name())
				}
			}
		}
	}
}

func newSessionCommand() *sessionCommand {
	rc := sessionCommand{
		fs: flag.NewFlagSet("session", flag.ExitOnError),
	}

	rc.fs.BoolVar(
		&rc.rf,
		"rf",
		false,
		"removes all sessions")

	return &rc
}

// run
/**
The run command, checks if the user wishes to run their chrome in headless mode, and whether they are pointing to
a file or passing raw json
*/
func (r *runCommand) run() {

	configString := r.fs.Arg(0)

	if configString == "" {
		log.Fatal("invalid config argument")
	}

	var bytes []byte
	var err error

	bytes, err = os.ReadFile(configString)

	if err != nil {
		log.Fatalf("failed to read json file due to: %v", err)
	}

	var config Configuration

	err = json.Unmarshal(bytes, &config)

	if err != nil {
		log.Fatalf("failed to decode json file: %v", err)
	}

	pth := createSessionDirectory(config.SessionId)

	for _, op := range config.Operations {
		switch op.Type {
		case "browser":
			runBrowserCommands(op.Settings, op.CommandList, pth)
		case "llm":
			err = runLlmCommands(op.Settings, op.CommandList, pth)
			log.Fatal(err)
		default:
			log.Fatalf("unknown operation type: %s", op.Type)
		}
	}
}

func newRunCommand() *runCommand {
	rc := runCommand{
		fs: flag.NewFlagSet("run", flag.ExitOnError),
	}

	return &rc
}

type versionCommand struct {
	fs *flag.FlagSet
}

func (v *versionCommand) init(args []string) error {
	return v.fs.Parse(args)
}

func (v *versionCommand) run() {
	fmt.Println("Version 0.0.0")
}

func (v *versionCommand) getName() string {
	return v.fs.Name()
}

func newVersionCommand() *versionCommand {
	vc := versionCommand{
		fs: flag.NewFlagSet("version", flag.ExitOnError),
	}

	return &vc
}

// root
/*
Checks for present subcommands and executes them
*/
func root(args []string) error {
	if len(args) < 1 {
		return errors.New("no command passed")
	}

	cmds := []runner{
		newRunCommand(),
		newVersionCommand(),
		newSessionCommand(),
		newLiveCommand(),
	}

	subcommand := os.Args[1]

	for _, cmd := range cmds {
		if cmd.getName() == subcommand {
			if err := cmd.init(os.Args[2:]); err == nil {
				cmd.run()
				return nil
			} else {
				return err
			}
		}
	}

	return fmt.Errorf("unknown command: %s", subcommand)
}

func main() {
	if err := root(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
