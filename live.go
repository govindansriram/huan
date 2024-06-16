package main

import (
	"agent/chrome"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

/*
Here you will find functions used to live sessions, along with helper functions tp help indicate processes completed running
when making wrapper libraries
*/

/*
processOperations

processes commands delivered to the agent
*/
func processOperations(
	filePath string,
	ctx context.Context,
	sessionPath string,
	waitTime *uint64) error {

	// read the command file
	byteSlice, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	_, fname := filepath.Split(filePath)
	nameWoExt := strings.Split(fname, ".")[0]
	filePath = filepath.Join(sessionPath, "responses", nameWoExt)

	// create a director for the command
	if err = os.Mkdir(filePath, 0777); err != nil {
		return err
	}

	op := &Operation{}

	err = json.Unmarshal(byteSlice, op)
	if err != nil {
		return err
	}

	job := chrome.InitFileJob()

	var responseErr error // error that signals the command failed

	/*
		response error will not be returned by this function, but any error that is, will automatically trigger the death
		of the session
	*/

	/**
	TODO: LLM operations, integrate tool calls, it should be able to extract info from the response json
	add task for post processing loading
	*/

	switch op.Type {
	case "browser":
		lastCommand := op.CommandList[len(op.CommandList)-1]
		action, err := chrome.AddOperation(lastCommand.Params, lastCommand.CommandName, filePath, job)
		if err != nil {
			// if the action does not exist shut down the session
			return err
		}
		responseErr = performAction(ctx, action, job, waitTime)
	case "llm":
		//waitSeconds := *waitTime / 1000
		if *waitTime > 32767 {
			return errors.New("command wait time exceeds limit for LLM'S the limit is 32767 seconds")
		}

		llmWaitTime := int32(*waitTime)
		op.Settings.Timeout = &llmWaitTime
		responseErr = runLlmCommands(op.Settings, op.CommandList, filePath)
	case "exit":
		// an exit command was received so the session must end
		return errors.New("session has manually exited")
	}

	if responseErr != nil {
		err = writeErr(filePath, responseErr)
	} else {
		err = writeSuccess(filePath)
	}

	// if success or command errors cannot be written return that error

	if err != nil {
		return err
	}

	return nil
}
