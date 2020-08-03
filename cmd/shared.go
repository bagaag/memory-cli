/*
This file is part of the software application Memory
See https://github.com/bagaag/memory
Copyright © 2020 Matt Wiseley
License: https://www.gnu.org/licenses/gpl-3.0.txt
*/

/* This file contains functions that are used throughout the cmd package. */

package cmd

import (
	"fmt"
	"memory/app"
	"memory/app/config"
	"memory/app/persist"
	"memory/util"
	"os"
	"os/exec"
	"strings"

	"github.com/chzyer/readline"
)

// filterInput allows certain keys to be intercepted during readline
func filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

// parseTypes populates an EntryType struct based on the --types flag
func parseTypes(typesArg []string) app.EntryTypes {
	types := app.EntryTypes{}
	for _, t := range typesArg {
		switch strings.TrimSpace(strings.ToLower(t)) {
		case "note", "notes":
			types.Note = true
		case "event", "events":
			types.Event = true
		case "person", "people":
			types.Person = true
		case "place", "places":
			types.Place = true
		case "thing", "things":
			types.Thing = true
		}
	}
	return types
}

// getLinkedEntry returns the entry and an 'exists' boolean at the
// given index of an array that combines the LinksTo and LinkedFrom
// slices of the given entry.
func getLinkedEntry(entry app.Entry, ix int) (app.Entry, bool) {
	a := append(entry.LinksTo, entry.LinkedFrom...)
	name := a[ix]
	entry, exists := app.GetEntry(name)
	if !exists {
		fmt.Printf("There is no entry named '%s'.\n", name)
	}
	return entry, exists
}

// editEntry converts an entry to YamlDown, launches an external editor, parses
// the edited content back into an entry and returns the edited entry.
func editEntry(origEntry app.Entry) (app.Entry, error) {
	initial, err := app.RenderYamlDown(origEntry)
	if err != nil {
		return app.Entry{}, err
	}
	edited, err := useEditor(initial)
	if err != nil {
		return app.Entry{}, err
	}
	editedEntry, err := app.ParseYamlDown(edited)
	if err != nil {
		return app.Entry{}, err
	}
	if origEntry.Name != editedEntry.Name {
		app.DeleteEntry(origEntry.Name)
		//TODO: update links on rename
	}
	app.PutEntry(editedEntry)
	app.Save()
	return editedEntry, nil
}

// useEditor launches config.editor with a temporary file containing the given string
// waits for the editor to exit and returns a string with the updated content.
func useEditor(s string) (string, error) {
	tmp, err := persist.CreateTempFile(s)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %s", err.Error())
	}
	cmd := exec.Command(config.EditorCommand, tmp)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to interact with editor: %s", err.Error())
	}
	var edited string
	if edited, err = persist.ReadTempFile(tmp); err != nil {
		return "", fmt.Errorf("failed to read temporary file: %s", err.Error())
	}
	if err := persist.RemoveTempFile(tmp); err != nil {
		return edited, fmt.Errorf("failed to delete temporary file: %s", err.Error())
	}
	return edited, nil
}

// Displays prompt for single character input and returns the character entered, or empty string.
func getSingleCharInput() string {
	fmt.Print(config.SubPrompt)
	ascii, _, err := util.ReadKeyStroke()
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}
	s := string(rune(ascii))
	if ascii == 3 { // Ctrl+C
		s = "^C"
	} else if ascii == 13 { // Enter
		s = ""
	}
	fmt.Println(s)
	return s
}

// subPrompt asks for additional info within a command.
func subPrompt(prompt string, value string, validate validator) (string, error) {
	rl.HistoryDisable()
	rl.SetPrompt(prompt)
	var err error
	var input = value
	for {
		input, err = rl.ReadlineWithDefault(input)
		if err != nil {
			break
		}
		if msg := validate(input); msg != "" {
			fmt.Println(msg)
		} else {
			break
		}
	}
	rl.HistoryEnable()
	rl.SetPrompt(config.Prompt)
	return strings.TrimSpace(input), err
}
