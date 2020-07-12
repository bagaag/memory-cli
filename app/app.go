/*
This file is part of the software application Memory
See https://github.com/bagaag/memory
Copyright © 2020 Matt Wiseley
License: https://www.gnu.org/licenses/gpl-3.0.txt
*/

//Package app contains an API for interacting with the application
//that is not bound to a particular UI.
package app

import (
	"errors"
	"memory/app/config"
	"memory/app/model"
	"memory/app/persist"
	"sort"
	"strings"
)

// EntryTypes is used to indicate one or more entry types in a single argument
type EntryTypes struct {
	Note   bool
	Event  bool
	Person bool
	Place  bool
	Thing  bool
}

// EntryResults is used to contain the results of GetEntries and the settings used
// to generate those results.
type EntryResults struct {
	Entries    []model.Entry
	Types      EntryTypes
	StartsWith string
	Contains   string
	Search     string
	Tags       []string
	Sort       SortOrder
	Limit      int
}

// HasAll returns true if either all are true or all are false.
func (t EntryTypes) HasAll() bool {
	if (t.Note && t.Event && t.Person && t.Place && t.Thing) ||
		(!t.Note && !t.Event && !t.Person && !t.Place && !t.Thing) {
		return true
	}
	return false
}

// String returns a string representation of the selected types.
func (t EntryTypes) String() string {
	s := "All types"
	if !t.HasAll() {
		a := []string{}
		if t.Note {
			a = append(a, "Notes")
		}
		if t.Event {
			a = append(a, "Events")
		}
		if t.Person {
			a = append(a, "People")
		}
		if t.Place {
			a = append(a, "Places")
		}
		if t.Thing {
			a = append(a, "Things")
		}
		s = strings.Join(a, ", ")
	}
	return s
}

// root contains all the data to be saved
type root struct {
	Notes []model.Note
	//Tags  map[string]model.Tag
}

// SortOrder is used to indicate one of the Sort constants
type SortOrder int

// SortRecent sorts entries by descending modified date
const SortRecent = SortOrder(0)

// SortName sorts entries alphabetically by name
const SortName = SortOrder(1)

// The data variable stores all the things that get saved.
var data = root{}

// Init reads data stored on the file system
// and initializes application variable.
func Init() error {

	if persist.PathExists(config.SavePath()) {
		if err := persist.Load(config.SavePath(), &data); err != nil {
			return err
		}
	}

	return nil
}

// Save writes application data to file storage.
func Save() error {
	return persist.Save(config.SavePath(), data)
}

// filterStartsWith filters a list of entries based on a name prefix.
func filterStartsWith(entries []model.Entry, startsWith string) []model.Entry {
	if startsWith == "" {
		return entries
	}
	startsWith = strings.ToLower(startsWith)
	ret := []model.Entry{}
	for _, e := range entries {
		entry := e.(model.Entry)
		if strings.HasPrefix(strings.ToLower(entry.Name()), startsWith) {
			ret = append(ret, entry)
		}
	}
	return ret
}

// filterContains filters a list of entries based on substring matches.
func filterContains(entries []model.Entry, contains string) []model.Entry {
	if contains == "" {
		return entries
	}
	contains = strings.ToLower(contains)
	ret := []model.Entry{}
	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry.Name()), contains) {
			ret = append(ret, entry)
		}
	}
	return ret
}

// tagMatches returns true if any of the tags in searchTags match the tags
// on the provided Entry.
func tagMatches(entry *model.Entry, searchTags []string) bool {
	for _, searchTag := range searchTags {
		for _, tag := range (*entry).Tags() {
			if tag == searchTag {
				return true
			}
		}
	}
	return false
}

// filterTags returns the subset of Entries in an array that have any of the
// tags specified.
func filterTags(entries []model.Entry, searchTags []string) []model.Entry {
	if len(searchTags) == 0 {
		return entries
	}
	// convert a copy of searchTags to lower case
	searchTagsLower := make([]string, len(searchTags))
	copy(searchTagsLower, searchTags)
	for i, searchTag := range searchTagsLower {
		searchTagsLower[i] = strings.ToLower(searchTag)
	}
	// filter array to those with matching tags
	ret := []model.Entry{}
	for _, entry := range entries {
		if tagMatches(&entry, searchTagsLower) {
			ret = append(ret, entry)
		}
	}
	return ret
}

func filterSearch(entries []model.Entry, keywords string) []model.Entry {
	//TODO: Implement https://blevesearch.com/
	return entries
}

func sortEntries(arr []model.Entry, field string, ascending bool) {
	var less func(i, j int) bool
	switch field {
	case "Modified":
		less = func(i, j int) bool {
			if ascending {
				return arr[i].Modified().UnixNano() < arr[j].Modified().UnixNano()
			} else {
				return arr[i].Modified().UnixNano() > arr[j].Modified().UnixNano()
			}
		}
	default: // Name
		less = func(i, j int) bool {
			if ascending {
				return arr[i].Name() < arr[j].Name()
			} else {
				return arr[i].Name() > arr[j].Name()
			}
		}
	}
	sort.Slice(arr, less)
}

// GetEntries returns an array of entries of the specified type(s) with
// specified filters and sorting applied.
func GetEntries(types EntryTypes, startsWith string, contains string,
	search string, tags []string, sort SortOrder, limit int) EntryResults {

	// collect and filter entries
	entriesArrays := make([][]model.Entry, 5)
	//TODO: Make these cases run concurrently
	allTypes := types.HasAll()
	if allTypes || types.Event {
	}
	if allTypes || types.Note {
		notes := data.Notes
		noteEntries := make([]model.Entry, len(notes))
		for i, note := range notes {
			noteEntries[i] = note
		}
		noteEntries = filterStartsWith(noteEntries, startsWith)
		noteEntries = filterContains(noteEntries, contains)
		noteEntries = filterTags(noteEntries, tags)
		noteEntries = filterSearch(noteEntries, search)
		entriesArrays[1] = noteEntries
	}
	if allTypes || types.Person {
	}
	if allTypes || types.Place {
	}
	if allTypes || types.Thing {
	}
	// combine filtered entries
	remainingCount := 0
	for _, entries := range entriesArrays {
		remainingCount += len(entries)
	}
	remainingEntries := make([]model.Entry, remainingCount)
	ix := 0
	for _, entries := range entriesArrays {
		for _, entry := range entries {
			remainingEntries[ix] = entry
			ix++
		}
	}
	// sort combined entries
	if sort == SortName {
		sortEntries(remainingEntries, "Name", true)
	} else { // SortRecent
		sortEntries(remainingEntries, "Modified", false)
	}
	// limit sorted results
	if limit <= 0 {
		limit = 999
	}
	if len(remainingEntries) > limit {
		remainingEntries = remainingEntries[:limit]
	}
	return EntryResults{
		Entries:    remainingEntries,
		Types:      types,
		StartsWith: startsWith,
		Contains:   contains,
		Search:     search,
		Tags:       tags,
		Sort:       sort,
		Limit:      limit,
	}
}

// EntryCount returns the total number of entries under management.
func EntryCount() int {
	return len(data.Notes)
}

// GetEntry returns a single entry or throws an error.
func GetEntry(entryType string, entryName string) (model.Entry, error) {
	switch strings.ToLower(entryType) {
	case "note":
		return GetNote(entryName)
	case "event":
	case "person":
	case "place":
	case "thing":
	default:
		return nil, errors.New(entryType + " is not a valid entry type.")
	}
	return nil, nil
}
